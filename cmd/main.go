package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/josephalai/sentanyl/core-service/hydrator"
	"github.com/josephalai/sentanyl/core-service/internal/sidecar"
	"github.com/josephalai/sentanyl/core-service/routes"
	"github.com/josephalai/sentanyl/pkg/audit"
	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/badges"
	"github.com/josephalai/sentanyl/pkg/config"
	"github.com/josephalai/sentanyl/pkg/db"
	httputil "github.com/josephalai/sentanyl/pkg/http"
	"github.com/josephalai/sentanyl/pkg/jobs"
	"github.com/josephalai/sentanyl/pkg/plans"
	"github.com/josephalai/sentanyl/pkg/publicchannel"
	"github.com/josephalai/sentanyl/pkg/storage"
)

var publicLegacySunset = time.Date(2027, 7, 15, 0, 0, 0, 0, time.UTC)

func main() {
	log.Println("core-service: starting up")

	// Load config from .env if present.
	if _, err := os.Stat(".env"); err == nil {
		configVals := config.LoadConfigFile(config.ConfigFile)
		config.MapConfigValues(configVals)
	}

	// Determine port (default 8081 for core-service).
	port := os.Getenv("CORE_SERVICE_PORT")
	if port == "" {
		port = "8081"
	}

	// Connect to MongoDB.
	db.MongoHost = envOrDefault("MONGO_HOST", "localhost")
	db.MongoPort = envOrDefault("MONGO_PORT", "27017")
	db.MongoDB = envOrDefault("MONGO_DB", "sentanyl")
	db.MongoDefaultCollectionName = "creators"
	db.UsingLocalMongo = true
	db.InitMongoConnection()

	// Ensure per-tenant compound unique indexes on the story-builder graph so
	// public IDs cannot collide across tenants (ID-004).
	routes.EnsureStoryGraphIndexes()

	// Ensure the platform ProviderEvent unique index (BILL-002 idempotency).
	routes.EnsurePlatformWebhookIndexes()

	// Retire any pre-hashing plaintext reset tokens at rest (ID-015).
	routes.RetirePlaintextResetTokens()
	routes.EnsureIdentityIndexes()
	auth.EnsureWorkspaceIndexes()
	routes.EnsureMachineCredentialIndexes()

	// ID-012: badge-assignment provenance idempotency invariant.
	badges.EnsureIndexes()

	// COM-EM-009: story-enrollment command-key invariant.
	routes.EnsureStorySessionIndexes()

	// Set up the service bridge for cross-service communication.
	lmsURL := envOrDefault("LMS_SERVICE_URL", "http://localhost:8082")
	marketingURL := envOrDefault("MARKETING_SERVICE_URL", "http://localhost:8083")
	bridge := routes.NewServiceBridge(lmsURL, marketingURL)
	routes.SetScriptBridge(bridge)

	// PowerMTA deliverability sidecar. When POWERMTA_SIDECAR_URL is unset
	// the client returns ErrSidecarUnconfigured for every method, and the
	// route handlers translate that into HTTP 503 instead of fake success.
	routes.SetSidecarClient(sidecar.New())

	// Start the hydrator worker. Cert + funnel PDFs are uploaded to GCS via
	// the shared storage provider; if GCS init fails (missing creds in dev),
	// the hydrator runs but PDF jobs surface a clear "GCS not configured"
	// error instead of silently writing to ephemeral disk.
	gcsBucket := envOrDefault("GCS_BUCKET", "sendhero-videos")
	gcsProject := envOrDefault("GCP_PROJECT_ID", "sendhero")
	var gcsProvider storage.StorageProvider
	if p, err := storage.NewGCSProvider(gcsProject); err != nil {
		log.Printf("[Core] GCS provider init failed (PDF rendering will fail until configured): %v", err)
	} else {
		gcsProvider = p
		defer p.Close()
	}
	h := hydrator.New(bridge, gcsProvider, gcsBucket)
	// Durable hydration sweep (OPS-001): registration + bootstrap enqueue —
	// the jobs worker started below executes the passes. Indexes first so the
	// bootstrap enqueue lands on the unique (type, idempotency_key) key.
	jobs.EnsureIndexes()
	h.Start()

	// Wire the same storage provider into the context-pack render endpoints
	// so "render a Saved Context to a Digital Download PDF" uses the same
	// GCS bucket + auth as the cert/funnel hydrator. nil propagates the same
	// 503 fail-closed semantics applied above.
	routes.SetContextRenderStorage(gcsProvider, gcsBucket)

	// OPS-005: platform audit ledger — indexes, durable-write fallback job,
	// and the retention sweep (core-service owns the single daily sweep).
	audit.Init("core-service")
	audit.StartRetentionSweep()

	// Set up Gin router.
	r := gin.Default()
	r.Use(httputil.CORSMiddleware())
	r.Use(audit.Middleware())

	r.GET("/health", httputil.HealthHandler("core-service"))

	// E2E-mode synchronous hydrate trigger. Lets the puppeteer harness skip
	// the 30s ticker so the same request that issues a cert can assert on
	// asset_url after a single retry. 403 in production.
	if os.Getenv("SENTANYL_E2E_MODE") == "1" {
		r.POST("/internal/test/hydrate-certs", func(c *gin.Context) {
			h.RunCertsNow()
			c.JSON(200, gin.H{"status": "ok"})
		})
		// Billing-state fixture for lifecycle flow 16: sets subscription
		// status / trial / past-due timestamps directly so enforcement can be
		// exercised without live Stripe. Never registered in production.
		r.POST("/internal/test/set-billing", routes.HandleTestSetBilling)
		r.POST("/internal/test/reconcile-billing", routes.HandleTestReconcileBilling)
		// Story-scheduler fast-forward for lifecycle/product flows: rewinds
		// active story_sessions' sent_at and runs one synchronous scheduler
		// pass so multi-day waits can be walked in seconds.
		r.POST("/internal/test/tick-stories", routes.HandleTestTickStories)
	}

	// Public auth routes (no JWT required). Rate-limited per IP: auth endpoints
	// throttle credential-stuffing/brute-force; request-reset is tighter because
	// each call sends an email (email-bomb prevention).
	authLimit := httputil.RateLimit(30, 15)
	emailLimit := httputil.RateLimit(6, 6)
	r.POST("/api/tenant/register", authLimit, routes.HandleTenantRegister)
	r.POST("/api/tenant/login", authLimit, routes.HandleTenantLogin)
	r.POST("/api/tenant/invitations/accept", authLimit, routes.HandleAcceptWorkspaceInvitation)
	r.POST("/api/customer/login", authLimit, routes.HandleCustomerLogin)
	r.POST("/api/customer/set-password", authLimit, routes.HandleCustomerSetPassword)
	r.POST("/api/customer/request-reset", emailLimit, routes.HandleCustomerRequestReset)

	// Stripe Connect OAuth callback (public — auth is via the state token).
	r.GET("/api/tenant/stripe/oauth/callback", routes.HandleStripeConnectCallback)

	// Platform billing webhook (public — auth is the Stripe signature).
	r.POST("/api/platform/stripe/webhook", routes.HandlePlatformStripeWebhook)

	// Protected tenant routes (require JWT). This UNGATED group carries the
	// routes an unpaid tenant must still reach — profile, settings, and the
	// billing endpoints themselves — so they can always get back to paying.
	tenantAPI := r.Group("/api/tenant")
	tenantAPI.Use(auth.RequireTenantAuth())
	{
		tenantAPI.GET("/profile", routes.HandleGetTenantProfile)
		tenantAPI.PUT("/settings", routes.HandleUpdateTenantSettings)
		// Session revocation (ID-005): current-token logout + all-device logout.
		tenantAPI.POST("/logout", routes.HandleTenantLogout)
		tenantAPI.POST("/logout-all", routes.HandleTenantLogoutAll)
		// Human identity ↔ workspace membership lifecycle (ID-009).
		tenantAPI.GET("/workspace/members", auth.RequirePermission(auth.PermSettingsManage), routes.HandleListWorkspaceMembers)
		tenantAPI.POST("/workspace/invitations", auth.RequireOwner(), routes.HandleInviteWorkspaceMember)
		tenantAPI.DELETE("/workspace/invitations/:id", auth.RequireOwner(), routes.HandleRevokeWorkspaceInvitation)
		tenantAPI.PUT("/workspace/members/:id", auth.RequireOwner(), routes.HandleUpdateWorkspaceMember)
		tenantAPI.POST("/workspace/transfer-ownership", auth.RequireOwner(), routes.HandleTransferWorkspaceOwnership)
		tenantAPI.POST("/workspace/select/:tenantId", routes.HandleSelectWorkspace)
		// The Settings "Reset All Data" button posts to /reset-all-data;
		// keep this aligned with the frontend contract at
		// frontend/src/pages/settings/SettingsPage.tsx:402.
		// Destructive reset is owner-only (ID-001).
		tenantAPI.DELETE("/reset-all-data", auth.RequirePermission(auth.PermDataDestroy), routes.HandleTenantResetAllData)

		// Operator job console (OPS-002) — owner-gated dead-letter read + replay.
		tenantAPI.GET("/ops/audit", auth.RequirePermission(auth.PermDataDestroy), routes.HandleOpsAuditList)
		tenantAPI.GET("/ops/jobs/overview", auth.RequirePermission(auth.PermDataDestroy), routes.HandleOpsJobOverview)
		tenantAPI.GET("/ops/jobs/dead", auth.RequirePermission(auth.PermDataDestroy), routes.HandleOpsDeadLetters)
		tenantAPI.POST("/ops/jobs/:id/replay", auth.RequirePermission(auth.PermDataDestroy), routes.HandleOpsReplayJob)

		// Platform billing (charging the tenant for Sentanyl itself). Reads are
		// available to any authenticated account so an unpaid tenant can always
		// see status; mutations require owner-level billing authority (ID-001).
		tenantAPI.GET("/billing", routes.HandleGetBillingStatus)
		tenantAPI.GET("/billing/invoices", routes.HandleGetBillingInvoices)
		tenantAPI.GET("/billing/plans", routes.HandleListBillingPlans)
		tenantAPI.POST("/billing/checkout-session", auth.RequirePermission(auth.PermBillingManage), routes.HandleCreateBillingCheckoutSession)
		tenantAPI.POST("/billing/change-plan", auth.RequirePermission(auth.PermBillingManage), routes.HandleChangeBillingPlan)
		tenantAPI.POST("/billing/schedule-cancel", auth.RequirePermission(auth.PermBillingManage), routes.HandleScheduleBillingCancellation)
		tenantAPI.POST("/billing/reactivate", auth.RequirePermission(auth.PermBillingManage), routes.HandleReactivateBillingSubscription)
		tenantAPI.POST("/billing/cancel-scheduled-change", auth.RequirePermission(auth.PermBillingManage), routes.HandleCancelScheduledBillingChange)
		tenantAPI.POST("/billing/portal-session", auth.RequirePermission(auth.PermBillingManage), routes.HandleCreateBillingPortalSession)

		// Machine API key (tenant send API + MCP) — owner-only secret management.
		tenantAPI.GET("/settings/api-key", auth.RequirePermission(auth.PermSecretsManage), routes.HandleGetTenantAPIKey)
		tenantAPI.POST("/settings/api-key", auth.RequirePermission(auth.PermSecretsManage), routes.HandleMintTenantAPIKey)
		tenantAPI.DELETE("/settings/api-key", auth.RequirePermission(auth.PermSecretsManage), routes.HandleRevokeTenantAPIKey)
		tenantAPI.PUT("/settings/api-key/scopes", auth.RequirePermission(auth.PermSecretsManage), routes.HandleUpdateTenantAPIKeyScopes)
		tenantAPI.GET("/settings/api-key/tools", auth.RequirePermission(auth.PermSecretsManage), routes.HandleListTenantAPIKeyTools)
		tenantAPI.GET("/settings/machine-credentials", auth.RequirePermission(auth.PermSecretsManage), routes.HandleListMachineCredentials)
		tenantAPI.POST("/settings/machine-credentials", auth.RequirePermission(auth.PermSecretsManage), routes.HandleCreateMachineCredential)
		tenantAPI.PUT("/settings/machine-credentials/:id", auth.RequirePermission(auth.PermSecretsManage), routes.HandleUpdateMachineCredential)
		tenantAPI.POST("/settings/machine-credentials/:id/rotate", auth.RequirePermission(auth.PermSecretsManage), routes.HandleRotateMachineCredential)
		tenantAPI.DELETE("/settings/machine-credentials/:id", auth.RequirePermission(auth.PermSecretsManage), routes.HandleRevokeMachineCredential)

		// Stripe Connect OAuth initiate + disconnect — owner-only secret mgmt.
		tenantAPI.GET("/stripe/connect", auth.RequirePermission(auth.PermSecretsManage), routes.HandleStripeConnectInitiate)
		tenantAPI.DELETE("/stripe/connect", auth.RequirePermission(auth.PermSecretsManage), routes.HandleStripeConnectDisconnect)
	}

	// Everything else on the tenant dashboard is GATED on the platform
	// subscription (trial/active/past_due-in-grace pass; expired/canceled 402).
	tenantGated := r.Group("/api/tenant")
	tenantGated.Use(auth.RequireTenantAuth(), auth.RequirePlatformSubscription())
	{
		// Tenant custom domains. Reads open to any account; mutations require
		// domain-management authority (owner/admin) (ID-001).
		tenantGated.POST("/domains", auth.RequirePermission(auth.PermDomainManage), routes.HandleAddTenantDomain)
		tenantGated.GET("/domains", routes.HandleListTenantDomains)
		tenantGated.DELETE("/domains/:id", auth.RequirePermission(auth.PermDomainManage), routes.HandleDeleteTenantDomain)
		tenantGated.POST("/domains/:id/verify", auth.RequirePermission(auth.PermDomainManage), routes.HandleVerifyTenantDomain)
		tenantGated.POST("/domains/adopt", auth.RequirePermission(auth.PermDomainManage), routes.HandleAdoptTenantDomain)

		// Context packs, brand profile, attribute schema
		routes.RegisterContextPackRoutes(tenantGated)
		routes.RegisterContextPackRenderRoutes(tenantGated)

		// Sending domain management (JWT-authenticated). Handlers read tenant
		// identity from the JWT context and never accept legacy subscriber_id.
		tenantGated.POST("/sending-domain", routes.HandleAddTenantSendingDomain)
		tenantGated.GET("/sending-domains", routes.HandleListTenantSendingDomains)
		tenantGated.GET("/sending-domain/:domainId", routes.HandleGetTenantSendingDomain)
		tenantGated.DELETE("/sending-domain/:domainId", routes.HandleDeleteTenantSendingDomain)
		tenantGated.POST("/sending-domain/:domainId/verify-dns", routes.HandleVerifyTenantSendingDomainDNS)
		tenantGated.POST("/sending-domain/:domainId/test-send", routes.HandleTenantSendingDomainTestSend)
		tenantGated.GET("/sending-domain/:domainId/test-send-status", routes.HandleGetTenantSendingDomainTestSendStatus)
		tenantGated.GET("/sending-domain/:domainId/stats", routes.HandleGetTenantSendingDomainStats)
		tenantGated.GET("/sending-domain/:domainId/reputation", routes.HandleGetTenantSendingDomainReputation)
		tenantGated.GET("/sending-domain/:domainId/warming", routes.HandleGetTenantSendingDomainWarming)
		tenantGated.GET("/sending-domain/:domainId/bounces", routes.HandleGetTenantSendingDomainBounces)
		tenantGated.POST("/sending-domain/:domainId/pause", routes.HandlePauseTenantSendingDomain)
		tenantGated.POST("/sending-domain/:domainId/resume", routes.HandleResumeTenantSendingDomain)

		// Story builder — stories, storylines, enactments, scenes, messages,
		// triggers, actions, badges, tags, users, email lists, stats, etc.
		routes.RegisterStoryRoutes(tenantGated)
	}

	// Public endpoint: end-user/subscriber registration. The legacy alias is
	// available only for the published compatibility window.
	r.POST("/api/register/user", httputil.LegacyAliasSunset(publicLegacySunset), routes.HandleRegisterUser)
	r.POST("/api/v1/public/register", publicchannel.RequireSignedContext(), httputil.Idempotency(), routes.HandleRegisterUser)

	// Sending domain management lives on tenantAPI under /sending-domain*.
	// The unauthenticated legacy /api/domain* aliases (subscriber_id query
	// param, no JWT) were removed in the phase-4 security sweep.

	// Script compiler (SentanylScript DSL).
	routes.RegisterScriptRoutes(r)

	// Story execution engine — internal endpoint + durable sweep job (W3-B).
	// The jobs worker claims only types registered in THIS process, so the
	// shared jobs collection routes story sweeps here and webhook deliveries
	// to marketing-service.
	routes.RegisterStoryEngineRoutes(r)
	jobs.EnsureIndexes()
	auth.EnsureSessionIndexes()
	auth.EnsurePrincipalIndexes()
	routes.EnsurePlanIntentIndexes()
	plans.EnsureUsageIndexes()
	routes.RegisterPlanIntentSweep()
	routes.RegisterStoryJobs()
	go jobs.RunWorker(context.Background(), jobs.WorkerConfig{Name: "core-" + auth.ServiceName("worker")})

	// Email click tracking.
	routes.RegisterTrackingRoutes(r)

	log.Printf("core-service: listening on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("core-service: failed to start: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
