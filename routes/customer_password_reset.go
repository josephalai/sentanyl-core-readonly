package routes

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
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

	token, hashed, err := auth.MintResetToken()
	if err != nil {
		c.JSON(http.StatusOK, accepted)
		return
	}
	expires := time.Now().Add(48 * time.Hour)
	if err := db.GetCollection(models.UserCollection).UpdateId(contact.Id, bson.M{
		"$set": bson.M{
			"password_reset_token":   hashed,
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

// RetirePlaintextResetTokens unsets any password_reset_token still stored in
// legacy plaintext form (pre ID-015 hashing). Runs at startup, idempotent.
// Rollback/recovery: affected customers re-request a reset — tokens expire
// after 48h regardless, so nothing durable is lost.
func RetirePlaintextResetTokens() {
	info, err := db.GetCollection(models.UserCollection).UpdateAll(
		bson.M{
			"password_reset_token": bson.M{
				"$exists": true,
				"$ne":     "",
				"$not":    bson.M{"$regex": "^sha256:"},
			},
		},
		bson.M{"$unset": bson.M{"password_reset_token": "", "password_reset_expires": ""}},
	)
	if err != nil {
		log.Printf("[customer-reset] plaintext token retirement: %v", err)
		return
	}
	if info.Updated > 0 {
		log.Printf("[customer-reset] retired %d legacy plaintext reset tokens", info.Updated)
	}
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
