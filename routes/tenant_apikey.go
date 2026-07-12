package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
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
		Select(bson.M{"api_key_hash": 1, "api_key_prefix": 1}).One(&tenant); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"exists": tenant.APIKeyHash != "",
		"prefix": tenant.APIKeyPrefix,
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
	key, err := auth.GenerateAPIKey()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID, bson.M{
		"$set": bson.M{
			"api_key_hash":   auth.HashAPIKey(key),
			"api_key_prefix": auth.APIKeyPrefix(key),
		},
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save key"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"api_key": key, // shown once — only the hash is stored
		"prefix":  auth.APIKeyPrefix(key),
	})
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
	c.JSON(http.StatusOK, gin.H{"status": "revoked"})
}
