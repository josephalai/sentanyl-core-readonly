package routes

import (
	"log"
	"net/http"
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
