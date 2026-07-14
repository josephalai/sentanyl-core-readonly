package routes

import (
	"context"
	"log"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/core-service/internal/billing"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/jobs"
	"github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/plans"
)

// BILL-003 plan-change saga. The intent row is written before the Stripe
// mutation; whichever of these three resolves it first wins, all landing on
// the same authoritative state:
//
//  1. the request handler's local write succeeding (confirmed inline),
//  2. the customer.subscription.updated webhook (confirmPlanIntents),
//  3. the reconciliation sweep polling Stripe for stale pending intents.

// EnsurePlanIntentIndexes creates the plan_change_intents indexes.
func EnsurePlanIntentIndexes() {
	col := db.GetCollection(models.PlanChangeIntentCollection)
	if err := col.EnsureIndex(mgo.Index{Key: []string{"tenant_id", "status"}, Background: true}); err != nil {
		log.Printf("billing: plan intent index: %v", err)
	}
	if err := col.EnsureIndex(mgo.Index{Key: []string{"status", "created_at"}, Background: true}); err != nil {
		log.Printf("billing: plan intent status index: %v", err)
	}
}

// resolvePlanIntent stamps a pending intent's outcome.
func resolvePlanIntent(id bson.ObjectId, status, resolvedTier, note string) {
	now := time.Now()
	if err := db.GetCollection(models.PlanChangeIntentCollection).Update(
		bson.M{"_id": id, "status": models.PlanChangeIntentPending},
		bson.M{"$set": bson.M{"status": status, "resolved_tier": resolvedTier, "note": note, "resolved_at": now}},
	); err != nil && err != mgo.ErrNotFound {
		log.Printf("billing: resolve plan intent %s: %v", id.Hex(), err)
	}
}

// confirmPlanIntents settles every pending intent for a tenant against the
// authoritative tier (from the webhook or reconciliation). Called with the
// tier Stripe actually reports — not what the intent hoped for.
func confirmPlanIntents(tenantID bson.ObjectId, authoritativeTier string) {
	var pending []models.PlanChangeIntent
	if err := db.GetCollection(models.PlanChangeIntentCollection).Find(bson.M{
		"tenant_id": tenantID, "status": models.PlanChangeIntentPending,
	}).All(&pending); err != nil {
		return
	}
	for _, in := range pending {
		resolvePlanIntent(in.Id, models.PlanChangeIntentConfirmed, authoritativeTier, "authoritative state applied")
	}
}

const planIntentSweepJobType = "billing.plan_intents.sweep"

// planIntentStaleAfter is how long an intent may stay pending before the
// sweep polls Stripe directly — normally the handler or the webhook settles
// it within seconds.
const planIntentStaleAfter = 2 * time.Minute

// RegisterPlanIntentSweep starts the reconciliation sweep for missed
// webhooks: stale pending intents are settled against Stripe's actual
// subscription price.
func RegisterPlanIntentSweep() {
	const interval = 5 * time.Minute
	jobs.Register(planIntentSweepJobType, func(ctx context.Context, job *jobs.Job) error {
		if err := jobs.EnqueueSweep(planIntentSweepJobType, time.Now().Add(interval), interval); err != nil {
			return err
		}
		reconcileStalePlanIntents(time.Now().Add(-planIntentStaleAfter))
		if job.RunAt.Unix()%3600 < int64(interval/time.Second) {
			jobs.PruneSucceeded(planIntentSweepJobType, 24*time.Hour)
		}
		return nil
	})
	if err := jobs.EnqueueSweep(planIntentSweepJobType, time.Now(), interval); err != nil {
		log.Printf("billing: bootstrap plan-intent sweep enqueue failed: %v", err)
	}
}

func reconcileStalePlanIntents(olderThan time.Time) {
	key := platformStripeKey()
	if key == "" {
		return // no Stripe in this environment — nothing to reconcile against
	}
	var stale []models.PlanChangeIntent
	if err := db.GetCollection(models.PlanChangeIntentCollection).Find(bson.M{
		"status":     models.PlanChangeIntentPending,
		"created_at": bson.M{"$lte": olderThan},
	}).Limit(50).All(&stale); err != nil {
		return
	}
	for _, in := range stale {
		item, err := billing.GetSubscriptionItem(key, in.StripeSubID)
		if err != nil {
			// Stripe unreachable or subscription gone: after 24h give up and
			// mark failed so operators see it; otherwise retry next sweep.
			if time.Since(in.CreatedAt) > 24*time.Hour {
				resolvePlanIntent(in.Id, models.PlanChangeIntentFailed, "", "stripe lookup failed: "+err.Error())
			}
			continue
		}
		tier, ok := plans.TierForPriceID(item.PriceID)
		if !ok {
			resolvePlanIntent(in.Id, models.PlanChangeIntentFailed, "", "unmapped stripe price "+item.PriceID)
			continue
		}
		if err := db.GetCollection(models.TenantCollection).UpdateId(in.TenantID,
			bson.M{"$set": bson.M{"plan_tier": tier, "plan_contract": plans.SnapshotContract(tier)}}); err != nil {
			log.Printf("billing: reconcile intent %s tenant write: %v", in.Id.Hex(), err)
			continue
		}
		plans.Invalidate(in.TenantID)
		resolvePlanIntent(in.Id, models.PlanChangeIntentConfirmed, tier, "reconciled from stripe")
		log.Printf("billing: plan intent %s reconciled to tier %s (tenant %s)", in.Id.Hex(), tier, in.TenantID.Hex())
	}
}
