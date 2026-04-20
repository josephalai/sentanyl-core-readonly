package routes

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

// RegisterContextPackRoutes wires context pack and brand profile endpoints
// onto an already-authenticated tenant router group.
func RegisterContextPackRoutes(tenantAPI *gin.RouterGroup) {
	// Context packs
	tenantAPI.GET("/context-packs", handleListContextPacks)
	tenantAPI.POST("/context-packs", handleCreateContextPack)
	tenantAPI.DELETE("/context-packs/:packId", handleDeleteContextPack)

	// Brand profile
	tenantAPI.GET("/brand-profile", handleGetBrandProfile)
	tenantAPI.PUT("/brand-profile", handleUpsertBrandProfile)

	// Attribute schema
	tenantAPI.GET("/attribute-schema", handleGetAttributeSchema)
	tenantAPI.PUT("/attribute-schema", handleUpsertAttributeSchema)
}

// ─── Context Packs ───────────────────────────────────────────────────────────

func handleListContextPacks(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var packs []pkgmodels.ContextPack
	err := db.GetCollection(pkgmodels.ContextPackCollection).Find(bson.M{
		"tenant_id":              tenantID,
		"timestamps.deleted_at":  nil,
	}).All(&packs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list context packs"})
		return
	}
	if packs == nil {
		packs = []pkgmodels.ContextPack{}
	}
	c.JSON(http.StatusOK, packs)
}

func handleCreateContextPack(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Name     string `json:"name"`
		FileName string `json:"file_name" binding:"required"`
		FileType string `json:"file_type" binding:"required"` // txt, pdf, pasted
		Content  string `json:"content" binding:"required"`
		Notes    string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := req.Name
	if name == "" {
		name = req.FileName
	}

	pack := pkgmodels.NewContextPack(tenantID, name, req.FileName, req.FileType)
	pack.Notes = req.Notes
	pack.OriginalSize = int64(len(req.Content))

	// Chunk into 2000-char segments (same strategy as LMS ReferenceService).
	const chunkSize = 2000
	content := req.Content
	for i := 0; i < len(content); i += chunkSize {
		end := i + chunkSize
		if end > len(content) {
			end = len(content)
		}
		pack.Chunks = append(pack.Chunks, pkgmodels.TextChunk{
			Index: len(pack.Chunks),
			Text:  content[i:end],
			Start: i,
			End:   end,
		})
	}

	now := time.Now()
	pack.CreatedAt = &now

	if err := db.GetCollection(pkgmodels.ContextPackCollection).Insert(pack); err != nil {
		log.Printf("[context-packs] insert error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create context pack"})
		return
	}
	c.JSON(http.StatusCreated, pack)
}

func handleDeleteContextPack(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	packId := c.Param("packId")
	now := time.Now()
	err := db.GetCollection(pkgmodels.ContextPackCollection).Update(
		bson.M{"public_id": packId, "tenant_id": tenantID},
		bson.M{"$set": bson.M{"timestamps.deleted_at": now}},
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "context pack not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": true})
}

// ─── Brand Profile ────────────────────────────────────────────────────────────

func handleGetBrandProfile(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var profile pkgmodels.BrandProfile
	err := db.GetCollection(pkgmodels.BrandProfileCollection).Find(bson.M{
		"tenant_id": tenantID,
	}).One(&profile)
	if err != nil {
		// Return empty profile if not yet created — callers treat 200+empty as "not set yet"
		c.JSON(http.StatusOK, pkgmodels.BrandProfile{})
		return
	}
	c.JSON(http.StatusOK, profile)
}

func handleUpsertBrandProfile(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		VoiceTone         string `json:"voice_tone"`
		AvatarDescription string `json:"avatar_description"`
		Positioning       string `json:"positioning"`
		FooterText        string `json:"footer_text"`
		LegalBlock        string `json:"legal_block"`
		CTAStyle          string `json:"cta_style"`
		DefaultGenPrefs   string `json:"default_gen_prefs"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	var existing pkgmodels.BrandProfile
	err := db.GetCollection(pkgmodels.BrandProfileCollection).Find(bson.M{"tenant_id": tenantID}).One(&existing)
	if err != nil {
		// Create
		profile := pkgmodels.NewBrandProfile(tenantID)
		profile.VoiceTone = req.VoiceTone
		profile.AvatarDescription = req.AvatarDescription
		profile.Positioning = req.Positioning
		profile.FooterText = req.FooterText
		profile.LegalBlock = req.LegalBlock
		profile.CTAStyle = req.CTAStyle
		profile.DefaultGenPrefs = req.DefaultGenPrefs
		profile.CreatedAt = &now
		if err2 := db.GetCollection(pkgmodels.BrandProfileCollection).Insert(profile); err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save brand profile"})
			return
		}
		c.JSON(http.StatusOK, profile)
		return
	}

	// Update
	update := bson.M{
		"voice_tone":          req.VoiceTone,
		"avatar_description":  req.AvatarDescription,
		"positioning":         req.Positioning,
		"footer_text":         req.FooterText,
		"legal_block":         req.LegalBlock,
		"cta_style":           req.CTAStyle,
		"default_gen_prefs":   req.DefaultGenPrefs,
		"timestamps.updated_at": now,
	}
	if err2 := db.GetCollection(pkgmodels.BrandProfileCollection).UpdateId(existing.Id, bson.M{"$set": update}); err2 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update brand profile"})
		return
	}
	existing.VoiceTone = req.VoiceTone
	existing.AvatarDescription = req.AvatarDescription
	existing.Positioning = req.Positioning
	existing.FooterText = req.FooterText
	existing.LegalBlock = req.LegalBlock
	existing.CTAStyle = req.CTAStyle
	existing.DefaultGenPrefs = req.DefaultGenPrefs
	c.JSON(http.StatusOK, existing)
}

// ─── Attribute Schema ─────────────────────────────────────────────────────────

func handleGetAttributeSchema(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var schema pkgmodels.AttributeSchema
	err := db.GetCollection(pkgmodels.AttributeSchemaCollection).Find(bson.M{"tenant_id": tenantID}).One(&schema)
	if err != nil {
		c.JSON(http.StatusOK, pkgmodels.AttributeSchema{Attributes: []pkgmodels.AttributeDef{}})
		return
	}
	if schema.Attributes == nil {
		schema.Attributes = []pkgmodels.AttributeDef{}
	}
	c.JSON(http.StatusOK, schema)
}

func handleUpsertAttributeSchema(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Attributes []pkgmodels.AttributeDef `json:"attributes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate: keys must be non-empty identifiers
	for _, attr := range req.Attributes {
		if strings.TrimSpace(attr.Key) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "attribute key must not be empty"})
			return
		}
	}

	now := time.Now()
	var existing pkgmodels.AttributeSchema
	err := db.GetCollection(pkgmodels.AttributeSchemaCollection).Find(bson.M{"tenant_id": tenantID}).One(&existing)
	if err != nil {
		// Create
		schema := pkgmodels.NewAttributeSchema(tenantID)
		schema.Attributes = req.Attributes
		schema.CreatedAt = &now
		if err2 := db.GetCollection(pkgmodels.AttributeSchemaCollection).Insert(schema); err2 != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save attribute schema"})
			return
		}
		c.JSON(http.StatusOK, schema)
		return
	}

	if err2 := db.GetCollection(pkgmodels.AttributeSchemaCollection).UpdateId(existing.Id, bson.M{"$set": bson.M{
		"attributes":            req.Attributes,
		"timestamps.updated_at": now,
	}}); err2 != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update attribute schema"})
		return
	}
	existing.Attributes = req.Attributes
	c.JSON(http.StatusOK, existing)
}
