package routes

import (
	"log"
	"net/http"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/models"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"
)

// HandleTenantResetAllData permanently removes ALL data for the authenticated
// tenant. This is a development/testing convenience — it wipes every collection
// scoped to the tenant's subscriber_id or tenant_id.
func HandleTenantResetAllData(c *gin.Context) {
	tid := auth.GetTenantID(c)
	if tid == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not authenticated"})
		return
	}
	tenantOID := auth.GetTenantObjectID(c)

	subscriberCollections := []models.MGCollection{
		models.StoryCollection,
		models.StorylineCollection,
		models.EnactmentCollection,
		models.SceneCollection,
		models.MessageCollection,
		models.MessageContentCollection,
		models.TriggerCollection,
		models.HotTriggerCollection,
		models.ActionCollection,
		models.BadgeCollection,
		models.BadgeTransactionCollection,
		models.TagCollection,
		models.TemplateCollection,
		models.TemplateVariablesCollection,
		models.UserCollection,
		models.UserBadgeCollection,
		models.InstantEmailCollection,
		models.ScheduledEmailCollection,
		models.SentEmailCollection,
		models.OutboundWebhookCollection,
		models.CreatorCollection,
		models.EmailListCollection,
		models.SendingDomainCollection,
	}

	tenantCollections := []models.MGCollection{
		models.FunnelCollection,
		models.FunnelRouteCollection,
		models.FunnelStageCollection,
		models.FunnelPageCollection,
		models.PageBlockCollection,
		models.PageFormCollection,
		models.ProductCollection,
		models.PurchaseLogCollection,
		models.FunnelTemplateCollection,
		models.OfferCollection,
		models.CouponCollection,
		models.SubscriptionCollection,
		models.SiteCollection,
		models.AssetCollection,
		models.MediaCollection,
		models.PlayerPresetCollection,
		models.MediaChannelCollection,
		models.MediaWebhookCollection,
		models.ViewerIdentityCollection,
		models.ViewingSessionCollection,
		models.MediaEventCollection,
		models.MediaLeadCaptureCollection,
		models.MediaDailyAggregateCollection,
		models.CourseEnrollmentCollection,
		models.LessonCompletionCollection,
		models.CertificateCollection,
		models.LMSQuizCollection,
		models.QuizAttemptCollection,
	}

	totalRemoved := 0

	for _, col := range subscriberCollections {
		info, err := db.GetCollection(col).RemoveAll(bson.M{"subscriber_id": tid})
		if err != nil {
			log.Printf("resetAllData: error clearing %s by subscriber_id: %v", col, err)
		} else if info.Removed > 0 {
			log.Printf("resetAllData: removed %d from %s (subscriber_id)", info.Removed, col)
			totalRemoved += info.Removed
		}
	}

	for _, col := range tenantCollections {
		info, err := db.GetCollection(col).RemoveAll(bson.M{"tenant_id": tenantOID})
		if err != nil {
			log.Printf("resetAllData: error clearing %s by tenant_id: %v", col, err)
		} else if info.Removed > 0 {
			log.Printf("resetAllData: removed %d from %s (tenant_id)", info.Removed, col)
			totalRemoved += info.Removed
		}
		info2, _ := db.GetCollection(col).RemoveAll(bson.M{"subscriber_id": tid})
		if info2 != nil && info2.Removed > 0 {
			log.Printf("resetAllData: removed %d more from %s (subscriber_id)", info2.Removed, col)
			totalRemoved += info2.Removed
		}
	}

	log.Printf("resetAllData: tenant %s — total %d documents removed", tid, totalRemoved)
	c.JSON(http.StatusOK, gin.H{
		"status":        "ok",
		"total_removed": totalRemoved,
	})
}
