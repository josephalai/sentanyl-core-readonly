package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/utils"
)

const storySessionCollection = "story_sessions"

// StorySession tracks a user's position in an executing story.
type StorySession struct {
	Id             bson.ObjectId `bson:"_id" json:"id"`
	PublicId       string        `bson:"public_id" json:"public_id"`
	UserPublicId   string        `bson:"user_public_id" json:"user_public_id"`
	SubscriberId   string        `bson:"subscriber_id" json:"subscriber_id"`
	StoryId        bson.ObjectId `bson:"story_id" json:"story_id"`
	StoryName      string        `bson:"story_name" json:"story_name"`
	StorylineIdx   int           `bson:"storyline_idx" json:"storyline_idx"`
	EnactmentIdx   int           `bson:"enactment_idx" json:"enactment_idx"`
	SentAt         time.Time     `bson:"sent_at" json:"sent_at"`
	WaitSeconds    int           `bson:"wait_seconds" json:"wait_seconds"`
	NextAction     string        `bson:"next_action" json:"next_action"` // "next_scene" | "mark_complete"
	Status         string        `bson:"status" json:"status"`           // "active" | "completed"
	CreatedAt      time.Time     `bson:"created_at" json:"created_at"`
}

// RegisterStoryEngineRoutes wires internal story-start endpoint.
func RegisterStoryEngineRoutes(r *gin.Engine) {
	r.POST("/internal/story/start", handleInternalStartStory)
}

func handleInternalStartStory(c *gin.Context) {
	var req struct {
		StoryName    string `json:"story_name"    binding:"required"`
		SubscriberId string `json:"subscriber_id" binding:"required"`
		UserPublicId string `json:"user_public_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := StartStoryForUser(req.StoryName, req.SubscriberId, req.UserPublicId); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// StartStoryForUser finds the named story, sends the first email, and creates
// a StorySession so the scheduler can advance through the sequence.
func StartStoryForUser(storyName, subscriberId, userPublicId string) error {
	log.Printf("story engine: StartStoryForUser story=%q user=%s", storyName, userPublicId)

	// Cancel any existing active session for this user+story so we don't double-send.
	db.GetCollection(storySessionCollection).UpdateAll(
		bson.M{"user_public_id": userPublicId, "story_name": storyName, "status": "active"},
		bson.M{"$set": bson.M{"status": "superseded"}},
	)

	// Find the story. Try the most-recently deployed one first.
	var stories []pkgmodels.Story
	if err := db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{
		"name":         storyName,
		"subscriber_id": subscriberId,
	}).All(&stories); err != nil || len(stories) == 0 {
		return fmt.Errorf("story %q not found for subscriber %s", storyName, subscriberId)
	}
	// Use the last (most recently deployed) story.
	story := stories[len(stories)-1]

	if len(story.Storylines) == 0 {
		return fmt.Errorf("story %q has no storylines", storyName)
	}
	sl := story.Storylines[0]
	if len(sl.Acts) == 0 {
		return fmt.Errorf("story %q storyline has no enactments", storyName)
	}
	en := sl.Acts[0]

	scene := getSceneFromEnactment(en, 0)
	if scene == nil {
		return fmt.Errorf("story %q enactment 0 has no scene", storyName)
	}

	// Build and send the email.
	content := getSceneContent(scene)
	if content == nil {
		return fmt.Errorf("story %q scene has no message content", storyName)
	}

	// Look up the user's real email address.
	toEmail := userPublicId + "@story.internal"
	var user pkgmodels.User
	if err := db.GetCollection(pkgmodels.UserCollection).Find(bson.M{"public_id": userPublicId}).One(&user); err == nil {
		if string(user.Email) != "" {
			toEmail = string(user.Email)
		}
	}

	baseURL := os.Getenv("PUBLIC_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost"
	}
	body := RewriteLinksForTracking(content.Body, userPublicId, baseURL)

	if err := sendStoryEmail(content.FromEmail, toEmail, content.Subject, body, content.ReplyTo); err != nil {
		log.Printf("story engine: email send failed: %v", err)
		// Don't abort — create session anyway so scheduler can retry.
	} else {
		log.Printf("story engine: sent email %q to user %s", content.Subject, userPublicId)
	}

	// Determine wait duration from OnSent trigger.
	waitSeconds, nextAction := getOnSentWait(en)

	session := &StorySession{
		Id:           bson.NewObjectId(),
		PublicId:     utils.GeneratePublicId(),
		UserPublicId: userPublicId,
		SubscriberId: subscriberId,
		StoryId:      story.Id,
		StoryName:    storyName,
		StorylineIdx: 0,
		EnactmentIdx: 0,
		SentAt:       time.Now(),
		WaitSeconds:  waitSeconds,
		NextAction:   nextAction,
		Status:       "active",
		CreatedAt:    time.Now(),
	}
	if err := db.GetCollection(storySessionCollection).Insert(session); err != nil {
		return fmt.Errorf("failed to create story session: %v", err)
	}
	log.Printf("story engine: session %s created (wait %ds → %s)", session.PublicId, waitSeconds, nextAction)
	return nil
}

// StartStoryScheduler launches the background goroutine that advances sessions.
func StartStoryScheduler() {
	log.Println("story engine: scheduler starting (2s interval)")
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			advanceExpiredSessions()
		}
	}()
}

func advanceExpiredSessions() {
	var sessions []StorySession
	if err := db.GetCollection(storySessionCollection).Find(bson.M{
		"status": "active",
	}).All(&sessions); err != nil {
		return
	}

	now := time.Now()
	for i := range sessions {
		s := sessions[i]
		deadline := s.SentAt.Add(time.Duration(s.WaitSeconds) * time.Second)
		if now.Before(deadline) {
			continue
		}
		log.Printf("story engine: session %s expired (waited %ds), action=%s", s.PublicId, s.WaitSeconds, s.NextAction)
		advanceSession(s)
	}
}

// advanceSession moves a session to the next enactment (or marks it complete).
func advanceSession(s StorySession) {
	switch s.NextAction {
	case "mark_complete", "advance_to_next_storyline":
		db.GetCollection(storySessionCollection).Update(
			bson.M{"public_id": s.PublicId},
			bson.M{"$set": bson.M{"status": "completed"}},
		)
		log.Printf("story engine: session %s completed", s.PublicId)
		return

	case "next_scene":
		// Advance to the next enactment in the storyline.
		var story pkgmodels.Story
		if err := db.GetCollection(pkgmodels.StoryCollection).FindId(s.StoryId).One(&story); err != nil {
			log.Printf("story engine: story not found for session %s: %v", s.PublicId, err)
			return
		}
		if s.StorylineIdx >= len(story.Storylines) {
			markSessionComplete(s)
			return
		}
		sl := story.Storylines[s.StorylineIdx]
		nextIdx := s.EnactmentIdx + 1
		if nextIdx >= len(sl.Acts) {
			markSessionComplete(s)
			return
		}
		en := sl.Acts[nextIdx]
		scene := getSceneFromEnactment(en, 0)
		if scene == nil {
			markSessionComplete(s)
			return
		}
		content := getSceneContent(scene)
		if content == nil {
			markSessionComplete(s)
			return
		}

		toEmail := s.UserPublicId + "@story.internal"
		var advUser pkgmodels.User
		if err2 := db.GetCollection(pkgmodels.UserCollection).Find(bson.M{"public_id": s.UserPublicId}).One(&advUser); err2 == nil {
			if string(advUser.Email) != "" {
				toEmail = string(advUser.Email)
			}
		}

		baseURL := os.Getenv("PUBLIC_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost"
		}
		body := RewriteLinksForTracking(content.Body, s.UserPublicId, baseURL)

		if err := sendStoryEmail(content.FromEmail, toEmail, content.Subject, body, content.ReplyTo); err != nil {
			log.Printf("story engine: advance email failed for session %s: %v", s.PublicId, err)
		} else {
			log.Printf("story engine: advanced session %s to enactment %d, sent %q", s.PublicId, nextIdx, content.Subject)
		}

		waitSeconds, nextAction := getOnSentWait(en)
		db.GetCollection(storySessionCollection).Update(
			bson.M{"public_id": s.PublicId},
			bson.M{"$set": bson.M{
				"enactment_idx": nextIdx,
				"sent_at":       time.Now(),
				"wait_seconds":  waitSeconds,
				"next_action":   nextAction,
			}},
		)

	default:
		markSessionComplete(s)
	}
}

func markSessionComplete(s StorySession) {
	db.GetCollection(storySessionCollection).Update(
		bson.M{"public_id": s.PublicId},
		bson.M{"$set": bson.M{"status": "completed"}},
	)
	log.Printf("story engine: session %s completed", s.PublicId)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func getSceneFromEnactment(en *pkgmodels.Enactment, idx int) *pkgmodels.Scene {
	if en == nil {
		return nil
	}
	if len(en.SendScenes) > idx {
		return en.SendScenes[idx]
	}
	if idx == 0 && en.SendScene != nil {
		return en.SendScene
	}
	return nil
}

func getSceneContent(scene *pkgmodels.Scene) *pkgmodels.MessageContent {
	if scene == nil {
		return nil
	}
	if scene.Message != nil && scene.Message.Content != nil {
		return scene.Message.Content
	}
	// Fallback: scene has direct subject/body fields (older format).
	if scene.Subject != "" || scene.Body != "" {
		return &pkgmodels.MessageContent{
			Subject:   scene.Subject,
			Body:      scene.Body,
			FromEmail: scene.FromEmail,
			FromName:  scene.FromName,
			ReplyTo:   scene.ReplyTo,
		}
	}
	return nil
}

func getOnSentWait(en *pkgmodels.Enactment) (waitSeconds int, nextAction string) {
	waitSeconds = 30 // default
	nextAction = "mark_complete"
	if en == nil || en.OnEvent == nil {
		return
	}
	triggers, ok := en.OnEvent[pkgmodels.OnSent]
	if !ok || len(triggers) == 0 {
		return
	}
	tr := triggers[0]
	if tr == nil || tr.DoAction == nil || tr.DoAction.When == nil {
		return
	}
	if tr.DoAction.When.WaitUntil != nil {
		wu := tr.DoAction.When.WaitUntil
		amount := wu.Amount
		switch wu.TimeUnit {
		case "seconds", "second":
			waitSeconds = amount
		case "minutes", "minute":
			waitSeconds = amount * 60
		case "hours", "hour":
			waitSeconds = amount * 3600
		default:
			waitSeconds = amount * 60
		}
	}
	nextAction = tr.DoAction.ActionName
	return
}

// sendStoryEmail sends an email via MailHog SMTP directly.
func sendStoryEmail(from, to, subject, htmlBody, replyTo string) error {
	smtpHost := os.Getenv("SMTP_HOST")
	if smtpHost == "" {
		smtpHost = "mailhog"
	}
	smtpPort := os.Getenv("SMTP_PORT")
	if smtpPort == "" {
		smtpPort = "1025"
	}
	addr := smtpHost + ":" + smtpPort

	if from == "" {
		from = "noreply@sentanyl.local"
	}

	var buf bytes.Buffer
	buf.WriteString("From: " + sanitize(from) + "\r\n")
	buf.WriteString("To: " + sanitize(to) + "\r\n")
	if replyTo != "" {
		buf.WriteString("Reply-To: " + sanitize(replyTo) + "\r\n")
	}
	buf.WriteString("Subject: " + sanitize(subject) + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(htmlBody)

	return smtp.SendMail(addr, nil, from, []string{to}, buf.Bytes())
}

func sanitize(s string) string {
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	return s
}

// callStartStory is called by marketing-service (via HTTP) when a start_story
// funnel action fires. This version is used internally within core-service itself.
func callStartStory(storyName, subscriberId, userPublicId string) {
	coreURL := os.Getenv("CORE_SERVICE_URL")
	if coreURL == "" {
		coreURL = "http://core-service:8081"
	}
	payload, _ := json.Marshal(map[string]string{
		"story_name":     storyName,
		"subscriber_id":  subscriberId,
		"user_public_id": userPublicId,
	})
	resp, err := http.Post(coreURL+"/internal/story/start", "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("callStartStory: HTTP error: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("callStartStory: story=%q user=%s → HTTP %d", storyName, userPublicId, resp.StatusCode)
}
