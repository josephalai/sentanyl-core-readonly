package routes

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// resolveTenantByHost resolves a tenant from a hostname using the same rules
// as marketing-service/internal/site.FindSiteByDomain: attached custom domain
// first, then the *.site.lvh.me dev pattern (case-insensitive, since browsers
// lowercase hostnames).  Returns the matched tenant id or empty + error.
func resolveTenantByHost(host string) (bson.ObjectId, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return "", fmt.Errorf("empty host")
	}
	var td models.TenantDomain
	if err := db.GetCollection(models.DomainCollection).Find(bson.M{
		"hostname":              host,
		"timestamps.deleted_at": nil,
	}).One(&td); err == nil {
		return td.TenantID, nil
	}
	if strings.HasSuffix(host, ".site.lvh.me") {
		publicID := strings.TrimSuffix(host, ".site.lvh.me")
		if publicID != "" {
			var site models.Site
			if err := db.GetCollection(models.SiteCollection).Find(bson.M{
				"public_id":             bson.RegEx{Pattern: "^" + regexp.QuoteMeta(publicID) + "$", Options: "i"},
				"status":                "published",
				"timestamps.deleted_at": nil,
			}).One(&site); err == nil {
				return site.TenantID, nil
			}
		}
	}
	var site models.Site
	if err := db.GetCollection(models.SiteCollection).Find(bson.M{
		"attached_domains":      bson.RegEx{Pattern: "^" + regexp.QuoteMeta(host) + "$", Options: "i"},
		"status":                "published",
		"timestamps.deleted_at": nil,
	}).One(&site); err == nil {
		return site.TenantID, nil
	}
	return "", fmt.Errorf("no tenant for host %q", host)
}

// HandleTenantRegister creates a new Tenant and AccountUser (Creator registration for the SaaS).
func HandleTenantRegister(c *gin.Context) {
	var req struct {
		BusinessName string `json:"business_name" binding:"required"`
		Email        string `json:"email" binding:"required"`
		Password     string `json:"password" binding:"required"`
		FirstName    string `json:"first_name"`
		LastName     string `json:"last_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "business_name, email, and password are required"})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	existingCount, _ := db.GetCollection(models.AccountUserCollection).Find(bson.M{"email": req.Email}).Count()
	if existingCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "an account with this email already exists"})
		return
	}

	tenant := models.NewTenant(req.BusinessName)

	accountUser := models.NewAccountUser(req.Email, tenant.Id)
	accountUser.Name.FirstName = req.FirstName
	accountUser.Name.LastName = req.LastName

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Println("Error hashing password:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}
	accountUser.PasswordHash = hash

	if err := db.GetCollection(models.TenantCollection).Insert(tenant); err != nil {
		log.Println("Error creating tenant:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create tenant"})
		return
	}
	if err := db.GetCollection(models.AccountUserCollection).Insert(accountUser); err != nil {
		log.Println("Error creating account user:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}

	token, err := auth.GenerateTenantToken(accountUser)
	if err != nil {
		log.Println("Error generating token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"token": token,
		"tenant": gin.H{
			"id":            tenant.Id.Hex(),
			"business_name": tenant.BusinessName,
		},
		"user": gin.H{
			"id":    accountUser.Id.Hex(),
			"email": accountUser.Email,
			"role":  accountUser.Role,
		},
	})
}

// HandleTenantLogin authenticates an AccountUser and returns a JWT.
func HandleTenantLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var accountUser models.AccountUser
	err := db.GetCollection(models.AccountUserCollection).Find(bson.M{"email": req.Email}).One(&accountUser)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	if !auth.CheckPasswordHash(req.Password, accountUser.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	token, err := auth.GenerateTenantToken(&accountUser)
	if err != nil {
		log.Println("Error generating token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	var tenant models.Tenant
	_ = db.GetCollection(models.TenantCollection).FindId(accountUser.TenantID).One(&tenant)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"tenant": gin.H{
			"id":            tenant.Id.Hex(),
			"business_name": tenant.BusinessName,
		},
		"user": gin.H{
			"id":    accountUser.Id.Hex(),
			"email": accountUser.Email,
			"role":  accountUser.Role,
		},
	})
}

// HandleCustomerLogin authenticates a Contact (Customer) for the tenant library.
func HandleCustomerLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	hostname := c.Request.Host
	if forwarded := c.GetHeader("X-Forwarded-Host"); forwarded != "" {
		hostname = forwarded
	}
	tenantID, err := resolveTenantByHost(hostname)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	var contact models.User
	err = db.GetCollection(models.UserCollection).Find(bson.M{
		"email":     models.EmailAddress(req.Email),
		"tenant_id": tenantID,
	}).One(&contact)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	if !auth.CheckPasswordHash(req.Password, contact.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	token, err := auth.GenerateCustomerToken(&contact, tenantID)
	if err != nil {
		log.Println("Error generating customer token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"contact": gin.H{
			"id":    contact.Id.Hex(),
			"email": string(contact.Email),
			"name": gin.H{
				"first": contact.Name.First,
				"last":  contact.Name.Last,
			},
		},
	})
}

// HandleCustomerSetPassword consumes a password_reset_token and sets the
// Contact's password, then issues a customer JWT. The tenant is resolved from
// the request Host (same path HandleCustomerLogin uses).
func HandleCustomerSetPassword(c *gin.Context) {
	var req struct {
		Token    string `json:"token" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "token and password are required"})
		return
	}
	if len(req.Password) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password must be at least 8 characters"})
		return
	}

	hostname := c.Request.Host
	if forwarded := c.GetHeader("X-Forwarded-Host"); forwarded != "" {
		hostname = forwarded
	}

	tenantID, err := resolveTenantByHost(hostname)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	var contact models.User
	err = db.GetCollection(models.UserCollection).Find(bson.M{
		"tenant_id":            tenantID,
		"password_reset_token": req.Token,
	}).One(&contact)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}
	if contact.PasswordResetExpires == nil || time.Now().After(*contact.PasswordResetExpires) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		log.Println("Error hashing password:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set password"})
		return
	}

	err = db.GetCollection(models.UserCollection).Update(
		bson.M{"_id": contact.Id},
		bson.M{
			"$set":   bson.M{"password_hash": hash, "timestamps.updated_at": time.Now()},
			"$unset": bson.M{"password_reset_token": "", "password_reset_expires": ""},
		},
	)
	if err != nil {
		log.Println("Error saving password:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set password"})
		return
	}

	token, err := auth.GenerateCustomerToken(&contact, tenantID)
	if err != nil {
		log.Println("Error generating customer token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"contact": gin.H{
			"id":    contact.Id.Hex(),
			"email": string(contact.Email),
		},
	})
}

// HandleGetTenantProfile returns the current tenant's profile information.
func HandleGetTenantProfile(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var tenant models.Tenant
	err := db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":                         tenant.Id.Hex(),
		"business_name":              tenant.BusinessName,
		"subscription_status":        tenant.SubscriptionStatus,
		"stripe_public_key":          tenant.StripePublicKey,
		"has_stripe":                 tenant.StripeSecretKey != "" || tenant.StripeConnectAccountID != "",
		"has_webhook_secret":         tenant.StripeWebhookSecret != "",
		"has_stripe_connect":         tenant.StripeConnectAccountID != "",
		"stripe_connect_account_id":  tenant.StripeConnectAccountID,
		"stripe_webhook_url":         fmt.Sprintf("%s/api/marketing/stripe/webhook?tenant_id=%s", publicAPIBase(), tenant.Id.Hex()),
		"has_mailgun":                tenant.MailgunAPIKey != "",
		"has_brevo":                  tenant.BrevoAPIKey != "",
	})
}

// publicAPIBase returns the externally-reachable base URL for the platform.
// Used to render the tenant's webhook URL in settings.
func publicAPIBase() string {
	if v := os.Getenv("PUBLIC_API_BASE"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost"
}

// HandleUpdateTenantSettings updates the tenant's integration settings.
func HandleUpdateTenantSettings(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		BusinessName        string `json:"business_name"`
		StripeSecretKey     string `json:"stripe_secret_key"`
		StripePublicKey     string `json:"stripe_public_key"`
		StripeWebhookSecret string `json:"stripe_webhook_secret"`
		MailgunAPIKey       string `json:"mailgun_api_key"`
		MailgunDomain       string `json:"mailgun_domain"`
		BrevoAPIKey         string `json:"brevo_api_key"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	update := bson.M{}
	if req.BusinessName != "" {
		update["business_name"] = req.BusinessName
	}
	if req.StripeSecretKey != "" {
		update["stripe_secret_key"] = req.StripeSecretKey
	}
	if req.StripePublicKey != "" {
		update["stripe_public_key"] = req.StripePublicKey
	}
	if req.StripeWebhookSecret != "" {
		update["stripe_webhook_secret"] = req.StripeWebhookSecret
	}
	if req.MailgunAPIKey != "" {
		update["mailgun_api_key"] = req.MailgunAPIKey
	}
	if req.MailgunDomain != "" {
		update["mailgun_domain"] = req.MailgunDomain
	}
	if req.BrevoAPIKey != "" {
		update["brevo_api_key"] = req.BrevoAPIKey
	}

	if len(update) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no fields to update"})
		return
	}

	err := db.GetCollection(models.TenantCollection).UpdateId(tenantID, bson.M{"$set": update})
	if err != nil {
		log.Println("Error updating tenant settings:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update settings"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "settings updated"})
}
