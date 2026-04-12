package routes

import (
	"encoding/base64"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
)

const trackingSeparator = "|"

// encodeTrackingToken encodes a URL + user public-ID for use in a tracking link.
func encodeTrackingToken(originalURL, userPublicId string) string {
	raw := originalURL + trackingSeparator + userPublicId
	return base64.URLEncoding.EncodeToString([]byte(raw))
}

// decodeTrackingToken reverses encodeTrackingToken.
func decodeTrackingToken(token string) (originalURL, userPublicId string, ok bool) {
	b, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(string(b), trackingSeparator, 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

var hrefRegex = regexp.MustCompile(`(?i)(href=["'])([^"']+)(["'])`)

// RewriteLinksForTracking replaces every href in the HTML body with a tracking
// redirect so clicks are recorded before the user reaches the destination.
func RewriteLinksForTracking(html, userPublicId, baseURL string) string {
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
		token := encodeTrackingToken(originalURL, userPublicId)
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

	originalURL, userPublicId, ok := decodeTrackingToken(token)
	if !ok {
		log.Printf("click tracking: invalid token: %s", token)
		c.Redirect(http.StatusFound, "/")
		return
	}

	log.Printf("click tracking: url=%s user=%s", originalURL, userPublicId)

	// Fire story click triggers asynchronously — don't block the redirect.
	go fireClickTrigger(userPublicId, originalURL)

	c.Redirect(http.StatusFound, originalURL)
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

	// Load the story.
	var story pkgmodels.Story
	if err := db.GetCollection(pkgmodels.StoryCollection).FindId(session.StoryId).One(&story); err != nil {
		return
	}
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
