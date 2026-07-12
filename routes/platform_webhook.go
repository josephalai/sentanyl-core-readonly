package routes

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	httputil "github.com/josephalai/sentanyl/pkg/http"
	"github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/plans"
)

// HandlePlatformStripeWebhook receives Stripe events for PLATFORM billing
// (the subscription each tenant pays Sentanyl) and syncs them onto
// Tenant.SubscriptionStatus. Public route; auth is the Stripe signature
// verified against STRIPE_PLATFORM_WEBHOOK_SECRET. Distinct from the
// per-tenant webhook in marketing-service (which serves tenants' own sales).
func HandlePlatformStripeWebhook(c *gin.Context) {
	secret := os.Getenv("STRIPE_PLATFORM_WEBHOOK_SECRET")
	if secret == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "platform webhook not configured"})
		return
	}

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
		return
	}
	if err := httputil.VerifyStripeSignature(c.GetHeader("Stripe-Signature"), rawBody, secret); err != nil {
		log.Printf("[platform webhook] signature verify failed: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signature"})
		return
	}

	var evt struct {
		Type string `json:"type"`
		Data struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &evt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	switch evt.Type {
	case "checkout.session.completed":
		err = platformCheckoutCompleted(evt.Data.Object)
	case "customer.subscription.created", "customer.subscription.updated":
		err = platformSubscriptionUpdated(evt.Data.Object)
	case "customer.subscription.deleted":
		err = platformSubscriptionDeleted(evt.Data.Object)
	case "invoice.payment_failed":
		err = platformInvoicePaymentFailed(evt.Data.Object)
	default:
		// Acknowledge unhandled events so Stripe stops retrying.
	}
	if err != nil {
		log.Printf("[platform webhook] %s: %v", evt.Type, err)
	}
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// resolvePlatformTenant finds the tenant a platform event belongs to, via
// explicit tenant metadata first, then the stored Stripe customer ID.
func resolvePlatformTenant(tenantIDHex, customerID string) (bson.ObjectId, bool) {
	if bson.IsObjectIdHex(tenantIDHex) {
		return bson.ObjectIdHex(tenantIDHex), true
	}
	if customerID != "" {
		var tenant models.Tenant
		if err := db.GetCollection(models.TenantCollection).
			Find(bson.M{"platform_stripe_customer_id": customerID}).
			Select(bson.M{"_id": 1}).One(&tenant); err == nil {
			return tenant.Id, true
		}
	}
	return "", false
}

func platformCheckoutCompleted(raw json.RawMessage) error {
	var session struct {
		Mode              string            `json:"mode"`
		ClientReferenceID string            `json:"client_reference_id"`
		Customer          string            `json:"customer"`
		Subscription      string            `json:"subscription"`
		Metadata          map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &session); err != nil {
		return err
	}
	if session.Mode != "subscription" {
		return nil
	}
	tenantID, ok := resolvePlatformTenant(firstNonEmpty(session.ClientReferenceID, session.Metadata["tenant_id"]), session.Customer)
	if !ok {
		log.Printf("[platform webhook] checkout.session.completed: cannot resolve tenant (customer=%s)", session.Customer)
		return nil
	}

	// The subscription may still be trialing (trial_end preserved from signup);
	// keep status "trial" in that case — subscription.updated flips it later.
	status := "active"
	var tenant models.Tenant
	if err := db.GetCollection(models.TenantCollection).FindId(tenantID).
		Select(bson.M{"trial_ends_at": 1}).One(&tenant); err == nil {
		if tenant.TrialEndsAt != nil && time.Now().Before(*tenant.TrialEndsAt) {
			status = "trial"
		}
	}

	return db.GetCollection(models.TenantCollection).UpdateId(tenantID, bson.M{
		"$set": bson.M{
			"platform_stripe_customer_id": session.Customer,
			"platform_subscription_id":    session.Subscription,
			"subscription_status":         status,
		},
		"$unset": bson.M{"past_due_at": ""},
	})
}

func platformSubscriptionUpdated(raw json.RawMessage) error {
	var sub struct {
		ID       string            `json:"id"`
		Status   string            `json:"status"`
		Customer string            `json:"customer"`
		Metadata map[string]string `json:"metadata"`
		Items    struct {
			Data []struct {
				Price struct {
					ID string `json:"id"`
				} `json:"price"`
			} `json:"data"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &sub); err != nil {
		return err
	}
	tenantID, ok := resolvePlatformTenant(sub.Metadata["tenant_id"], sub.Customer)
	if !ok {
		log.Printf("[platform webhook] subscription.updated: cannot resolve tenant (customer=%s)", sub.Customer)
		return nil
	}

	set := bson.M{"platform_subscription_id": sub.ID}
	// Sync the plan tier: the item's Price is authoritative (it's what the
	// tenant pays); subscription metadata is the fallback for prices that
	// aren't in this deployment's env mapping.
	if len(sub.Items.Data) > 0 {
		if tier, ok := plans.TierForPriceID(sub.Items.Data[0].Price.ID); ok {
			set["plan_tier"] = tier
		} else if _, ok := plans.ByTier(sub.Metadata["plan_tier"]); ok {
			set["plan_tier"] = sub.Metadata["plan_tier"]
		}
	} else if _, ok := plans.ByTier(sub.Metadata["plan_tier"]); ok {
		set["plan_tier"] = sub.Metadata["plan_tier"]
	}
	unset := bson.M{}
	switch sub.Status {
	case "trialing":
		set["subscription_status"] = "trial"
		unset["past_due_at"] = ""
	case "active":
		set["subscription_status"] = "active"
		unset["past_due_at"] = ""
	case "past_due":
		set["subscription_status"] = "past_due"
		if err := stampPastDueIfUnset(tenantID); err != nil {
			return err
		}
	case "canceled", "unpaid", "incomplete_expired":
		set["subscription_status"] = "canceled"
	default: // incomplete — no change
		return nil
	}

	// On a tier upgrade, release limit-held contacts and restart the limit
	// clock — covers checkout-driven upgrades that bypass /billing/change-plan.
	if newTier, ok := set["plan_tier"].(string); ok {
		var current models.Tenant
		if err := db.GetCollection(models.TenantCollection).FindId(tenantID).
			Select(bson.M{"plan_tier": 1}).One(&current); err == nil {
			if plans.ForTenant(newTier).ContactLimit > plans.ForTenant(current.PlanTier).ContactLimit {
				plans.ReleaseHeldContacts(tenantID)
				unset["limit_grace_started_at"] = ""
			}
		}
	}

	update := bson.M{"$set": set}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	return db.GetCollection(models.TenantCollection).UpdateId(tenantID, update)
}

func platformSubscriptionDeleted(raw json.RawMessage) error {
	var sub struct {
		Customer string            `json:"customer"`
		Metadata map[string]string `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &sub); err != nil {
		return err
	}
	tenantID, ok := resolvePlatformTenant(sub.Metadata["tenant_id"], sub.Customer)
	if !ok {
		return nil
	}
	return db.GetCollection(models.TenantCollection).UpdateId(tenantID,
		bson.M{"$set": bson.M{"subscription_status": "canceled"}})
}

func platformInvoicePaymentFailed(raw json.RawMessage) error {
	var invoice struct {
		Customer string `json:"customer"`
	}
	if err := json.Unmarshal(raw, &invoice); err != nil {
		return err
	}
	tenantID, ok := resolvePlatformTenant("", invoice.Customer)
	if !ok {
		return nil
	}
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID,
		bson.M{"$set": bson.M{"subscription_status": "past_due"}}); err != nil {
		return err
	}
	return stampPastDueIfUnset(tenantID)
}

// stampPastDueIfUnset records when the tenant first went past_due so the
// grace-period clock doesn't reset on every Stripe retry event.
func stampPastDueIfUnset(tenantID bson.ObjectId) error {
	err := db.GetCollection(models.TenantCollection).Update(
		bson.M{"_id": tenantID, "past_due_at": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"past_due_at": time.Now().UTC()}},
	)
	if err == mgo.ErrNotFound {
		return nil // already stamped
	}
	return err
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
