package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/josephalai/sentanyl/core-service/hydrator"
	"github.com/josephalai/sentanyl/core-service/routes"
	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/config"
	"github.com/josephalai/sentanyl/pkg/db"
	httputil "github.com/josephalai/sentanyl/pkg/http"
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

	// Start the hydrator worker (stub for now).
	h := hydrator.New(bridge)
	go h.Start()

	// Set up Gin router.
	r := gin.Default()
	r.Use(httputil.CORSMiddleware())

	// Public auth routes (no JWT required).
	r.POST("/api/tenant/register", routes.HandleTenantRegister)
	r.POST("/api/tenant/login", routes.HandleTenantLogin)
	r.POST("/api/customer/login", routes.HandleCustomerLogin)

	// Protected tenant routes (require JWT).
	tenantAPI := r.Group("/api/tenant")
	tenantAPI.Use(auth.RequireTenantAuth())
	{
		tenantAPI.GET("/profile", routes.HandleGetTenantProfile)
		tenantAPI.PUT("/settings", routes.HandleUpdateTenantSettings)
		tenantAPI.DELETE("/reset", routes.HandleTenantResetAllData)

		// Tenant custom domains
		tenantAPI.POST("/domains", routes.HandleAddTenantDomain)
		tenantAPI.GET("/domains", routes.HandleListTenantDomains)
		tenantAPI.DELETE("/domains/:id", routes.HandleDeleteTenantDomain)
		tenantAPI.POST("/domains/:id/verify", routes.HandleVerifyTenantDomain)

		// Context packs, brand profile, attribute schema
		routes.RegisterContextPackRoutes(tenantAPI)

		// Sending domain management (JWT-authenticated, /sending-domain prefix avoids conflict with tenant vanity /domains).
		tenantAPI.POST("/sending-domain", routes.HandleAddDomain)
		tenantAPI.GET("/sending-domains", routes.HandleGetDomains)
		tenantAPI.GET("/sending-domain/:domainId", routes.HandleGetDomain)
		tenantAPI.DELETE("/sending-domain/:domainId", routes.HandleDeleteDomain)
		tenantAPI.POST("/sending-domain/:domainId/verify-dns", routes.HandleVerifyDNS)
		tenantAPI.POST("/sending-domain/:domainId/test-send", routes.HandleTestSend)
		tenantAPI.GET("/sending-domain/:domainId/test-send-status", routes.HandleGetTestSendStatus)
		tenantAPI.GET("/sending-domain/:domainId/stats", routes.HandleGetDomainStats)
		tenantAPI.GET("/sending-domain/:domainId/reputation", routes.HandleGetDomainReputation)
		tenantAPI.GET("/sending-domain/:domainId/warming", routes.HandleGetDomainWarming)
		tenantAPI.GET("/sending-domain/:domainId/bounces", routes.HandleGetDomainBounces)
		tenantAPI.POST("/sending-domain/:domainId/pause", routes.HandlePauseDomain)
		tenantAPI.POST("/sending-domain/:domainId/resume", routes.HandleResumeDomain)

		// Story builder — stories, storylines, enactments, scenes, messages,
		// triggers, actions, badges, tags, users, email lists, stats, etc.
		routes.RegisterStoryRoutes(tenantAPI)
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
