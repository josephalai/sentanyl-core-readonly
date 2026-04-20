package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/core-service/scripting"
	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

// fixtureEntry describes a named fixture script available via the API.
type fixtureEntry struct {
	ID          string `json:"id"`
	Category    string `json:"category"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source,omitempty"`
}

var allFixtures = []fixtureEntry{
	// ── Basic ──────────────────────────────────────────────────────────────
	{ID: "simple-one-storyline", Category: "basic", Name: "Simple One Storyline", Description: "Minimal single-storyline campaign.", Source: scripting.FixtureSimpleOneStoryline},
	{ID: "multi-storyline", Category: "basic", Name: "Multi Storyline", Description: "Campaign with 3 storylines in sequence.", Source: scripting.FixtureMultiStoryline},
	{ID: "multi-enactment", Category: "basic", Name: "Multi Enactment", Description: "Storyline with 3 enactments.", Source: scripting.FixtureMultiEnactment},
	{ID: "multi-scene", Category: "basic", Name: "Multi Scene", Description: "Enactment with 3 scenes.", Source: scripting.FixtureMultiScene},

	// ── Advanced branching & flow ──────────────────────────────────────────
	{ID: "conditional-badge-routing", Category: "advanced", Name: "Conditional Badge Routing", Description: "Routing by badge status.", Source: scripting.FixtureConditionalBadgeRouting},
	{ID: "click-branching", Category: "advanced", Name: "Click Branching", Description: "Click/not-click branching.", Source: scripting.FixtureClickBranching},
	{ID: "open-branching", Category: "advanced", Name: "Open Branching", Description: "Open/not-open branching.", Source: scripting.FixtureOpenBranching},
	{ID: "bounded-retry", Category: "advanced", Name: "Bounded Retry", Description: "Retry with max count and fallback.", Source: scripting.FixtureBoundedRetry},
	{ID: "loop-to-prior-enactment", Category: "advanced", Name: "Loop to Prior Enactment", Description: "Loop back to an earlier enactment.", Source: scripting.FixtureLoopToPriorEnactment},
	{ID: "failure-fallback", Category: "advanced", Name: "Failure Fallback", Description: "Failure path with badge transaction.", Source: scripting.FixtureFailureFallback},
	{ID: "completion-path", Category: "advanced", Name: "Completion Path", Description: "Completion with badges and next story reference.", Source: scripting.FixtureCompletionPath},
	{ID: "dot-access-triggers", Category: "advanced", Name: "Dot-Access Triggers", Description: "Trigger conditions using dot-access expressions.", Source: scripting.FixtureDotAccessTriggers},
	{ID: "deferred-transitions", Category: "advanced", Name: "Deferred Transitions", Description: "Timed and deferred story transitions.", Source: scripting.FixtureDeferredTransitions},
	{ID: "hybrid-transitions", Category: "advanced", Name: "Hybrid Transitions", Description: "Mix of immediate and deferred transitions.", Source: scripting.FixtureHybridTransitions},

	// ── Authoring patterns ─────────────────────────────────────────────────
	{ID: "compact-campaign", Category: "authoring", Name: "Compact Campaign", Description: "Compact authoring sugar.", Source: scripting.FixtureCompactCampaign},
	{ID: "default-sender", Category: "authoring", Name: "Default Sender", Description: "Basic sender default inheritance.", Source: scripting.FixtureDefaultSender},
	{ID: "links-and-policies", Category: "authoring", Name: "Links & Policies", Description: "Link definitions and policy blocks.", Source: scripting.FixtureLinksAndPolicies},
	{ID: "pattern-reuse", Category: "authoring", Name: "Pattern Reuse", Description: "Reusable scene patterns.", Source: scripting.FixturePatternReuse},
	{ID: "scenes-range", Category: "authoring", Name: "Scenes Range", Description: "Scene range shorthand (scenes 1..3).", Source: scripting.FixtureScenesRange},
	{ID: "data-block-for-loop", Category: "authoring", Name: "Data Block + For Loop", Description: "Data blocks with for-loop expansion.", Source: scripting.FixtureDataBlockForLoop},
	{ID: "inline-data-loop", Category: "authoring", Name: "Inline Data Loop", Description: "Inline data with loop generation.", Source: scripting.FixtureInlineDataLoop},
	{ID: "enactment-defaults", Category: "authoring", Name: "Enactment Defaults", Description: "Enactment-level default settings.", Source: scripting.FixtureEnactmentDefaults},
	{ID: "scene-defaults-triggers", Category: "authoring", Name: "Scene Defaults + Triggers", Description: "Scene defaults combined with trigger patterns.", Source: scripting.FixtureSceneDefaultsTriggers},
	{ID: "handlebars-vars", Category: "authoring", Name: "Handlebars Variables", Description: "Template variable substitution.", Source: scripting.FixtureHandlebarsVars},
	{ID: "scene-template-name", Category: "authoring", Name: "Scene Template Name", Description: "Using named email templates in scenes.", Source: scripting.FixtureSceneTemplateName},

	// ── Generative / LLM ──────────────────────────────────────────────────
	{ID: "storyline-generation", Category: "generative", Name: "Storyline Generation", Description: "AI-assisted storyline generation.", Source: scripting.FixtureStorylineGeneration},
	{ID: "full-generative-campaign", Category: "generative", Name: "Full Generative Campaign", Description: "Complete AI-generated campaign with funnels and products.", Source: scripting.FixtureFullGenerativeCampaign},

	// ── Comprehensive ──────────────────────────────────────────────────────
	{ID: "full-campaign", Category: "comprehensive", Name: "Full Campaign", Description: "Comprehensive campaign exercising all features.", Source: scripting.FixtureFullCampaign},
	{ID: "multi-story-sequence", Category: "comprehensive", Name: "Multi-Story Sequence", Description: "Multiple stories with cross-story transitions.", Source: scripting.FixtureMultiStorySequence},
	{ID: "mailhog-full-sequence", Category: "comprehensive", Name: "Full Email Sequence", Description: "Complete email sequence with delivery tracking.", Source: scripting.FixtureMailhogFullSequence},
	{ID: "multi-storyline-enactment-scene", Category: "comprehensive", Name: "Multi-Storyline + Enactment + Scene", Description: "Full nesting: multiple storylines, enactments, and scenes.", Source: scripting.FixtureMultiStorylineEnactmentScene},

	// ── E2E scenarios ──────────────────────────────────────────────────────
	{ID: "compound-trigger-conditions", Category: "e2e", Name: "Compound Trigger Conditions", Description: "Complex AND/OR trigger condition trees.", Source: scripting.FixtureCompoundTriggerConditions},
	{ID: "conditional-routing", Category: "e2e", Name: "Conditional Routing", Description: "Badge-gated conditional routing.", Source: scripting.FixtureConditionalRouting},
	{ID: "conditional-trigger", Category: "e2e", Name: "Conditional Trigger", Description: "Triggers with conditional logic.", Source: scripting.FixtureConditionalTrigger},
	{ID: "storyline-badge-gating", Category: "e2e", Name: "Storyline Badge Gating", Description: "Badge-gated storyline access control.", Source: scripting.FixtureStorylineBadgeGating},
	{ID: "story-interruption", Category: "e2e", Name: "Story Interruption", Description: "High-priority story interrupting a running campaign.", Source: scripting.FixtureStoryInterruption},
	{ID: "outbound-webhooks", Category: "e2e", Name: "Outbound Webhooks", Description: "Webhook-triggered automations.", Source: scripting.FixtureOutboundWebhooks},
	{ID: "persistent-links", Category: "e2e", Name: "Persistent Links", Description: "Click tracking with persistent link scope.", Source: scripting.FixturePersistentLinks},
	{ID: "next-story-hopping", Category: "e2e", Name: "Next Story Hopping", Description: "Chaining multiple stories in sequence.", Source: scripting.FixtureNextStoryHopping},
	{ID: "storyline-on-fail-routes", Category: "e2e", Name: "Storyline On-Fail Routes", Description: "Failure path routing between storylines.", Source: scripting.FixtureStorylineOnFailRoutes},

	// ── Atomic feature coverage ────────────────────────────────────────────
	{ID: "atomic-all-trigger-types", Category: "atomic", Name: "All Trigger Types", Description: "Exercises all 15 trigger types.", Source: scripting.FixtureAtomicAllTriggerTypes},
	{ID: "atomic-all-action-types", Category: "atomic", Name: "All Action Types", Description: "Exercises all 20 action types.", Source: scripting.FixtureAtomicAllActionTypes},
	{ID: "atomic-badge-integration", Category: "atomic", Name: "Badge Integration", Description: "All badge-related constructs.", Source: scripting.FixtureAtomicBadgeIntegration},
	{ID: "atomic-conditions-routing", Category: "atomic", Name: "Conditions & Routing", Description: "All condition types and routing patterns.", Source: scripting.FixtureAtomicConditionsAndRouting},
	{ID: "atomic-scene-features", Category: "atomic", Name: "Scene Features", Description: "Scene-specific feature coverage.", Source: scripting.FixtureAtomicSceneFeatures},
	{ID: "atomic-badge-campaign", Category: "atomic", Name: "Badge Campaign", Description: "Generative features with badge integration.", Source: scripting.FixtureAtomicBadgeCampaign},

	// ── Integration Tests: core stories ────────────────────────────────────
	{ID: "it-story-lifecycle", Category: "integration", Name: "Story Lifecycle", Description: "Full story lifecycle from start to completion.", Source: scripting.IntegrationTestStoryLifecycle},
	{ID: "it-storyline-badge-gating", Category: "integration", Name: "Storyline Badge Gating", Description: "Badge-gated storyline access.", Source: scripting.IntegrationTestStorylineBadgeGating},
	{ID: "it-trigger-types", Category: "integration", Name: "All Trigger Types", Description: "Every trigger type exercised.", Source: scripting.IntegrationTestTriggerTypes},
	{ID: "it-action-types", Category: "integration", Name: "All Action Types", Description: "Every action type exercised.", Source: scripting.IntegrationTestActionTypes},
	{ID: "it-conditional-routing", Category: "integration", Name: "Conditional Routing", Description: "Badge-gated conditional routing.", Source: scripting.IntegrationTestConditionalRouting},
	{ID: "it-click-branching", Category: "integration", Name: "Click Branching", Description: "Click/not-click branching paths.", Source: scripting.IntegrationTestClickBranching},
	{ID: "it-retry-and-loops", Category: "integration", Name: "Retry & Loops", Description: "Retry logic and loop patterns.", Source: scripting.IntegrationTestRetryAndLoops},
	{ID: "it-story-interruption", Category: "integration", Name: "Story Interruption", Description: "High-priority story interrupting a running campaign.", Source: scripting.IntegrationTestStoryInterruption},
	{ID: "it-deferred-transitions", Category: "integration", Name: "Deferred Transitions", Description: "Timed/deferred story transitions.", Source: scripting.IntegrationTestDeferredTransitions},
	{ID: "it-persistent-links", Category: "integration", Name: "Persistent Links", Description: "Persistent click-tracking links.", Source: scripting.IntegrationTestPersistentLinks},
	{ID: "it-scene-template-vars", Category: "integration", Name: "Scene Template Vars", Description: "Template variable substitution in scenes.", Source: scripting.IntegrationTestSceneTemplateVars},
	{ID: "it-scene-defaults", Category: "integration", Name: "Scene Defaults", Description: "Scene default settings inheritance.", Source: scripting.IntegrationTestSceneDefaults},
	{ID: "it-enactment-defaults", Category: "integration", Name: "Enactment Defaults", Description: "Enactment default settings.", Source: scripting.IntegrationTestEnactmentDefaults},
	{ID: "it-default-sender", Category: "integration", Name: "Default Sender", Description: "Sender default inheritance.", Source: scripting.IntegrationTestDefaultSender},
	{ID: "it-links-and-policies", Category: "integration", Name: "Links & Policies", Description: "Link and policy definitions.", Source: scripting.IntegrationTestLinksAndPolicies},
	{ID: "it-pattern-reuse", Category: "integration", Name: "Pattern Reuse", Description: "Reusable scene patterns.", Source: scripting.IntegrationTestPatternReuse},
	{ID: "it-data-blocks-for-loops", Category: "integration", Name: "Data Blocks + For Loops", Description: "Data blocks with for-loop expansion.", Source: scripting.IntegrationTestDataBlocksForLoops},
	{ID: "it-scenes-range", Category: "integration", Name: "Scenes Range", Description: "Scene range shorthand.", Source: scripting.IntegrationTestScenesRange},
	{ID: "it-multi-story", Category: "integration", Name: "Multi-Story", Description: "Multiple stories in one script.", Source: scripting.IntegrationTestMultiStory},
	{ID: "it-next-story-hopping", Category: "integration", Name: "Next Story Hopping", Description: "Cross-story chaining.", Source: scripting.IntegrationTestNextStoryHopping},
	{ID: "it-storyline-on-fail", Category: "integration", Name: "Storyline On-Fail Routes", Description: "Failure path routing.", Source: scripting.IntegrationTestStorylineOnFailRoutes},
	{ID: "it-badge-integration", Category: "integration", Name: "Badge Integration", Description: "Full badge lifecycle.", Source: scripting.IntegrationTestBadgeIntegration},
	{ID: "it-condition-guards", Category: "integration", Name: "Condition Guards", Description: "Guard conditions on triggers.", Source: scripting.IntegrationTestConditionGuards},
	{ID: "it-multi-scene-drip", Category: "integration", Name: "Multi-Scene Drip", Description: "Drip email sequence across scenes.", Source: scripting.IntegrationTestMultiSceneDrip},
	{ID: "it-skip-storyline-expiry", Category: "integration", Name: "Skip on Storyline Expiry", Description: "Auto-advance on storyline expiry.", Source: scripting.IntegrationTestSkipStorylineExpiry},
	{ID: "it-dot-access", Category: "integration", Name: "Dot-Access Expressions", Description: "Dot-access data references.", Source: scripting.IntegrationTestDotAccess},
	{ID: "it-compound-conditions", Category: "integration", Name: "Compound Conditions", Description: "AND/OR compound condition trees.", Source: scripting.IntegrationTestCompoundConditions},

	// ── Integration Tests: funnels & commerce ──────────────────────────────
	{ID: "it-funnel-video", Category: "funnels", Name: "Funnel with Video", Description: "Funnel page with embedded video intelligence.", Source: scripting.IntegrationTestFunnelWithVideo},
	{ID: "it-checkout-purchase", Category: "funnels", Name: "Checkout & Purchase", Description: "Checkout flow with purchase tracking.", Source: scripting.IntegrationTestCheckoutPurchase},
	{ID: "it-full-pipeline", Category: "funnels", Name: "Full Sales Pipeline", Description: "End-to-end sales pipeline.", Source: scripting.IntegrationTestFullPipeline},
	{ID: "it-video-watch-operators", Category: "funnels", Name: "Video Watch Operators", Description: "Video progress and watch threshold operators.", Source: scripting.IntegrationTestVideoWatchOperators},
	{ID: "it-lead-magnet", Category: "funnels", Name: "Lead Magnet Funnel", Description: "Lead magnet opt-in funnel.", Source: scripting.IntegrationTestLeadMagnetFunnel},
	{ID: "it-webinar-funnel", Category: "funnels", Name: "Webinar Funnel", Description: "Webinar registration + replay funnel with reminder campaign.", Source: scripting.IntegrationTestWebinarFunnel},
	{ID: "it-product-launch", Category: "funnels", Name: "Product Launch Funnel", Description: "Full product launch with waitlist, checkout, and upsell.", Source: scripting.IntegrationTestProductLaunchFunnel},
	{ID: "it-multi-route-membership", Category: "funnels", Name: "Multi-Route Membership", Description: "Membership site with multiple access tiers.", Source: scripting.IntegrationTestMultiRouteMembership},
	{ID: "it-upsell-pipeline", Category: "funnels", Name: "Upsell Pipeline", Description: "One-click upsell pipeline.", Source: scripting.IntegrationTestUpsellPipeline},
	{ID: "it-abandon-recovery", Category: "funnels", Name: "Abandon Recovery", Description: "Cart/page abandonment recovery sequence.", Source: scripting.IntegrationTestAbandonRecovery},

	// ── Integration Tests: combo real-world scenarios ──────────────────────
	{ID: "it-combo-ecommerce", Category: "real-world", Name: "E-Commerce Funnel", Description: "Full e-commerce funnel with products, offers, and automation.", Source: scripting.IntegrationTestComboEcommerceFunnel},
	{ID: "it-combo-saas-onboarding", Category: "real-world", Name: "SaaS Onboarding", Description: "SaaS trial onboarding campaign with badge-gated upgrades.", Source: scripting.IntegrationTestComboSaaSOnboarding},
	{ID: "it-combo-newsletter", Category: "real-world", Name: "Newsletter Engagement", Description: "Newsletter engagement campaign with click branching.", Source: scripting.IntegrationTestComboNewsletterEngagement},
	{ID: "it-combo-event-promo", Category: "real-world", Name: "Event Promotion", Description: "Live event promotion funnel with registration and reminders.", Source: scripting.IntegrationTestComboEventPromotion},

	// ── Integration Tests: LMS & Video ─────────────────────────────────────
	{ID: "it-global-website", Category: "lms-video", Name: "Global Website", Description: "Multi-page website with navigation and SEO.", Source: scripting.IntegrationTestGlobalWebsite},
	{ID: "it-lms-modules", Category: "lms-video", Name: "LMS Modules", Description: "Course with modules, lessons, and quizzes.", Source: scripting.IntegrationTestLMSModules},
	{ID: "it-video-intelligence", Category: "lms-video", Name: "Video Intelligence", Description: "Video player with badge rules, chapters, and turnstiles.", Source: scripting.IntegrationTestVideoIntelligence},
}

// RegisterScriptRoutes wires up all /api/script/* endpoints.
func RegisterScriptRoutes(r *gin.Engine) {
	r.GET("/api/script/fixtures", handleListFixtures)
	r.GET("/api/script/fixture/:id", handleGetFixture)
	r.POST("/api/script/compile", handleCompileScript)
	r.POST("/api/script/validate", handleValidateScript)
	r.POST("/api/script/deploy", handleDeployScript)
	r.POST("/api/script/ai", handleScriptAI)
	r.GET("/api/script/reference", handleScriptReference)
}

func handleListFixtures(c *gin.Context) {
	summaries := make([]gin.H, 0, len(allFixtures))
	for _, f := range allFixtures {
		summaries = append(summaries, gin.H{
			"id":          f.ID,
			"category":    f.Category,
			"name":        f.Name,
			"description": f.Description,
		})
	}
	c.JSON(http.StatusOK, gin.H{"fixtures": summaries})
}

func handleGetFixture(c *gin.Context) {
	id := c.Param("id")
	for _, f := range allFixtures {
		if f.ID == id {
			c.JSON(http.StatusOK, f)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "fixture not found"})
}

func handleCompileScript(c *gin.Context) {
	var req struct {
		Source       string `json:"source" binding:"required"`
		SubscriberID string `json:"subscriber_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source is required"})
		return
	}

	creatorID := bson.NewObjectId()
	result := scripting.CompileScript(req.Source, req.SubscriberID, creatorID)

	diags := formatDiagnostics(result.Diagnostics)
	c.JSON(http.StatusOK, gin.H{
		"stories":     result.Stories,
		"funnels":     result.Funnels,
		"products":    result.Products,
		"offers":      result.Offers,
		"sites":       []interface{}{},
		"badges":      badgeMapToSlice(result.Badges),
		"quizzes":     result.Quizzes,
		"diagnostics": diags,
	})
}

func handleValidateScript(c *gin.Context) {
	var req struct {
		Source string `json:"source" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source is required"})
		return
	}

	result := scripting.ValidateScript(req.Source)
	diags := formatDiagnostics(result.Diagnostics)
	c.JSON(http.StatusOK, gin.H{
		"valid":       !result.Diagnostics.HasErrors(),
		"diagnostics": diags,
	})
}

func handleDeployScript(c *gin.Context) {
	var req struct {
		Source        string `json:"source" binding:"required"`
		SubscriberID  string `json:"subscriber_id"`
		StoryIndices  []int  `json:"story_indices"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "source is required"})
		return
	}

	creatorID := bson.NewObjectId()
	result := scripting.CompileScript(req.Source, req.SubscriberID, creatorID)
	if result.Diagnostics.HasErrors() {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":       "compilation failed",
			"diagnostics": formatDiagnostics(result.Diagnostics),
		})
		return
	}

	// Resolve the tenant ObjectId from the subscriber_id hex string.
	// The compiler sets SubscriberId (string) but Offer/Product have TenantID (ObjectId)
	// which was previously set by the monolith's auth middleware. Stamp it here.
	var tenantOID bson.ObjectId
	if bson.IsObjectIdHex(req.SubscriberID) {
		tenantOID = bson.ObjectIdHex(req.SubscriberID)
	}

	// Persist compiled stories (filter by indices if requested).
	stories := result.Stories
	if len(req.StoryIndices) > 0 {
		filtered := make([]*pkgmodels.Story, 0)
		for _, idx := range req.StoryIndices {
			if idx >= 0 && idx < len(stories) {
				filtered = append(filtered, stories[idx])
			}
		}
		stories = filtered
	}

	for _, s := range stories {
		if err := db.GetCollection(pkgmodels.StoryCollection).Insert(s); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist story: " + err.Error()})
			return
		}
	}

	for _, f := range result.Funnels {
		if tenantOID != "" && f.TenantID == "" {
			f.TenantID = tenantOID
		}
		if err := db.GetCollection(pkgmodels.FunnelCollection).Insert(f); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist funnel: " + err.Error()})
			return
		}
	}

	for _, p := range result.Products {
		if tenantOID != "" && p.TenantID == "" {
			p.TenantID = tenantOID
		}
		if err := db.GetCollection(pkgmodels.ProductCollection).Insert(p); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist product: " + err.Error()})
			return
		}
	}

	for _, o := range result.Offers {
		if tenantOID != "" && o.TenantID == "" {
			o.TenantID = tenantOID
		}
		if err := db.GetCollection(pkgmodels.OfferCollection).Insert(o); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist offer: " + err.Error()})
			return
		}
	}

	for _, b := range result.Badges {
		if b == nil {
			continue
		}
		if err := db.GetCollection(pkgmodels.BadgeCollection).Insert(b); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist badge: " + err.Error()})
			return
		}
	}

	// Persist LMS quizzes (separate collection, linked to products via ProductID + ModuleSlug).
	for _, q := range result.Quizzes {
		if q == nil {
			continue
		}
		if err := db.GetCollection(pkgmodels.LMSQuizCollection).Insert(q); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist quiz: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"stories":     result.Stories,
		"funnels":     result.Funnels,
		"products":    result.Products,
		"offers":      result.Offers,
		"sites":       []interface{}{},
		"badges":      badgeMapToSlice(result.Badges),
		"quizzes":     result.Quizzes,
		"diagnostics": formatDiagnostics(result.Diagnostics),
	})
}

// handleScriptAI is defined in script_ai.go

func handleScriptReference(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"reference": "SentanylScript DSL reference — see docs."})
}

// badgeMapToSlice converts the compiler's map[name]*Badge into the array
// shape the frontend expects: [{name, public_id}, ...].
func badgeMapToSlice(m map[string]*pkgmodels.Badge) []gin.H {
	out := make([]gin.H, 0, len(m))
	for _, b := range m {
		if b == nil {
			continue
		}
		out = append(out, gin.H{"name": b.Name, "public_id": b.PublicId})
	}
	return out
}

func formatDiagnostics(diags scripting.Diagnostics) []gin.H {
	out := make([]gin.H, 0, len(diags))
	for _, d := range diags {
		level := "warning"
		if d.Level == scripting.DiagError {
			level = "error"
		}
		out = append(out, gin.H{
			"level":   level,
			"message": d.Message,
			"pos":     d.Pos.String(),
		})
	}
	return out
}
