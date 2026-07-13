package routes

import (
	"encoding/base64"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

const trackingSeparator = "|"

// encodeTrackingToken encodes a URL + user public-ID (+ optional EmailSend
// public-ID for per-email stats) for use in a tracking link.
func encodeTrackingToken(originalURL, userPublicId, sendPublicId string) string {
	raw := originalURL + trackingSeparator + userPublicId
	if sendPublicId != "" {
		raw += trackingSeparator + sendPublicId
	}
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// decodeTrackingToken reverses encodeTrackingToken. Tokens minted before the
// per-email stats rollout carry no send id — sendPublicId comes back empty.
func decodeTrackingToken(token string) (originalURL, userPublicId, sendPublicId string, ok bool) {
	b, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", "", "", false
	}
	parts := strings.SplitN(string(b), trackingSeparator, 3)
	if len(parts) < 2 {
		return "", "", "", false
	}
	if len(parts) == 3 {
		sendPublicId = parts[2]
	}
	return parts[0], parts[1], sendPublicId, true
}

var hrefRegex = regexp.MustCompile(`(?i)(href=["'])([^"']+)(["'])`)

// RewriteLinksForTracking replaces every href in the HTML body with a tracking
// redirect so clicks are recorded before the user reaches the destination.
// sendPublicId (optional) rides in the token so the click also stamps the
// per-email EmailSend row.
func RewriteLinksForTracking(html, userPublicId, baseURL, sendPublicId string) string {
	if baseURL == "" || userPublicId == "" {
		return html
	}
	return hrefRegex.ReplaceAllStringFunc(html, func(match string) string {
		parts := hrefRegex.FindStringSubmatch(match)
		if len(parts) != 4 {
			return match
		}
		originalURL := parts[2]
		if strings.HasPrefix(originalURL, "mailto:") || strings.HasPrefix(originalURL, "#") {
			return match
		}
		token := encodeTrackingToken(originalURL, userPublicId, sendPublicId)
		trackingURL := strings.TrimRight(baseURL, "/") + "/api/track/click/" + token
		return parts[1] + trackingURL + parts[3]
	})
}

// RegisterTrackingRoutes wires click tracking.
func RegisterTrackingRoutes(r *gin.Engine) {
	r.GET("/api/track/click/:token", handleClickTracking)
}

func handleClickTracking(c *gin.Context) {
	token := c.Param("token")

	originalURL, userPublicId, sendPublicId, ok := decodeTrackingToken(token)
	if !ok {
		log.Printf("click tracking: invalid token: %s", token)
		c.Redirect(http.StatusFound, "/")
		return
	}

	log.Printf("click tracking: url=%s user=%s send=%s", originalURL, userPublicId, sendPublicId)

	// Stamp the per-email tracking row (counter, not control flow).
	if sendPublicId != "" {
		go stampEmailSendClick(sendPublicId, originalURL)
	}

	// Fire story click triggers asynchronously — don't block the redirect.
	go fireClickTrigger(userPublicId, originalURL)

	c.Redirect(http.StatusFound, originalURL)
}

// stampEmailSendClick records a click on the unified EmailSend row.
func stampEmailSendClick(sendPublicId, url string) {
	now := time.Now()
	update := bson.M{
		"$push": bson.M{"click_events": pkgmodels.EmailClickEvent{URL: url, At: now}},
	}
	// Set first_clicked_at only once.
	var row pkgmodels.EmailSend
	if err := db.GetCollection(pkgmodels.EmailSendCollection).Find(bson.M{"public_id": sendPublicId}).One(&row); err != nil {
		return
	}
	if row.FirstClickedAt == nil {
		update["$set"] = bson.M{"first_clicked_at": now}
	}
	if err := db.GetCollection(pkgmodels.EmailSendCollection).Update(bson.M{"public_id": sendPublicId}, update); err != nil {
		log.Printf("click tracking: email send stamp failed for %s: %v", sendPublicId, err)
	}
}

// fireClickTrigger checks if the user's active story session has an OnClick
// trigger for the clicked URL. If so, marks the enactment complete and
// advances the story.
func fireClickTrigger(userPublicId, clickedURL string) {
	// Find active session for this user.
	var session StorySession
	if err := db.GetCollection(storySessionCollection).Find(bson.M{
		"user_public_id": userPublicId,
		"status":         "active",
	}).One(&session); err != nil {
		return // No active session — nothing to do.
	}

	// Load the story. Script-deployed stories reference storylines by id —
	// hydrate so OnClick triggers fire for them too.
	var story pkgmodels.Story
	if err := db.GetCollection(pkgmodels.StoryCollection).FindId(session.StoryId).One(&story); err != nil {
		return
	}
	hydrateStoryGraph(&story)
	if session.StorylineIdx >= len(story.Storylines) {
		return
	}
	sl := story.Storylines[session.StorylineIdx]
	if session.EnactmentIdx >= len(sl.Acts) {
		return
	}
	en := sl.Acts[session.EnactmentIdx]
	if en == nil {
		return
	}

	// Look for an OnClick trigger matching the URL (stored in UserActionValue).
	for _, triggers := range en.OnEvent {
		for _, tr := range triggers {
			if tr == nil || tr.TriggerType != pkgmodels.OnClick {
				continue
			}
			if tr.UserActionValue == "" || !strings.Contains(clickedURL, tr.UserActionValue) {
				continue
			}
			// Matching click trigger found — execute its action.
			if tr.DoAction != nil {
				action := tr.DoAction.ActionName
				log.Printf("click trigger fired: %s for user %s on %s", action, userPublicId, clickedURL)
				switch {
				case strings.HasPrefix(action, "mark_complete"), action == "mark_complete":
					markStorySessionComplete(session.PublicId)
				case strings.HasPrefix(action, "next_scene"), action == "next_scene":
					// advanceSession dispatches on NextAction, which holds the
					// enactment's on-sent default (often mark_complete). The
					// click trigger's own action must win for this advance.
					session.NextAction = "next_scene"
					advanceSession(session)
				}
			}
			return
		}
	}
}

func markStorySessionComplete(sessionPublicId string) {
	db.GetCollection(storySessionCollection).Update(
		bson.M{"public_id": sessionPublicId},
		bson.M{"$set": bson.M{"status": "completed"}},
	)
	log.Printf("click tracking: session %s marked complete", sessionPublicId)
}
