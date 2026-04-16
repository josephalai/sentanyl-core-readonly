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
	{ID: "simple-one-storyline", Category: "basic", Name: "Simple One Storyline", Description: "Minimal single-storyline campaign.", Source: scripting.FixtureSimpleOneStoryline},
	{ID: "multi-storyline", Category: "basic", Name: "Multi Storyline", Description: "Campaign with 3 storylines in sequence.", Source: scripting.FixtureMultiStoryline},
	{ID: "multi-enactment", Category: "basic", Name: "Multi Enactment", Description: "Storyline with 3 enactments.", Source: scripting.FixtureMultiEnactment},
	{ID: "multi-scene", Category: "basic", Name: "Multi Scene", Description: "Enactment with 3 scenes.", Source: scripting.FixtureMultiScene},
	{ID: "conditional-badge-routing", Category: "advanced", Name: "Conditional Badge Routing", Description: "Routing by badge status.", Source: scripting.FixtureConditionalBadgeRouting},
	{ID: "click-branching", Category: "advanced", Name: "Click Branching", Description: "Click/not-click branching.", Source: scripting.FixtureClickBranching},
	{ID: "open-branching", Category: "advanced", Name: "Open Branching", Description: "Open/not-open branching.", Source: scripting.FixtureOpenBranching},
	{ID: "bounded-retry", Category: "advanced", Name: "Bounded Retry", Description: "Retry with max count and fallback.", Source: scripting.FixtureBoundedRetry},
	{ID: "loop-to-prior-enactment", Category: "advanced", Name: "Loop to Prior Enactment", Description: "Loop back to an earlier enactment.", Source: scripting.FixtureLoopToPriorEnactment},
	{ID: "failure-fallback", Category: "advanced", Name: "Failure Fallback", Description: "Failure path with badge transaction.", Source: scripting.FixtureFailureFallback},
	{ID: "completion-path", Category: "advanced", Name: "Completion Path", Description: "Completion with badges and next story reference.", Source: scripting.FixtureCompletionPath},
	{ID: "full-campaign", Category: "comprehensive", Name: "Full Campaign", Description: "Comprehensive campaign exercising all features.", Source: scripting.FixtureFullCampaign},
	{ID: "compact-campaign", Category: "authoring", Name: "Compact Campaign", Description: "Compact authoring sugar.", Source: scripting.FixtureCompactCampaign},
	{ID: "default-sender", Category: "authoring", Name: "Default Sender", Description: "Basic sender default inheritance.", Source: scripting.FixtureDefaultSender},
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
