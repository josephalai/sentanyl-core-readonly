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
	var tenantDomain models.TenantDomain
	err := db.GetCollection(models.DomainCollection).Find(bson.M{
		"hostname":              hostname,
		"timestamps.deleted_at": nil,
	}).One(&tenantDomain)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "domain not found"})
		return
	}

	var contact models.User
	err = db.GetCollection(models.UserCollection).Find(bson.M{
		"email":     models.EmailAddress(req.Email),
		"tenant_id": tenantDomain.TenantID,
	}).One(&contact)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	if !auth.CheckPasswordHash(req.Password, contact.PasswordHash) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	token, err := auth.GenerateCustomerToken(&contact, tenantDomain.TenantID)
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
		"id":                  tenant.Id.Hex(),
		"business_name":       tenant.BusinessName,
		"subscription_status": tenant.SubscriptionStatus,
		"stripe_public_key":   tenant.StripePublicKey,
		"has_stripe":          tenant.StripeSecretKey != "",
		"has_mailgun":         tenant.MailgunAPIKey != "",
		"has_brevo":           tenant.BrevoAPIKey != "",
	})
}

// HandleUpdateTenantSettings updates the tenant's integration settings.
func HandleUpdateTenantSettings(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req struct {
		BusinessName    string `json:"business_name"`
		StripeSecretKey string `json:"stripe_secret_key"`
		StripePublicKey string `json:"stripe_public_key"`
		MailgunAPIKey   string `json:"mailgun_api_key"`
		MailgunDomain   string `json:"mailgun_domain"`
		BrevoAPIKey     string `json:"brevo_api_key"`
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
