package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"
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
	}
	if err := c.ShouldBindJSON(&req); err != nil || !bson.IsObjectIdHex(req.TenantID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id (hex) required"})
		return
	}

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

	update := bson.M{}
	if len(set) > 0 {
		update["$set"] = set
	}
	if len(unset) > 0 {
		update["$unset"] = unset
	}
	if err := db.GetCollection(models.TenantCollection).UpdateId(bson.ObjectIdHex(req.TenantID), update); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
