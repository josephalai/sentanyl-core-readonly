package routes

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/josephalai/sentanyl/pkg/audit"
	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/utils"

	"github.com/gin-gonic/gin"
	mgo "gopkg.in/mgo.v2"
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

	// ID-010: registration is atomic w.r.t. concurrent duplicate signups. The
	// unique index on account_users.email (EnsureIdentityIndexes) is the race
	// backstop the Count() pre-check cannot provide; on a duplicate-key insert
	// we roll back the tenant so a failed second insert never orphans a Tenant.
	tenant := models.NewTenant(req.BusinessName)

	// Trial-recycling guard: a signup whose normalized email (case, gmail
	// dots, +tags stripped) matches a previous tenant's fingerprint gets no
	// fresh trial — the account is created, but gated until payment.
	tenant.SignupEmailNormalized = utils.NormalizeSignupEmail(req.Email)
	if n, _ := db.GetCollection(models.TenantCollection).Find(bson.M{
		"signup_email_normalized": tenant.SignupEmailNormalized,
	}).Count(); n > 0 {
		expired := time.Now().UTC()
		tenant.TrialEndsAt = &expired
		log.Printf("tenant register: trial-recycling fingerprint match for %s — trial not granted", tenant.SignupEmailNormalized)
	}

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
		// Roll back the just-created tenant so a lost race (or any account
		// insert failure) never leaves an orphaned Tenant (ID-010).
		_ = db.GetCollection(models.TenantCollection).RemoveId(tenant.Id)
		if mgo.IsDup(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "an account with this email already exists"})
			return
		}
		log.Println("Error creating account user:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create account"})
		return
	}
	membership := models.NewWorkspaceMembership(tenant.Id, accountUser.Id, auth.RoleOwner, accountUser.Id)
	membership.Source = "registration"
	if err := db.GetCollection(models.WorkspaceMembershipCollection).Insert(membership); err != nil {
		_ = db.GetCollection(models.AccountUserCollection).RemoveId(accountUser.Id)
		_ = db.GetCollection(models.TenantCollection).RemoveId(tenant.Id)
		log.Println("Error creating owner membership:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create workspace owner"})
		return
	}
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenant.Id, bson.M{"$set": bson.M{"owner_membership_id": membership.Id}}); err != nil {
		_ = db.GetCollection(models.WorkspaceMembershipCollection).RemoveId(membership.Id)
		_ = db.GetCollection(models.AccountUserCollection).RemoveId(accountUser.Id)
		_ = db.GetCollection(models.TenantCollection).RemoveId(tenant.Id)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to establish workspace ownership"})
		return
	}

	// Seed a usable starting point so the dashboard and Advisor aren't empty on
	// day one. Best-effort: a seed failure is logged but never fails signup.
	seedTenantDefaults(tenant.Id)

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

// seedTenantDefaults gives a brand-new workspace a usable starting point: one
// audience list, three contact-label badges (tags), and a ready-to-embed
// lead-capture form already wired to add submitters to the list and tag them
// "Lead". This mirrors what the Advisor's lists_create / badges_create /
// forms_create tools produce, so a fresh tenant (and its Advisor) has something
// to build on instead of an empty workspace. Best-effort — every step is
// non-fatal; a seed failure is logged and registration still succeeds.
func seedTenantDefaults(tenantID bson.ObjectId) {
	now := time.Now()
	tenantHex := tenantID.Hex()

	list := &models.EmailList{
		Id:           bson.NewObjectId(),
		PublicId:     utils.GeneratePublicId(),
		SubscriberId: tenantHex,
		Name:         "All Subscribers",
	}
	list.CreatedAt = &now
	if err := db.GetCollection(models.EmailListCollection).Insert(list); err != nil {
		log.Printf("seedTenantDefaults: list insert failed for %s: %v", tenantHex, err)
		return
	}

	mkBadge := func(name string) *models.Badge {
		b := &models.Badge{
			Id:           bson.NewObjectId(),
			PublicId:     utils.GeneratePublicId(),
			TenantID:     tenantID,
			SubscriberId: tenantHex,
			Name:         name,
			Kind:         models.BadgeKindContactLabel,
		}
		b.CreatedAt = &now
		if err := db.GetCollection(models.BadgeCollection).Insert(b); err != nil {
			log.Printf("seedTenantDefaults: badge %q insert failed for %s: %v", name, tenantHex, err)
		}
		return b
	}
	leadBadge := mkBadge("Lead")
	mkBadge("Customer")
	mkBadge("VIP")

	form := &models.PageForm{
		Id:       bson.NewObjectId(),
		PublicId: utils.GeneratePublicId(),
		TenantID: tenantID,
		Name:     "Newsletter Signup",
		FormType: "lead",
		Fields: []*models.FormField{
			{FieldName: "first_name", FieldType: "text", Required: true, MapsTo: "first_name", Placeholder: "First name"},
			{FieldName: "email", FieldType: "email", Required: true, Placeholder: "you@example.com"},
		},
		OnSubmit: &models.FormOnSubmit{
			UpsertContact:   true,
			WriteAttributes: true,
			AddToListIds:    []string{list.PublicId},
			AssignBadgeIds:  []string{leadBadge.PublicId},
		},
	}
	form.CreatedAt = &now
	if err := db.GetCollection(models.PageFormCollection).Insert(form); err != nil {
		log.Printf("seedTenantDefaults: form insert failed for %s: %v", tenantHex, err)
	}
}

// HandleTenantLogin authenticates an AccountUser and returns a JWT.
func HandleTenantLogin(c *gin.Context) {
	var req struct {
		Email       string `json:"email" binding:"required"`
		Password    string `json:"password" binding:"required"`
		WorkspaceID string `json:"workspace_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email and password are required"})
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	// ID-015: lock out an identity after repeated failures (credential-stuffing).
	if locked, until := auth.LoginLocked("tenant", req.Email); locked {
		auditLogin(c, "auth.login.lockout", "", "identity locked out")
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts; try again after " + until.UTC().Format(time.RFC3339)})
		return
	}

	var accountUser models.AccountUser
	err := db.GetCollection(models.AccountUserCollection).Find(bson.M{"email": req.Email}).One(&accountUser)
	if err != nil {
		auth.RecordLoginFailure("tenant", req.Email)
		auditLogin(c, "auth.login.failure", "", "unknown identity")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	if !auth.CheckPasswordHash(req.Password, accountUser.PasswordHash) {
		auth.RecordLoginFailure("tenant", req.Email)
		auditLoginFor(c, "auth.login.failure", &accountUser, "bad password")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	auth.ClearLoginFailures("tenant", req.Email)

	selectedTenant := accountUser.TenantID
	if req.WorkspaceID != "" {
		if !bson.IsObjectIdHex(req.WorkspaceID) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "workspace_id is invalid"})
			return
		}
		selectedTenant = bson.ObjectIdHex(req.WorkspaceID)
	}
	workspaceUser, ok := auth.UserForWorkspace(&accountUser, selectedTenant)
	if !ok && req.WorkspaceID == "" {
		var membership models.WorkspaceMembership
		if err := db.GetCollection(models.WorkspaceMembershipCollection).Find(bson.M{
			"identity_id": accountUser.Id, "status": models.MembershipActive, "timestamps.deleted_at": nil,
		}).Sort("_id").One(&membership); err == nil {
			workspaceUser, ok = auth.UserForWorkspace(&accountUser, membership.TenantID)
		}
	}
	if !ok {
		auditLoginFor(c, "auth.login.failure", &accountUser, "no active workspace membership")
		c.JSON(http.StatusForbidden, gin.H{"error": "no active membership for that workspace"})
		return
	}
	auditLoginFor(c, "auth.login", workspaceUser, "")

	token, err := auth.GenerateTenantToken(workspaceUser)
	if err != nil {
		log.Println("Error generating token:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	var tenant models.Tenant
	_ = db.GetCollection(models.TenantCollection).FindId(workspaceUser.TenantID).One(&tenant)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"tenant": gin.H{
			"id":            tenant.Id.Hex(),
			"business_name": tenant.BusinessName,
		},
		"user": gin.H{
			"id":    accountUser.Id.Hex(),
			"email": accountUser.Email,
			"role":  workspaceUser.Role,
		},
	})
}

// auditLogin records a semantic auth event with no resolved account (unknown
// identity / lockout). Never carries the attempted email — identity probing
// evidence lives in the lockout counters, not the ledger.
func auditLogin(c *gin.Context, action, actorID, reason string) {
	e := audit.FromContext(c)
	e.Action, e.Reason = action, reason
	e.ActorKind, e.ActorID = "anonymous", actorID
	e.Outcome = "failure"
	if action == "auth.login" {
		e.Outcome = "success"
	}
	audit.Record(e)
}

// auditLoginFor records an auth event attributed to a resolved AccountUser.
func auditLoginFor(c *gin.Context, action string, u *models.AccountUser, reason string) {
	e := audit.FromContext(c)
	e.Action, e.Reason = action, reason
	e.ActorKind, e.ActorID = "human", u.Id.Hex()
	e.TenantID = u.TenantID
	e.Outcome = "failure"
	if action == "auth.login" {
		e.Outcome = "success"
	}
	audit.Record(e)
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

	// ID-015: per-(tenant,email) lockout after repeated failures.
	lockScope := "customer:" + tenantID.Hex()
	if locked, until := auth.LoginLocked(lockScope, req.Email); locked {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "too many attempts; try again after " + until.UTC().Format(time.RFC3339)})
		return
	}

	var contact models.User
	err = db.GetCollection(models.UserCollection).Find(bson.M{
		"email":     models.EmailAddress(req.Email),
		"tenant_id": tenantID,
	}).One(&contact)
	if err != nil {
		auth.RecordLoginFailure(lockScope, req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}

	if !auth.CheckPasswordHash(req.Password, contact.PasswordHash) {
		auth.RecordLoginFailure(lockScope, req.Email)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid email or password"})
		return
	}
	auth.ClearLoginFailures(lockScope, req.Email)

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
		"password_reset_token": auth.HashResetToken(req.Token),
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

	// ID-005: a password change invalidates every previously issued token for
	// this contact (stolen-token recovery is the whole point of the reset).
	if n, rerr := auth.RevokeSessionsForPrincipal(models.AuthSessionKindCustomer, contact.Id.Hex()); rerr == nil && n > 0 {
		log.Printf("customer set-password: revoked %d prior sessions for contact %s", n, contact.Id.Hex())
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

// HandleTenantLogout revokes the current token's session (ID-005). The token
// stops verifying on every service immediately, not at its natural expiry.
func HandleTenantLogout(c *gin.Context) {
	jti := auth.GetJTI(c)
	if jti == "" {
		// Legacy token without a session row — nothing to revoke server-side.
		c.JSON(http.StatusOK, gin.H{"status": "ok", "revoked": 0})
		return
	}
	if err := auth.RevokeSessionByJTI(jti); err != nil {
		log.Printf("tenant logout: revoke jti failed: %v", err)
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "revoked": 1})
}

// HandleTenantLogoutAll revokes every live session of the authenticated
// account user — all devices, all tokens (ID-005 logout-all).
func HandleTenantLogoutAll(c *gin.Context) {
	accountID := auth.GetAccountUserID(c)
	if accountID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	n, err := auth.RevokeSessionsForPrincipal(models.AuthSessionKindTenant, accountID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to revoke sessions"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "revoked": n})
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

	certsDefault := false
	if tenant.CertificatesDefaultEnabled != nil {
		certsDefault = *tenant.CertificatesDefaultEnabled
	}
	c.JSON(http.StatusOK, gin.H{
		"id":                           tenant.Id.Hex(),
		"business_name":                tenant.BusinessName,
		"subscription_status":          tenant.SubscriptionStatus,
		"trial_ends_at":                tenant.TrialEndsAt,
		"stripe_public_key":            tenant.StripePublicKey,
		"has_stripe":                   tenant.StripeSecretKey != "" || tenant.StripeConnectAccountID != "",
		"has_webhook_secret":           tenant.StripeWebhookSecret != "",
		"has_stripe_connect":           tenant.StripeConnectAccountID != "",
		"stripe_connect_account_id":    tenant.StripeConnectAccountID,
		"stripe_webhook_url":           fmt.Sprintf("%s/api/marketing/stripe/webhook?tenant_id=%s", publicAPIBase(), tenant.Id.Hex()),
		"has_mailgun":                  tenant.MailgunAPIKey != "",
		"has_brevo":                    tenant.BrevoAPIKey != "",
		"certificates_default_enabled": certsDefault,
		"postal_address":               tenant.PostalAddress,
		"inbox_auto_respond_enabled":   tenant.InboxAutoRespond(),
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
		BusinessName               string  `json:"business_name"`
		StripeSecretKey            string  `json:"stripe_secret_key"`
		StripePublicKey            string  `json:"stripe_public_key"`
		StripeWebhookSecret        string  `json:"stripe_webhook_secret"`
		MailgunAPIKey              string  `json:"mailgun_api_key"`
		PostalAddress              *string `json:"postal_address,omitempty"`
		MailgunDomain              string  `json:"mailgun_domain"`
		BrevoAPIKey                string  `json:"brevo_api_key"`
		CertificatesDefaultEnabled *bool   `json:"certificates_default_enabled,omitempty"`
		CertificateDefaultTemplate *string `json:"certificate_default_template,omitempty"`
		InboxAutoRespondEnabled    *bool   `json:"inbox_auto_respond_enabled,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	update := bson.M{}
	if req.BusinessName != "" {
		update["business_name"] = req.BusinessName
	}
	if req.PostalAddress != nil {
		update["postal_address"] = strings.TrimSpace(*req.PostalAddress)
	}
	if req.StripeSecretKey != "" {
		enc, err := utils.EncryptSecret(req.StripeSecretKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure stripe key"})
			return
		}
		update["stripe_secret_key"] = enc
	}
	if req.StripePublicKey != "" {
		update["stripe_public_key"] = req.StripePublicKey
	}
	if req.StripeWebhookSecret != "" {
		enc, err := utils.EncryptSecret(req.StripeWebhookSecret)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure webhook secret"})
			return
		}
		update["stripe_webhook_secret"] = enc
	}
	if req.MailgunAPIKey != "" {
		enc, err := utils.EncryptSecret(req.MailgunAPIKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure mailgun key"})
			return
		}
		update["mailgun_api_key"] = enc
	}
	if req.MailgunDomain != "" {
		update["mailgun_domain"] = req.MailgunDomain
	}
	if req.BrevoAPIKey != "" {
		enc, err := utils.EncryptSecret(req.BrevoAPIKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to secure brevo key"})
			return
		}
		update["brevo_api_key"] = enc
	}
	if req.CertificatesDefaultEnabled != nil {
		update["certificates_default_enabled"] = *req.CertificatesDefaultEnabled
	}
	if req.CertificateDefaultTemplate != nil {
		update["certificate_default_template"] = *req.CertificateDefaultTemplate
	}
	if req.InboxAutoRespondEnabled != nil {
		update["inbox_auto_respond_enabled"] = *req.InboxAutoRespondEnabled
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
