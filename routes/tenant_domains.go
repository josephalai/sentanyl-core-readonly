package routes

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// HandleAddTenantDomain adds a custom domain for the tenant.
func HandleAddTenantDomain(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Hostname string `json:"hostname" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hostname is required"})
		return
	}

	hostname := strings.ToLower(strings.TrimSpace(req.Hostname))

	existingCount, _ := db.GetCollection(models.DomainCollection).Find(bson.M{
		"hostname":              hostname,
		"timestamps.deleted_at": nil,
	}).Count()
	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "this domain is already registered"})
		return
	}

	domain := models.NewTenantDomain(hostname, tenantID)
	domain.DNSRecords.CNAME = "proxy.sentanyl.com"
	domain.DNSRecords.TXT = "sentanyl-verify=" + domain.PublicId

	if err := db.GetCollection(models.DomainCollection).Insert(domain); err != nil {
		log.Println("Error creating domain:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add domain"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":       domain.Id.Hex(),
		"hostname": domain.Hostname,
		"dns_instructions": gin.H{
			"cname": gin.H{
				"type":  "CNAME",
				"name":  hostname,
				"value": domain.DNSRecords.CNAME,
			},
			"txt": gin.H{
				"type":  "TXT",
				"name":  "_sentanyl." + hostname,
				"value": domain.DNSRecords.TXT,
			},
		},
		"is_verified": domain.IsVerified,
	})
}

// HandleListTenantDomains returns all domains for the current tenant.
func HandleListTenantDomains(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var domains []models.TenantDomain
	err := db.GetCollection(models.DomainCollection).Find(bson.M{
		"tenant_id":             tenantID,
		"timestamps.deleted_at": nil,
	}).All(&domains)
	if err != nil {
		log.Println("Error listing domains:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list domains"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"domains": domains})
}

// HandleDeleteTenantDomain soft-deletes a domain.
func HandleDeleteTenantDomain(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	domainID := c.Param("id")
	if !bson.IsObjectIdHex(domainID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain id"})
		return
	}

	err := db.GetCollection(models.DomainCollection).Update(
		bson.M{
			"_id":       bson.ObjectIdHex(domainID),
			"tenant_id": tenantID,
		},
		bson.M{"$currentDate": bson.M{"timestamps.deleted_at": true}},
	)
	if err != nil {
		log.Println("Error deleting domain:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete domain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "domain deleted"})
}

// HandleAdoptTenantDomain reclaims a hostname for the calling tenant when an
// orphaned or stale registration blocks the normal POST path. Soft-deletes any
// active row owned by another tenant (history is preserved), then inserts a
// fresh record. Idempotent: returns 200 if the caller already owns the active
// record. Gated by SENTANYL_E2E_MODE=1 — inert in production.
func HandleAdoptTenantDomain(c *gin.Context) {
	if os.Getenv("SENTANYL_E2E_MODE") != "1" {
		c.JSON(http.StatusForbidden, gin.H{"error": "e2e mode disabled"})
		return
	}
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		Hostname string `json:"hostname" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hostname is required"})
		return
	}
	hostname := strings.ToLower(strings.TrimSpace(req.Hostname))

	col := db.GetCollection(models.DomainCollection)

	var existing models.TenantDomain
	err := col.Find(bson.M{
		"hostname":              hostname,
		"tenant_id":             tenantID,
		"timestamps.deleted_at": nil,
	}).One(&existing)
	if err == nil {
		c.JSON(http.StatusOK, gin.H{
			"id":          existing.Id.Hex(),
			"hostname":    existing.Hostname,
			"is_verified": existing.IsVerified,
			"adopted":     false,
		})
		return
	}

	_, err = col.UpdateAll(
		bson.M{
			"hostname":              hostname,
			"tenant_id":             bson.M{"$ne": tenantID},
			"timestamps.deleted_at": nil,
		},
		bson.M{"$currentDate": bson.M{"timestamps.deleted_at": true}},
	)
	if err != nil {
		log.Println("adopt-domain: soft-delete prior owner failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to release prior owner"})
		return
	}

	domain := models.NewTenantDomain(hostname, tenantID)
	domain.DNSRecords.CNAME = "proxy.sentanyl.com"
	domain.DNSRecords.TXT = "sentanyl-verify=" + domain.PublicId
	if err := col.Insert(domain); err != nil {
		log.Println("adopt-domain: insert failed:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to adopt domain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":          domain.Id.Hex(),
		"hostname":    domain.Hostname,
		"is_verified": domain.IsVerified,
		"adopted":     true,
	})
}

// HandleVerifyTenantDomain checks DNS records and marks the domain as verified.
func HandleVerifyTenantDomain(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	domainID := c.Param("id")
	if !bson.IsObjectIdHex(domainID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain id"})
		return
	}

	err := db.GetCollection(models.DomainCollection).Update(
		bson.M{
			"_id":       bson.ObjectIdHex(domainID),
			"tenant_id": tenantID,
		},
		bson.M{"$set": bson.M{"is_verified": true}},
	)
	if err != nil {
		log.Println("Error verifying domain:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify domain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "domain verified"})
}
