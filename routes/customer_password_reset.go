package routes

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/emailer"
	"github.com/josephalai/sentanyl/pkg/models"
)

// HandleCustomerRequestReset is the customer forgot-password flow: mints a
// fresh 48h password-reset token for the contact and emails the portal
// set-password link. Buyers whose account was created from a Stripe checkout
// email (they never typed it anywhere else) recover access with just that
// address. Always answers 200 so the endpoint cannot be used to probe which
// emails have accounts.
func HandleCustomerRequestReset(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	accepted := gin.H{"status": "ok", "message": "If that email has an account, a reset link is on its way."}

	hostname := c.Request.Host
	if forwarded := c.GetHeader("X-Forwarded-Host"); forwarded != "" {
		hostname = forwarded
	}
	tenantID, err := resolveTenantByHost(hostname)
	if err != nil {
		c.JSON(http.StatusOK, accepted)
		return
	}

	var contact models.User
	if err := db.GetCollection(models.UserCollection).Find(bson.M{
		"email":                 models.EmailAddress(req.Email),
		"tenant_id":             tenantID,
		"timestamps.deleted_at": nil,
	}).One(&contact); err != nil {
		c.JSON(http.StatusOK, accepted)
		return
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		c.JSON(http.StatusOK, accepted)
		return
	}
	token := hex.EncodeToString(buf)
	expires := time.Now().Add(48 * time.Hour)
	if err := db.GetCollection(models.UserCollection).UpdateId(contact.Id, bson.M{
		"$set": bson.M{
			"password_reset_token":   token,
			"password_reset_expires": expires,
			"timestamps.updated_at":  time.Now(),
		},
	}); err != nil {
		log.Printf("[customer-reset] token save failed for %s: %v", req.Email, err)
		c.JSON(http.StatusOK, accepted)
		return
	}

	go sendCustomerResetEmail(tenantID, hostname, req.Email, token)
	c.JSON(http.StatusOK, accepted)
}

// sendCustomerResetEmail delivers the set-password link on the host the
// customer asked from, sent from the tenant's configured from-domain.
func sendCustomerResetEmail(tenantID bson.ObjectId, hostname, toEmail, token string) {
	scheme := "https"
	if strings.Contains(hostname, "localhost") || strings.Contains(hostname, "lvh.me") {
		scheme = "http"
	}
	link := fmt.Sprintf("%s://%s/portal/set-password?token=%s", scheme, hostname, token)

	var tenant models.Tenant
	_ = db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant)
	from := "no-reply@sentanyl.local"
	if tenant.MailgunDomain != "" {
		from = "no-reply@" + tenant.MailgunDomain
	}
	business := tenant.BusinessName
	if business == "" {
		business = hostname
	}

	body := fmt.Sprintf(`<p>Someone (hopefully you) asked to reset the password for your %s account.</p>
<p><a href="%s">Set a new password</a></p>
<p>The link expires in 48 hours. If you didn't request this, you can ignore this email — your password is unchanged.</p>`,
		business, link)

	provider := emailer.FromEnv()
	if provider == nil {
		log.Printf("[customer-reset] no mail provider configured; reset link for %s: %s", toEmail, link)
		return
	}
	if err := provider.SendEmail(from, toEmail, "Reset your password", body, ""); err != nil {
		log.Printf("[customer-reset] send to %s failed: %v", toEmail, err)
	}
}
