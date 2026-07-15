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
		ID       string `json:"id"`
		Type     string `json:"type"`
		Livemode bool   `json:"livemode"`
		Created  int64  `json:"created"`
		Data     struct {
			Object json.RawMessage `json:"object"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rawBody, &evt); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json"})
		return
	}

	// BILL-002: persist the ProviderEvent before processing, keyed by the
	// Stripe event id. A duplicate delivery of an already-processed event is
	// acknowledged without reprocessing (idempotent).
	if evt.ID != "" {
		already, err := recordProviderEvent(evt.ID, evt.Type, evt.Livemode, evt.Created)
		if err != nil {
			// Could not even record the event — ask Stripe to retry (BILL-001).
			log.Printf("[platform webhook] provider-event persist failed: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "event not persisted"})
			return
		}
		if already {
			c.JSON(http.StatusOK, gin.H{"received": true, "duplicate": true})
			return
		}
	}

	switch evt.Type {
	case "checkout.session.completed":
		err = platformCheckoutCompleted(evt.Data.Object)
	case "customer.subscription.created", "customer.subscription.updated":
		err = platformSubscriptionUpdated(evt.Data.Object)
	case "customer.subscription.deleted":
		err = platformSubscriptionDeleted(evt.Data.Object)
	case "invoice.paid", "invoice.payment_succeeded":
		err = platformInvoicePaid(evt.Data.Object)
	case "invoice.payment_failed":
		err = platformInvoicePaymentFailed(evt.Data.Object)
	default:
		// Unhandled event type — nothing to do, acknowledge as processed.
	}

	// BILL-001: a processing failure returns a retryable non-2xx so Stripe
	// redelivers, instead of the old unconditional 200 that dropped the update.
	// The per-type handlers already return nil for permanent/unresolvable cases
	// (e.g. unknown tenant), which are acknowledged as processed.
	if err != nil {
		log.Printf("[platform webhook] %s: %v", evt.Type, err)
		markProviderEventFailed(evt.ID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "processing failed"})
		return
	}
	markProviderEventProcessed(evt.ID)
	c.JSON(http.StatusOK, gin.H{"received": true})
}

// recordProviderEvent inserts (or finds) the ProviderEvent for a Stripe event
// id. It returns already=true when the event has already been processed, so the
// caller can skip reprocessing. A first delivery, or a retry of a previously
// failed event, returns already=false and increments the attempt counter.
func recordProviderEvent(eventID, eventType string, livemode bool, created int64) (already bool, err error) {
	col := db.GetCollection(models.ProviderEventCollection)
	now := time.Now().UTC()
	insertErr := col.Insert(&models.ProviderEvent{
		Id:              bson.NewObjectId(),
		Provider:        "stripe_platform",
		EventID:         eventID,
		Type:            eventType,
		Livemode:        livemode,
		ProviderCreated: created,
		Status:          models.ProviderEventReceived,
		Attempts:        1,
		CreatedAt:       now,
		UpdatedAt:       now,
	})
	if insertErr == nil {
		return false, nil
	}
	if !mgo.IsDup(insertErr) {
		return false, insertErr
	}
	// Already seen: processed events are idempotently skipped; a prior failure
	// is retried (attempt count bumped).
	var existing models.ProviderEvent
	if err := col.Find(bson.M{"provider": "stripe_platform", "event_id": eventID}).One(&existing); err != nil {
		return false, err
	}
	if existing.Status == models.ProviderEventProcessed {
		return true, nil
	}
	_ = col.UpdateId(existing.Id, bson.M{"$inc": bson.M{"attempts": 1}, "$set": bson.M{"updated_at": now}})
	return false, nil
}

func markProviderEventProcessed(eventID string) {
	if eventID == "" {
		return
	}
	_ = db.GetCollection(models.ProviderEventCollection).Update(
		bson.M{"provider": "stripe_platform", "event_id": eventID},
		bson.M{"$set": bson.M{"status": models.ProviderEventProcessed, "last_error": "", "updated_at": time.Now().UTC()}},
	)
}

func markProviderEventFailed(eventID string, cause error) {
	if eventID == "" {
		return
	}
	_ = db.GetCollection(models.ProviderEventCollection).Update(
		bson.M{"provider": "stripe_platform", "event_id": eventID},
		bson.M{"$set": bson.M{"status": models.ProviderEventFailed, "last_error": cause.Error(), "updated_at": time.Now().UTC()}},
	)
}

// EnsurePlatformWebhookIndexes creates the unique ProviderEvent index.
func EnsurePlatformWebhookIndexes() {
	if err := db.GetCollection(models.ProviderEventCollection).EnsureIndex(mgo.Index{
		Key:        []string{"provider", "event_id"},
		Unique:     true,
		Background: true,
	}); err != nil {
		log.Printf("[platform webhook] failed to ensure provider_event index: %v", err)
	}
	// BILL-012: one projection row per (tenant, invoice).
	if err := db.GetCollection(models.InvoiceProjectionCollection).EnsureIndex(mgo.Index{
		Key:        []string{"tenant_id", "stripe_invoice_id"},
		Unique:     true,
		Background: true,
	}); err != nil {
		log.Printf("[platform webhook] failed to ensure invoice_projection index: %v", err)
	}
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
		ID                string            `json:"id"`
		Status            string            `json:"status"`
		Customer          string            `json:"customer"`
		Metadata          map[string]string `json:"metadata"`
		CancelAtPeriodEnd bool              `json:"cancel_at_period_end"`
		CurrentPeriodEnd  int64             `json:"current_period_end"`
		Items             struct {
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
	if sub.CancelAtPeriodEnd && sub.CurrentPeriodEnd > 0 {
		set["billing_change_kind"] = models.BillingChangeCancel
		set["billing_change_effective_at"] = time.Unix(sub.CurrentPeriodEnd, 0).UTC()
		unset["pending_plan_tier"] = ""
	} else {
		var current models.Tenant
		if db.GetCollection(models.TenantCollection).FindId(tenantID).Select(bson.M{"billing_change_kind": 1}).One(&current) == nil && current.BillingChangeKind == models.BillingChangeCancel {
			unset["billing_change_kind"] = ""
			unset["billing_change_effective_at"] = ""
			unset["pending_plan_tier"] = ""
			markScheduledIntents(tenantID, models.BillingChangeCancel, models.PlanChangeIntentCanceled, "cancellation reversed at Stripe")
		}
	}
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

	// BILL-011: freeze the tier's terms as an immutable contract whenever the
	// tier is (re)set, so later price-table edits never change this
	// subscriber's promises.
	if newTier, ok := set["plan_tier"].(string); ok {
		set["plan_contract"] = plans.SnapshotContract(newTier)
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
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID, update); err != nil {
		return err
	}
	// BILL-003: authoritative subscription state arrived — settle any pending
	// plan-change intents against the tier Stripe actually reports.
	if tier, ok := set["plan_tier"].(string); ok {
		confirmPlanIntents(tenantID, tier)
		plans.Invalidate(tenantID)
	}
	return nil
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
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID, bson.M{
		"$set":   bson.M{"subscription_status": "canceled"},
		"$unset": bson.M{"billing_change_kind": "", "billing_change_effective_at": "", "pending_plan_tier": ""},
	}); err != nil {
		return err
	}
	markScheduledIntents(tenantID, models.BillingChangeCancel, models.PlanChangeIntentConfirmed, "subscription ended at paid-through boundary")
	return nil
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
	upsertInvoiceProjection(tenantID, raw, "payment_failed")
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID,
		bson.M{"$set": bson.M{"subscription_status": "past_due"}}); err != nil {
		return err
	}
	return stampPastDueIfUnset(tenantID)
}

// platformInvoicePaid projects a successful platform invoice (BILL-012) so the
// tenant has a durable billing history independent of live Stripe reads.
func platformInvoicePaid(raw json.RawMessage) error {
	var inv struct {
		Customer string `json:"customer"`
	}
	if err := json.Unmarshal(raw, &inv); err != nil {
		return err
	}
	tenantID, ok := resolvePlatformTenant("", inv.Customer)
	if !ok {
		return nil
	}
	upsertInvoiceProjection(tenantID, raw, "paid")
	return nil
}

// upsertInvoiceProjection writes/updates one immutable invoice row from a
// Stripe invoice payload. Idempotent on (tenant, stripe_invoice_id).
func upsertInvoiceProjection(tenantID bson.ObjectId, raw json.RawMessage, status string) {
	var inv struct {
		ID               string `json:"id"`
		Number           string `json:"number"`
		AmountDue        int64  `json:"amount_due"`
		AmountPaid       int64  `json:"amount_paid"`
		Currency         string `json:"currency"`
		HostedInvoiceURL string `json:"hosted_invoice_url"`
		PeriodStart      int64  `json:"period_start"`
		PeriodEnd        int64  `json:"period_end"`
	}
	if err := json.Unmarshal(raw, &inv); err != nil || inv.ID == "" {
		return
	}
	now := time.Now().UTC()
	set := bson.M{
		"tenant_id":          tenantID,
		"stripe_invoice_id":  inv.ID,
		"number":             inv.Number,
		"status":             status,
		"amount_due_minor":   inv.AmountDue,
		"amount_paid_minor":  inv.AmountPaid,
		"currency":           inv.Currency,
		"hosted_invoice_url": inv.HostedInvoiceURL,
		"updated_at":         now,
	}
	if inv.PeriodStart > 0 {
		ps := time.Unix(inv.PeriodStart, 0).UTC()
		set["period_start"] = ps
	}
	if inv.PeriodEnd > 0 {
		pe := time.Unix(inv.PeriodEnd, 0).UTC()
		set["period_end"] = pe
	}
	_, _ = db.GetCollection(models.InvoiceProjectionCollection).Upsert(
		bson.M{"tenant_id": tenantID, "stripe_invoice_id": inv.ID},
		bson.M{"$set": set, "$setOnInsert": bson.M{"created_at": now}},
	)
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
