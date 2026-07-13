package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

// handleStoryStats returns per-email engagement for one story: one row per
// (storyline_idx, enactment_idx) — i.e. per scene email — aggregated over the
// unified EmailSend rows, plus story-level totals.
func handleStoryStats(c *gin.Context) {
	sid := auth.GetTenantID(c)
	storyID := c.Param("id")

	var story pkgmodels.Story
	if err := db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{"public_id": storyID, "subscriber_id": sid}).One(&story); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "story not found"})
		return
	}

	pipeline := []bson.M{
		{"$match": bson.M{"story_public_id": storyID, "tenant_id": story.TenantID}},
		{"$group": bson.M{
			"_id":     bson.M{"storyline_idx": "$storyline_idx", "enactment_idx": "$enactment_idx"},
			"subject": bson.M{"$last": "$subject"},
			"sent":    bson.M{"$sum": 1},
			"opened":  bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$gt": []interface{}{"$opened_at", nil}}, 1, 0}}},
			"clicked": bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$gt": []interface{}{"$first_clicked_at", nil}}, 1, 0}}},
			"bounced": bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$gt": []interface{}{"$bounced_at", nil}}, 1, 0}}},
			"unsub":   bson.M{"$sum": bson.M{"$cond": []interface{}{bson.M{"$gt": []interface{}{"$unsubscribed_at", nil}}, 1, 0}}},
		}},
		{"$sort": bson.M{"_id.storyline_idx": 1, "_id.enactment_idx": 1}},
	}

	var rows []bson.M
	if err := db.GetCollection(pkgmodels.EmailSendCollection).Pipe(pipeline).All(&rows); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "aggregation failed"})
		return
	}

	emails := make([]gin.H, 0, len(rows))
	totals := gin.H{"sent": 0, "opened": 0, "clicked": 0, "bounced": 0, "unsubscribed": 0}
	add := func(k string, v int) { totals[k] = totals[k].(int) + v }
	for _, r := range rows {
		id, _ := r["_id"].(bson.M)
		toInt := func(v interface{}) int {
			switch n := v.(type) {
			case int:
				return n
			case int64:
				return int(n)
			case float64:
				return int(n)
			}
			return 0
		}
		sent, opened := toInt(r["sent"]), toInt(r["opened"])
		clicked, bounced, unsub := toInt(r["clicked"]), toInt(r["bounced"]), toInt(r["unsub"])
		emails = append(emails, gin.H{
			"storyline_idx": toInt(id["storyline_idx"]),
			"enactment_idx": toInt(id["enactment_idx"]),
			"subject":       r["subject"],
			"sent":          sent,
			"opened":        opened,
			"clicked":       clicked,
			"bounced":       bounced,
			"unsubscribed":  unsub,
		})
		add("sent", sent)
		add("opened", opened)
		add("clicked", clicked)
		add("bounced", bounced)
		add("unsubscribed", unsub)
	}

	c.JSON(http.StatusOK, gin.H{
		"story":  gin.H{"public_id": story.PublicId, "name": story.Name},
		"emails": emails,
		"totals": totals,
	})
}
