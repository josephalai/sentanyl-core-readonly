package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/plans"
	"github.com/josephalai/sentanyl/pkg/utils"
)

// HandleTestSetBilling is an e2e-only fixture endpoint (registered solely
// under SENTANYL_E2E_MODE) that writes a tenant's platform-billing state so
// lifecycle flow 16 can walk trial→expired→past_due→active without Stripe.
// Offsets are hours relative to now; negative values put the timestamp in the
// past. Omitted (nil) offsets $unset the field.
func HandleTestSetBilling(c *gin.Context) {
	var req struct {
		TenantID              string   `json:"tenant_id"`
		SubscriptionStatus    string   `json:"subscription_status"`
		TrialEndsOffsetHours  *float64 `json:"trial_ends_offset_hours"`
		PastDueAtOffsetHours  *float64 `json:"past_due_at_offset_hours"`
		// Tier-limit fixtures (flow 16 tier walk):
		PlanTier              string   `json:"plan_tier"`
		LimitGraceOffsetHours *float64 `json:"limit_grace_offset_hours"`
		SeedContacts          int      `json:"seed_contacts"`
		// PlatformSubscriptionID lets the downgrade-refusal step pass the
		// "no active subscription" guard without a real Stripe subscription
		// (the 409 usage check fires before any Stripe API call).
		PlatformSubscriptionID string `json:"platform_subscription_id"`
		// ReleaseHeld invokes the same plans.ReleaseHeldContacts an upgrade
		// runs, so the flow can prove held-contact release without Stripe.
		ReleaseHeld bool `json:"release_held"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || !bson.IsObjectIdHex(req.TenantID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id (hex) required"})
		return
	}
	tenantID := bson.ObjectIdHex(req.TenantID)

	set := bson.M{}
	unset := bson.M{}
	if req.SubscriptionStatus != "" {
		set["subscription_status"] = req.SubscriptionStatus
	}
	if req.TrialEndsOffsetHours != nil {
		set["trial_ends_at"] = time.Now().UTC().Add(time.Duration(*req.TrialEndsOffsetHours * float64(time.Hour)))
	} else {
		unset["trial_ends_at"] = ""
	}
	if req.PastDueAtOffsetHours != nil {
		set["past_due_at"] = time.Now().UTC().Add(time.Duration(*req.PastDueAtOffsetHours * float64(time.Hour)))
	} else {
		unset["past_due_at"] = ""
	}
	if req.PlanTier != "" {
		set["plan_tier"] = req.PlanTier
	}
	if req.PlatformSubscriptionID != "" {
		set["platform_subscription_id"] = req.PlatformSubscriptionID
	}
	if req.LimitGraceOffsetHours != nil {
		set["limit_grace_started_at"] = time.Now().UTC().Add(time.Duration(*req.LimitGraceOffsetHours * float64(time.Hour)))
	} else {
		unset["limit_grace_started_at"] = ""
		unset["limit_notified_state"] = ""
	}

	update := bson.M{}
	if len(set) > 0 {
		update["$set"] = set
	}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	if err := db.GetCollection(models.TenantCollection).UpdateId(tenantID, update); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Bulk-seed subscribed contacts so flows can cross tier limits without
	// thousands of API calls.
	if req.SeedContacts > 0 {
		docs := make([]interface{}, 0, req.SeedContacts)
		now := time.Now()
		batch := time.Now().UnixNano()
		for i := 0; i < req.SeedContacts; i++ {
			u := models.User{
				Id:           bson.NewObjectId(),
				PublicId:     utils.GeneratePublicId(),
				TenantID:     tenantID,
				SubscriberId: req.TenantID,
				Email:        models.EmailAddress(fmt.Sprintf("seed-%d-%d@limit.test", batch, i)),
			}
			u.Subscribed = true
			u.SoftDeletes.CreatedAt = &now
			docs = append(docs, u)
		}
		if err := db.GetCollection(models.UserCollection).Insert(docs...); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "seed contacts: " + err.Error()})
			return
		}
	}
	released := 0
	if req.ReleaseHeld {
		released = plans.ReleaseHeldContacts(tenantID)
	}
	plans.Invalidate(tenantID)
	c.JSON(http.StatusOK, gin.H{"status": "ok", "released": released})
}
