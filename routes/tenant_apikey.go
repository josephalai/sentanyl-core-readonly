package routes

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/mcptools"
	"github.com/josephalai/sentanyl/pkg/models"
)

// Self-serve tenant API-key management (Settings → Developers). The key
// authenticates machine-to-machine callers — the tenant send API and the
// Sentanyl MCP server. Only the SHA-256 hash + a display prefix are stored;
// the plaintext is returned exactly once at mint. Same semantics as the
// operator CLI (marketing-service/cmd/tenant-apikey).

// HandleGetTenantAPIKey reports whether a key exists and its display prefix.
func HandleGetTenantAPIKey(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var tenant models.Tenant
	if err := db.GetCollection(models.TenantCollection).FindId(tenantID).
		Select(bson.M{"api_key_hash": 1, "api_key_prefix": 1, "api_key_allowed_tools": 1}).One(&tenant); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"exists":        tenant.APIKeyHash != "",
		"prefix":        tenant.APIKeyPrefix,
		"allowed_tools": tenant.APIKeyAllowedTools,
	})
}

// HandleMintTenantAPIKey mints (or rotates) the tenant API key. Rotation
// invalidates the previous key immediately.
func HandleMintTenantAPIKey(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		AllowedTools *[]string `json:"allowed_tools"`
	}
	_ = c.ShouldBindJSON(&req) // optional body; empty request mints an unscoped key

	key, err := auth.GenerateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}
	set := bson.M{
		"api_key_hash":   auth.HashAPIKey(key),
		"api_key_prefix": auth.APIKeyPrefix(key),
	}
	if req.AllowedTools != nil {
		valid, errMsg := validateAllowedTools(*req.AllowedTools)
		if errMsg != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
			return
		}
		set["api_key_allowed_tools"] = valid
	}
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID, bson.M{
		"$set": set,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save key"})
		return
	}
	revokeMachineSessions(tenantID)
	c.JSON(http.StatusOK, gin.H{
		"api_key": key, // shown once — only the hash is stored
		"prefix":  auth.APIKeyPrefix(key),
	})
}

// HandleUpdateTenantAPIKeyScopes changes the key's tool allowlist without
// rotating the key. An empty array clears scoping (all tools allowed).
func HandleUpdateTenantAPIKeyScopes(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		AllowedTools []string `json:"allowed_tools"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "allowed_tools required"})
		return
	}
	valid, errMsg := validateAllowedTools(req.AllowedTools)
	if errMsg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	update := bson.M{"$set": bson.M{"api_key_allowed_tools": valid}}
	if len(valid) == 0 {
		update = bson.M{"$unset": bson.M{"api_key_allowed_tools": ""}}
	}
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID, update); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update scopes"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"allowed_tools": valid})
}

// HandleListTenantAPIKeyTools lists the MCP tool registry so the Settings UI
// can offer a scope picker whose names always match validateAllowedTools.
func HandleListTenantAPIKeyTools(c *gin.Context) {
	tools := make([]gin.H, 0, len(mcptools.Tools))
	for _, t := range mcptools.Tools {
		tools = append(tools, gin.H{
			"name":        t.Name,
			"description": t.Description,
			"service":     t.Service,
		})
	}
	c.JSON(http.StatusOK, gin.H{"tools": tools})
}

// validateAllowedTools checks names against the MCP tool registry.
func validateAllowedTools(names []string) ([]string, string) {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if mcptools.Find(n) == nil {
			return nil, "unknown tool: " + n
		}
		out = append(out, n)
	}
	return out, ""
}

// HandleRevokeTenantAPIKey deletes the key; machine callers stop working
// until a new one is minted.
func HandleRevokeTenantAPIKey(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID, bson.M{
		"$unset": bson.M{"api_key_hash": "", "api_key_prefix": ""},
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke key"})
		return
	}
	revokeMachineSessions(tenantID)
	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}

// revokeMachineSessions kills outstanding machine-principal tokens when the
// API key they were minted under is rotated or revoked (MCP-001 kill-switch).
// Non-fatal: the key change itself already blocks new mints.
func revokeMachineSessions(tenantID bson.ObjectId) {
	if n, err := auth.RevokeMachineSessions(tenantID); err != nil {
		log.Printf("api-key: revoke machine sessions for tenant %s: %v", tenantID.Hex(), err)
	} else if n > 0 {
		log.Printf("api-key: revoked %d machine session(s) for tenant %s", n, tenantID.Hex())
	}
}
