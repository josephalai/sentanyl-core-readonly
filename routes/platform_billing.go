package routes

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/core-service/internal/billing"
	"github.com/josephalai/sentanyl/pkg/audit"
	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/plans"
)

// Platform billing endpoints — charging the tenant for Sentanyl itself via
// Stripe Billing on the platform account. These must be registered on the
// UNGATED tenant group so an unpaid tenant can still reach them to pay.

func platformStripeKey() string { return os.Getenv("STRIPE_PLATFORM_SECRET_KEY") }

// HandleListBillingPlans returns the public tier catalog, with each tier's
// availability (a tier is offered only when its Stripe Price is configured).
func HandleListBillingPlans(c *gin.Context) {
	type planOut struct {
		plans.Plan
		Available bool `json:"available"`
	}
	out := make([]planOut, 0, len(plans.All))
	for _, p := range plans.All {
		out = append(out, planOut{Plan: p, Available: plans.StripePriceID(p.Tier) != ""})
	}
	c.JSON(http.StatusOK, gin.H{"plans": out})
}

// HandleGetBillingStatus returns the tenant's platform subscription state,
// computed with the same SubscriptionAllowed logic the enforcement middleware
// uses so the dashboard and the gate can never disagree.
func HandleGetBillingStatus(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var tenant models.Tenant
	if err := db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	now := time.Now()
	daysLeft := 0
	if tenant.SubscriptionStatus == "trial" && tenant.TrialEndsAt != nil {
		if d := tenant.TrialEndsAt.Sub(now); d > 0 {
			daysLeft = int(d.Hours()/24) + 1
		}
	}
	resp := gin.H{
		"subscription_status": tenant.SubscriptionStatus,
		"trial_ends_at":       tenant.TrialEndsAt,
		"past_due_at":         tenant.PastDueAt,
		"has_subscription":    tenant.PlatformSubscriptionID != "",
		"gated":               !auth.SubscriptionAllowed(&tenant, now),
		"days_left":           daysLeft,
	}
	if tenant.BillingChangeKind != "" && tenant.BillingChangeEffectiveAt != nil {
		resp["pending_change"] = gin.H{
			"kind": tenant.BillingChangeKind, "to_tier": tenant.PendingPlanTier,
			"effective_at": tenant.BillingChangeEffectiveAt,
		}
	}
	if limits, err := plans.ContactLimitStatus(&tenant, now); err == nil {
		resp["plan_tier"] = limits.Plan.Tier
		resp["plan"] = limits.Plan
		resp["usage"] = limits.Usage
		resp["limit_state"] = limits.State
		if limits.GraceEndsAt != nil {
			resp["grace_ends_at"] = limits.GraceEndsAt
		}
	} else {
		log.Printf("[platform billing] limit status: %v", err)
		resp["plan_tier"] = plans.ForTenant(tenant.PlanTier).Tier
		resp["plan"] = plans.ForTenant(tenant.PlanTier)
	}
	c.JSON(http.StatusOK, resp)
}

// HandleCreateBillingCheckoutSession lazily creates the platform Stripe
// customer, then returns a hosted Checkout URL (subscription mode) that
// preserves whatever trial time the tenant has left.
func HandleCreateBillingCheckoutSession(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body struct {
		Tier string `json:"tier"`
	}
	_ = c.ShouldBindJSON(&body) // empty body → default tier below

	tier := body.Tier
	if tier == "" {
		tier = plans.DefaultTier
	}
	if _, ok := plans.ByTier(tier); !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown plan tier"})
		return
	}
	key, priceID := platformStripeKey(), plans.StripePriceID(tier)
	if key == "" || priceID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "platform billing is not configured"})
		return
	}

	var tenant models.Tenant
	if err := db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}

	if tenant.PlatformStripeCustomerID == "" {
		email, _ := c.Get(auth.ContextEmail)
		emailStr, _ := email.(string)
		customerID, err := billing.CreateCustomer(key, emailStr, tenant.BusinessName, tenantID.Hex())
		if err != nil {
			log.Printf("[platform billing] create customer: %v", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create billing customer"})
			return
		}
		if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID,
			bson.M{"$set": bson.M{"platform_stripe_customer_id": customerID}}); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save billing customer"})
			return
		}
		tenant.PlatformStripeCustomerID = customerID
	}

	base := publicAPIBase()
	// BILL-008: a stable idempotency key per (tenant, customer, tier) collapses
	// duplicate submits (double-click, retry) into one Stripe Checkout Session
	// instead of minting a fresh one each call.
	idemKey := "checkout:" + tenantID.Hex() + ":" + tenant.PlatformStripeCustomerID + ":" + tier
	url, err := billing.CreateSubscriptionCheckoutSession(
		key, priceID, tenant.PlatformStripeCustomerID, tenantID.Hex(), tier,
		base+"/billing?checkout=success", base+"/billing?checkout=cancel",
		idemKey, tenant.TrialEndsAt,
	)
	if err != nil {
		log.Printf("[platform billing] create checkout session: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create checkout session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// HandleChangeBillingPlan swaps an active subscription to a different tier
// with proration. Upgrades apply immediately; downgrades are refused (409)
// while current usage exceeds the target tier's limits.
func HandleChangeBillingPlan(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var body struct {
		Tier string `json:"tier"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Tier == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tier is required"})
		return
	}
	target, ok := plans.ByTier(body.Tier)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown plan tier"})
		return
	}
	key, priceID := platformStripeKey(), plans.StripePriceID(target.Tier)
	if key == "" || priceID == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "this plan is not available yet"})
		return
	}

	var tenant models.Tenant
	if err := db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	if tenant.PlatformSubscriptionID == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "no active subscription — subscribe first"})
		return
	}
	current := plans.ForTenant(tenant.PlanTier)
	if current.Tier == target.Tier {
		c.JSON(http.StatusOK, gin.H{"plan_tier": current.Tier, "changed": false})
		return
	}
	if tenant.BillingChangeKind != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "a billing change is already scheduled"})
		return
	}

	// Downgrades must fit: refuse while usage exceeds the target's limits so a
	// tenant can't dodge enforcement by paying less than their list costs.
	if target.PriceUSD < current.PriceUSD {
		usage, err := plans.GetUsage(tenantID)
		if err == nil && (usage.Contacts > target.ContactLimit || usage.Domains > target.DomainLimit) {
			c.JSON(http.StatusConflict, gin.H{
				"error": "current usage exceeds the limits of that plan",
				"usage": usage,
				"plan":  target,
			})
			return
		}
	}

	item, err := billing.GetSubscriptionItem(key, tenant.PlatformSubscriptionID)
	if err != nil {
		log.Printf("[platform billing] get subscription item: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to load subscription"})
		return
	}
	if target.PriceUSD < current.PriceUSD {
		if item.CurrentPeriodEnd.IsZero() || !item.CurrentPeriodEnd.After(time.Now()) {
			c.JSON(http.StatusBadGateway, gin.H{"error": "subscription has no future billing boundary"})
			return
		}
		intent := models.NewScheduledBillingChange(tenantID, models.BillingChangePlan, current.Tier, target.Tier, tenant.PlatformSubscriptionID, item.CurrentPeriodEnd)
		if err := projectBillingChange(tenantID, models.BillingChangePlan, target.Tier, item.CurrentPeriodEnd); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "a billing change is already scheduled"})
			return
		}
		if err := db.GetCollection(models.PlanChangeIntentCollection).Insert(intent); err != nil {
			_ = clearBillingChange(tenantID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record scheduled downgrade"})
			return
		}
		recordBillingLifecycleAudit(c, "billing.plan.downgrade.schedule", target.Tier, item.CurrentPeriodEnd)
		c.JSON(http.StatusAccepted, gin.H{"plan_tier": current.Tier, "to_tier": target.Tier, "scheduled": true, "effective_at": item.CurrentPeriodEnd})
		return
	}
	// BILL-003: persist the intent BEFORE the Stripe mutation. If Stripe
	// succeeds but the local write fails, the pending intent guarantees the
	// webhook or the reconciliation sweep settles the tier — never silent
	// divergence between what the tenant pays and what they're recorded as.
	intent := models.NewPlanChangeIntent(tenantID, current.Tier, target.Tier, tenant.PlatformSubscriptionID)
	if err := db.GetCollection(models.PlanChangeIntentCollection).Insert(intent); err != nil {
		log.Printf("[platform billing] insert plan intent: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record plan change"})
		return
	}
	if err := billing.UpdateSubscriptionPrice(key, tenant.PlatformSubscriptionID, item.ItemID, priceID); err != nil {
		log.Printf("[platform billing] update subscription price: %v", err)
		resolvePlanIntent(intent.Id, models.PlanChangeIntentFailed, "", "stripe update failed: "+err.Error())
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to change plan"})
		return
	}
	// Set the tier immediately; the subscription.updated webhook confirms it.
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID,
		bson.M{"$set": bson.M{"plan_tier": target.Tier}}); err != nil {
		// Intent stays pending — the webhook/sweep repairs the local tier.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "plan changed in Stripe but failed to save — it will sync shortly"})
		return
	}
	resolvePlanIntent(intent.Id, models.PlanChangeIntentConfirmed, target.Tier, "applied inline")
	plans.Invalidate(tenantID)
	released := 0
	if target.PriceUSD > current.PriceUSD {
		released = plans.ReleaseHeldContacts(tenantID)
		// Fresh headroom: restart the limit clock at the new tier.
		_ = db.GetCollection(models.TenantCollection).UpdateId(tenantID,
			bson.M{"$unset": bson.M{"limit_grace_started_at": ""}})
	}
	ae := audit.FromContext(c)
	ae.Action, ae.Outcome = "billing.plan.change", "success"
	ae.TargetType, ae.TargetID = "plan", target.Tier
	audit.Record(ae)
	c.JSON(http.StatusOK, gin.H{"plan_tier": target.Tier, "changed": true, "contacts_released": released})
}

func projectBillingChange(tenantID bson.ObjectId, kind, toTier string, effectiveAt time.Time) error {
	return db.GetCollection(models.TenantCollection).Update(
		bson.M{"_id": tenantID, "$or": []bson.M{{"billing_change_kind": bson.M{"$exists": false}}, {"billing_change_kind": ""}}},
		bson.M{"$set": bson.M{"billing_change_kind": kind, "pending_plan_tier": toTier, "billing_change_effective_at": effectiveAt.UTC()}},
	)
}

func clearBillingChange(tenantID bson.ObjectId) error {
	return db.GetCollection(models.TenantCollection).UpdateId(tenantID, bson.M{"$unset": bson.M{
		"billing_change_kind": "", "pending_plan_tier": "", "billing_change_effective_at": "",
	}})
}

func recordBillingLifecycleAudit(c *gin.Context, action, target string, effectiveAt time.Time) {
	e := audit.FromContext(c)
	e.Action, e.Outcome, e.TargetType, e.TargetID = action, "success", "platform_subscription", target
	e.Meta = bson.M{"effective_at": effectiveAt.UTC()}
	audit.Record(e)
}

// HandleScheduleBillingCancellation keeps access through the paid-through
// period while projecting the exact cancellation date in Sentanyl.
func HandleScheduleBillingCancellation(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	var tenant models.Tenant
	if tenantID == "" || db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if tenant.PlatformSubscriptionID == "" || tenant.BillingChangeKind != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "subscription cannot be scheduled for cancellation"})
		return
	}
	key := platformStripeKey()
	if key == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "platform billing is not configured"})
		return
	}
	item, err := billing.GetSubscriptionItem(key, tenant.PlatformSubscriptionID)
	if err != nil || !item.CurrentPeriodEnd.After(time.Now()) {
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to load subscription billing boundary"})
		return
	}
	intent := models.NewScheduledBillingChange(tenantID, models.BillingChangeCancel, tenant.PlanTier, "", tenant.PlatformSubscriptionID, item.CurrentPeriodEnd)
	if err := projectBillingChange(tenantID, models.BillingChangeCancel, "", item.CurrentPeriodEnd); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "a billing change is already scheduled"})
		return
	}
	if err := db.GetCollection(models.PlanChangeIntentCollection).Insert(intent); err != nil {
		_ = clearBillingChange(tenantID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record cancellation"})
		return
	}
	if err := billing.SetSubscriptionCancelAtPeriodEnd(key, tenant.PlatformSubscriptionID, true); err != nil {
		resolvePlanIntent(intent.Id, models.PlanChangeIntentFailed, "", "stripe cancellation failed: "+err.Error())
		_ = clearBillingChange(tenantID)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to schedule cancellation"})
		return
	}
	recordBillingLifecycleAudit(c, "billing.subscription.cancel.schedule", tenant.PlatformSubscriptionID, item.CurrentPeriodEnd)
	c.JSON(http.StatusAccepted, gin.H{"scheduled": true, "effective_at": item.CurrentPeriodEnd})
}

// HandleReactivateBillingSubscription reverses an end-of-period cancellation.
func HandleReactivateBillingSubscription(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	var tenant models.Tenant
	if tenantID == "" || db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if tenant.BillingChangeKind != models.BillingChangeCancel || tenant.PlatformSubscriptionID == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "no scheduled cancellation to reverse"})
		return
	}
	if platformStripeKey() == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "platform billing is not configured"})
		return
	}
	if err := db.GetCollection(models.TenantCollection).Update(
		bson.M{"_id": tenantID, "billing_change_kind": models.BillingChangeCancel},
		bson.M{"$set": bson.M{"billing_change_kind": models.BillingChangeReactivate}},
	); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "cancellation is already being reversed"})
		return
	}
	if err := billing.SetSubscriptionCancelAtPeriodEnd(platformStripeKey(), tenant.PlatformSubscriptionID, false); err != nil {
		_ = db.GetCollection(models.TenantCollection).UpdateId(tenantID, bson.M{"$set": bson.M{"billing_change_kind": models.BillingChangeCancel}})
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to reactivate subscription"})
		return
	}
	markScheduledIntents(tenantID, models.BillingChangeCancel, models.PlanChangeIntentCanceled, "cancellation reversed")
	intent := models.NewPlanChangeIntent(tenantID, tenant.PlanTier, tenant.PlanTier, tenant.PlatformSubscriptionID)
	intent.Kind = models.BillingChangeReactivate
	intent.Status = models.PlanChangeIntentConfirmed
	now := time.Now().UTC()
	intent.ResolvedAt = &now
	intent.Note = "reactivated inline"
	_ = db.GetCollection(models.PlanChangeIntentCollection).Insert(intent)
	_ = clearBillingChange(tenantID)
	recordBillingLifecycleAudit(c, "billing.subscription.reactivate", tenant.PlatformSubscriptionID, now)
	c.JSON(http.StatusOK, gin.H{"reactivated": true})
}

// HandleCancelScheduledBillingChange withdraws a local future downgrade.
func HandleCancelScheduledBillingChange(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	var tenant models.Tenant
	if tenantID == "" || db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant) != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if tenant.BillingChangeKind != models.BillingChangePlan {
		c.JSON(http.StatusConflict, gin.H{"error": "no scheduled downgrade to cancel"})
		return
	}
	markScheduledIntents(tenantID, models.BillingChangePlan, models.PlanChangeIntentCanceled, "scheduled downgrade withdrawn")
	_ = clearBillingChange(tenantID)
	recordBillingLifecycleAudit(c, "billing.plan.downgrade.cancel", tenant.PendingPlanTier, time.Now())
	c.JSON(http.StatusOK, gin.H{"canceled": true})
}

// HandleCreateBillingPortalSession returns a Stripe Billing Portal URL for
// card updates, invoices, and cancellation.
func HandleCreateBillingPortalSession(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	key := platformStripeKey()
	if key == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "platform billing is not configured"})
		return
	}

	var tenant models.Tenant
	if err := db.GetCollection(models.TenantCollection).FindId(tenantID).One(&tenant); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found"})
		return
	}
	if tenant.PlatformStripeCustomerID == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "no billing customer yet — add a payment method first"})
		return
	}

	url, err := billing.CreatePortalSession(key, tenant.PlatformStripeCustomerID, publicAPIBase()+"/billing")
	if err != nil {
		log.Printf("[platform billing] create portal session: %v", err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "failed to create portal session"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": url})
}

// HandleGetBillingInvoices returns the tenant's projected invoice history
// (BILL-012), newest first — a durable read model independent of live Stripe.
func HandleGetBillingInvoices(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	if tenantID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var invoices []models.InvoiceProjection
	_ = db.GetCollection(models.InvoiceProjectionCollection).
		Find(bson.M{"tenant_id": tenantID}).Sort("-created_at").Limit(100).All(&invoices)
	if invoices == nil {
		invoices = []models.InvoiceProjection{}
	}
	c.JSON(http.StatusOK, gin.H{"invoices": invoices})
}
