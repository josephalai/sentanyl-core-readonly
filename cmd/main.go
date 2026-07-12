package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/josephalai/sentanyl/core-service/hydrator"
	"github.com/josephalai/sentanyl/core-service/internal/sidecar"
	"github.com/josephalai/sentanyl/core-service/routes"
	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/config"
	"github.com/josephalai/sentanyl/pkg/db"
	httputil "github.com/josephalai/sentanyl/pkg/http"
	"github.com/josephalai/sentanyl/pkg/storage"
)

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
	go h.Start()

	// Wire the same storage provider into the context-pack render endpoints
	// so "render a Saved Context to a Digital Download PDF" uses the same
	// GCS bucket + auth as the cert/funnel hydrator. nil propagates the same
	// 503 fail-closed semantics applied above.
	routes.SetContextRenderStorage(gcsProvider, gcsBucket)

	// Set up Gin router.
	r := gin.Default()
	r.Use(httputil.CORSMiddleware())

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
		// The Settings "Reset All Data" button posts to /reset-all-data;
		// keep this aligned with the frontend contract at
		// frontend/src/pages/settings/SettingsPage.tsx:402.
		tenantAPI.DELETE("/reset-all-data", routes.HandleTenantResetAllData)

		// Platform billing (charging the tenant for Sentanyl itself).
		tenantAPI.GET("/billing", routes.HandleGetBillingStatus)
		tenantAPI.GET("/billing/plans", routes.HandleListBillingPlans)
		tenantAPI.POST("/billing/checkout-session", routes.HandleCreateBillingCheckoutSession)
		tenantAPI.POST("/billing/change-plan", routes.HandleChangeBillingPlan)
		tenantAPI.POST("/billing/portal-session", routes.HandleCreateBillingPortalSession)

		// Machine API key (tenant send API + MCP) — self-serve mint/rotate/revoke.
		tenantAPI.GET("/settings/api-key", routes.HandleGetTenantAPIKey)
		tenantAPI.POST("/settings/api-key", routes.HandleMintTenantAPIKey)
		tenantAPI.DELETE("/settings/api-key", routes.HandleRevokeTenantAPIKey)

		// Stripe Connect OAuth initiate + disconnect.
		tenantAPI.GET("/stripe/connect", routes.HandleStripeConnectInitiate)
		tenantAPI.DELETE("/stripe/connect", routes.HandleStripeConnectDisconnect)
	}

	// Everything else on the tenant dashboard is GATED on the platform
	// subscription (trial/active/past_due-in-grace pass; expired/canceled 402).
	tenantGated := r.Group("/api/tenant")
	tenantGated.Use(auth.RequireTenantAuth(), auth.RequirePlatformSubscription())
	{
		// Tenant custom domains
		tenantGated.POST("/domains", routes.HandleAddTenantDomain)
		tenantGated.GET("/domains", routes.HandleListTenantDomains)
		tenantGated.DELETE("/domains/:id", routes.HandleDeleteTenantDomain)
		tenantGated.POST("/domains/:id/verify", routes.HandleVerifyTenantDomain)
		tenantGated.POST("/domains/adopt", routes.HandleAdoptTenantDomain)

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

	// Public endpoint: end-user/subscriber registration (no tenant JWT).
	r.POST("/api/register/user", routes.HandleRegisterUser)

	// Sending domain management — moved to tenantAPI above under /sending-domain.
	// Legacy bare paths kept as aliases so existing DNS records/bookmarks still work.
	r.POST("/api/domain", routes.HandleAddDomain)
	r.GET("/api/domains", routes.HandleGetDomains)
	r.GET("/api/domain/:domainId", routes.HandleGetDomain)
	r.DELETE("/api/domain/:domainId", routes.HandleDeleteDomain)
	r.POST("/api/domain/:domainId/verify-dns", routes.HandleVerifyDNS)
	r.POST("/api/domain/:domainId/test-send", routes.HandleTestSend)
	r.GET("/api/domain/:domainId/test-send-status", routes.HandleGetTestSendStatus)
	r.GET("/api/domain/:domainId/stats", routes.HandleGetDomainStats)
	r.GET("/api/domain/:domainId/reputation", routes.HandleGetDomainReputation)
	r.GET("/api/domain/:domainId/warming", routes.HandleGetDomainWarming)
	r.GET("/api/domain/:domainId/bounces", routes.HandleGetDomainBounces)
	r.POST("/api/domain/:domainId/pause", routes.HandlePauseDomain)
	r.POST("/api/domain/:domainId/resume", routes.HandleResumeDomain)

	// Script compiler (SentanylScript DSL).
	routes.RegisterScriptRoutes(r)

	// Story execution engine — internal endpoint + scheduler goroutine.
	routes.RegisterStoryEngineRoutes(r)
	routes.StartStoryScheduler()

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
