package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	"github.com/josephalai/sentanyl/pkg/emailer"
	"github.com/josephalai/sentanyl/pkg/jobs"
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

// RegisterStoryEngineRoutes wires the internal story-start endpoint behind
// the signed service-token check (API-001): fail-closed in production,
// warn-only in dev so local tooling keeps working.
func RegisterStoryEngineRoutes(r *gin.Engine) {
	r.POST("/internal/story/start", auth.RequireServiceAuth(), handleInternalStartStory)
}

// HandleTestTickStories is the e2e fast-forward hook (SENTANYL_E2E_MODE=1
// only — registered next to /internal/test/hydrate-certs in main). It
// rewinds sent_at on active sessions by `seconds` and synchronously runs one
// scheduler pass so time-based story waits can be exercised without sleeping.
func HandleTestTickStories(c *gin.Context) {
	var req struct {
		Seconds      int    `json:"seconds"`
		UserPublicId string `json:"user_public_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Seconds <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "seconds (>0) is required"})
		return
	}
	sel := bson.M{"status": "active"}
	if req.UserPublicId != "" {
		sel["user_public_id"] = req.UserPublicId
	}
	var sessions []StorySession
	if err := db.GetCollection(storySessionCollection).Find(sel).All(&sessions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	for _, s := range sessions {
		db.GetCollection(storySessionCollection).UpdateId(s.Id, bson.M{
			"$set": bson.M{"sent_at": s.SentAt.Add(-time.Duration(req.Seconds) * time.Second)},
		})
	}
	advanceExpiredSessions()
	c.JSON(http.StatusOK, gin.H{"status": "ok", "rewound_seconds": req.Seconds, "sessions": len(sessions)})
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
	hydrateStoryGraph(&story)

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
	if user.UnsubscribedAt != nil {
		return fmt.Errorf("user %s has unsubscribed — story not started", userPublicId)
	}

	baseURL := os.Getenv("PUBLIC_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost"
	}
	send := recordStoryEmailSend(&story, subscriberId, userPublicId, toEmail, content.Subject, 0, 0)
	body := RewriteLinksForTracking(content.Body, userPublicId, baseURL, send.PublicId)
	body = injectOpenPixel(body, baseURL, send.PublicId)
	unsubURL := emailer.UnsubURL(baseURL, userPublicId)
	body = emailer.AppendUnsubFooter(body, unsubURL, emailer.TenantPostalAddress(subscriberId))

	replyTo, sendHeaders := storyReplyHeaders(send, content.ReplyTo, unsubURL)
	if err := sendStoryEmail(content.FromEmail, toEmail, content.Subject, body, replyTo, sendHeaders); err != nil {
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

// Story sweep job (W3-B / OPS-001): the session-advance pass runs as a durable
// self-rescheduling job instead of an in-process ticker, so it survives
// restarts and — via the job lease — never double-advances (and double-sends
// email) when multiple core-service replicas run (OPS-004).
const (
	storySweepJobType  = "story.sweep"
	storySweepInterval = 5 * time.Second
)

// RegisterStoryJobs binds the story sweep handler and bootstraps the sweep
// chain. Call at startup after Mongo is up; the worker must also be running.
func RegisterStoryJobs() {
	jobs.Register(storySweepJobType, func(ctx context.Context, job *jobs.Job) error {
		// Re-arm the chain FIRST so a crash mid-sweep never stalls scheduling.
		if err := jobs.EnqueueSweep(storySweepJobType, time.Now().Add(storySweepInterval), storySweepInterval); err != nil {
			return err
		}
		advanceExpiredSessions()
		// Sweep rows accrue one per interval; prune the succeeded ones hourly.
		if job.RunAt.Unix()%3600 < int64(storySweepInterval/time.Second) {
			jobs.PruneSucceeded(storySweepJobType, 24*time.Hour)
		}
		return nil
	})
	if err := jobs.EnqueueSweep(storySweepJobType, time.Now(), storySweepInterval); err != nil {
		log.Printf("story engine: bootstrap sweep enqueue failed: %v", err)
	}
	log.Printf("story engine: durable sweep registered (%s interval)", storySweepInterval)
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
		hydrateStoryGraph(&story)
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
		if advUser.UnsubscribedAt != nil {
			log.Printf("story engine: user %s unsubscribed — completing session %s without sending", s.UserPublicId, s.PublicId)
			markSessionComplete(s)
			return
		}

		// Claim the advancement with a compare-and-set BEFORE sending: the
		// durable sweep and the e2e tick hook (or two sweeps around a lease
		// expiry) can race on the same expired session, and only the actor
		// that wins this update may send the step email. The predicate pins
		// the exact state this actor read (idx + sent_at), so the loser
		// matches zero documents and returns silently.
		waitSeconds, nextAction := getOnSentWait(en)
		if err := db.GetCollection(storySessionCollection).Update(
			bson.M{
				"public_id":     s.PublicId,
				"status":        "active",
				"enactment_idx": s.EnactmentIdx,
				"sent_at":       s.SentAt,
			},
			bson.M{"$set": bson.M{
				"enactment_idx": nextIdx,
				"sent_at":       time.Now(),
				"wait_seconds":  waitSeconds,
				"next_action":   nextAction,
			}},
		); err != nil {
			log.Printf("story engine: session %s advancement already claimed elsewhere", s.PublicId)
			return
		}

		baseURL := os.Getenv("PUBLIC_BASE_URL")
		if baseURL == "" {
			baseURL = "http://localhost"
		}
		send := recordStoryEmailSend(&story, s.SubscriberId, s.UserPublicId, toEmail, content.Subject, s.StorylineIdx, nextIdx)
		body := RewriteLinksForTracking(content.Body, s.UserPublicId, baseURL, send.PublicId)
		body = injectOpenPixel(body, baseURL, send.PublicId)
		unsubURL := emailer.UnsubURL(baseURL, s.UserPublicId)
		body = emailer.AppendUnsubFooter(body, unsubURL, emailer.TenantPostalAddress(s.SubscriberId))

		replyTo, sendHeaders := storyReplyHeaders(send, content.ReplyTo, unsubURL)
		if err := sendStoryEmail(content.FromEmail, toEmail, content.Subject, body, replyTo, sendHeaders); err != nil {
			log.Printf("story engine: advance email failed for session %s: %v", s.PublicId, err)
		} else {
			log.Printf("story engine: advanced session %s to enactment %d, sent %q", s.PublicId, nextIdx, content.Subject)
		}

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

// hydrateStoryGraph reconstructs the embedded Storylines→Acts→Scenes/Triggers
// tree from the id references the script deployer persists (the inverse of
// Story.ReadyMongoStore, which strips embedded docs and stores each entity in
// its own collection). Stories inserted with embedded storylines are left
// untouched. Without this, form-triggered stories deployed via /api/script
// fail with "has no storylines".
func hydrateStoryGraph(story *pkgmodels.Story) {
	if story == nil || story.StorylineIds == nil || len(story.StorylineIds.Ids) == 0 {
		return
	}
	// ID-004: scope every reference resolution to the story's own tenant so a
	// corrupted or injected id reference cannot pull another tenant's graph.
	tid := story.SubscriberId
	// Merge (not just fallback): a story can hold both embedded storylines
	// (GUI-added) and id references (script-deployed) — the engine must see
	// the union, same as hydrateStory on the tenant routes.
	present := make(map[string]bool, len(story.Storylines))
	for _, sl := range story.Storylines {
		if sl != nil {
			present[sl.PublicId] = true
		}
	}
	for _, slID := range story.StorylineIds.Ids {
		var sl pkgmodels.Storyline
		if err := db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"_id": slID, "subscriber_id": tid}).One(&sl); err != nil {
			continue
		}
		if present[sl.PublicId] {
			continue
		}
		present[sl.PublicId] = true
		if len(sl.Acts) == 0 && sl.ActIds != nil {
			for _, actID := range sl.ActIds.Ids {
				var en pkgmodels.Enactment
				if err := db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"_id": actID, "subscriber_id": tid}).One(&en); err != nil {
					continue
				}
				hydrateEnactment(&en, tid)
				sl.Acts = append(sl.Acts, &en)
			}
			sort.Slice(sl.Acts, func(i, j int) bool { return sl.Acts[i].NaturalOrder < sl.Acts[j].NaturalOrder })
		}
		story.Storylines = append(story.Storylines, &sl)
	}
	sort.Slice(story.Storylines, func(i, j int) bool {
		return story.Storylines[i].NaturalOrder < story.Storylines[j].NaturalOrder
	})
}

// hydrateEnactment resolves an enactment's scene/trigger references. tid is the
// owning tenant (subscriber_id); every lookup is scoped to it (ID-004).
func hydrateEnactment(en *pkgmodels.Enactment, tid string) {
	if en.SendScene == nil && en.SendSceneId != nil && en.SendSceneId.Id.Valid() {
		en.SendScene = loadScene(en.SendSceneId.Id, tid)
	}
	if len(en.SendScenes) == 0 && en.SendScenesIds != nil {
		for _, scID := range en.SendScenesIds.Ids {
			if sc := loadScene(scID, tid); sc != nil {
				en.SendScenes = append(en.SendScenes, sc)
			}
		}
	}
	if len(en.OnEvent) == 0 && en.OnEventIds != nil {
		en.OnEvent = map[string][]*pkgmodels.Trigger{}
		for _, trID := range en.OnEventIds.Ids {
			var tr pkgmodels.Trigger
			if err := db.GetCollection(pkgmodels.TriggerCollection).Find(bson.M{"_id": trID, "subscriber_id": tid}).One(&tr); err != nil {
				continue
			}
			if tr.DoAction == nil && tr.DoActionId != nil && tr.DoActionId.Id.Valid() {
				var act pkgmodels.Action
				if err := db.GetCollection(pkgmodels.ActionCollection).Find(bson.M{"_id": tr.DoActionId.Id, "subscriber_id": tid}).One(&act); err == nil {
					tr.DoAction = &act
				}
			}
			en.OnEvent[tr.TriggerType] = append(en.OnEvent[tr.TriggerType], &tr)
		}
	}
}

func loadScene(id bson.ObjectId, tid string) *pkgmodels.Scene {
	var sc pkgmodels.Scene
	if err := db.GetCollection(pkgmodels.SceneCollection).Find(bson.M{"_id": id, "subscriber_id": tid}).One(&sc); err != nil {
		return nil
	}
	if sc.Message == nil && sc.MessageId != nil && sc.MessageId.Id.Valid() {
		var msg pkgmodels.Message
		if err := db.GetCollection(pkgmodels.MessageCollection).Find(bson.M{"_id": sc.MessageId.Id, "subscriber_id": tid}).One(&msg); err == nil {
			sc.Message = &msg
		}
	}
	return &sc
}

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

// recordStoryEmailSend inserts the unified per-email tracking row for a story
// engine send. Failures are logged, never fatal — tracking must not block a
// send.
func recordStoryEmailSend(story *pkgmodels.Story, subscriberId, userPublicId, toEmail, subject string, storylineIdx, enactmentIdx int) *pkgmodels.EmailSend {
	tenantID := story.TenantID
	if tenantID == "" && bson.IsObjectIdHex(subscriberId) {
		tenantID = bson.ObjectIdHex(subscriberId)
	}
	send := pkgmodels.NewEmailSend(tenantID, pkgmodels.EmailSendSourceStory, toEmail, subject)
	send.ContactPublicID = userPublicId
	send.StoryPublicID = story.PublicId
	send.StoryName = story.Name
	send.StorylineIdx = storylineIdx
	send.EnactmentIdx = enactmentIdx
	send.MessageID, _ = emailer.ReplyCorrelation(send.PublicId)
	if err := db.GetCollection(pkgmodels.EmailSendCollection).Insert(send); err != nil {
		log.Printf("story engine: email send row insert failed: %v", err)
	}
	return send
}

// storyReplyHeaders resolves the Reply-To and extra headers for a story send:
// unsubscribe headers always; Message-ID + VERP Reply-To when platform reply
// ingestion is configured. A scene-configured reply_to always wins over VERP.
func storyReplyHeaders(send *pkgmodels.EmailSend, sceneReplyTo, unsubURL string) (string, map[string]string) {
	headers := emailer.UnsubHeaders(unsubURL)
	msgID, verpReplyTo := emailer.ReplyCorrelation(send.PublicId)
	if msgID != "" {
		headers["Message-ID"] = msgID
	}
	if sceneReplyTo == "" {
		return verpReplyTo, headers
	}
	return sceneReplyTo, headers
}

// injectOpenPixel appends the unified 1x1 open-tracking pixel, inside </body>
// when present.
func injectOpenPixel(html, baseURL, sendPublicId string) string {
	if sendPublicId == "" {
		return html
	}
	pixel := `<img src="` + strings.TrimRight(baseURL, "/") + `/api/marketing/track/open?e=` + sendPublicId + `" width="1" height="1" style="display:none" alt=""/>`
	if i := strings.LastIndex(html, "</body>"); i >= 0 {
		return html[:i] + pixel + html[i:]
	}
	return html + pixel
}

// sendStoryEmail sends an email via the EMAIL_PROVIDER-selected provider
// (warmup router / PowerMTA / Brevo / plain SMTP-MailHog).
func sendStoryEmail(from, to, subject, htmlBody, replyTo string, extraHeaders map[string]string) error {
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
	for k, v := range extraHeaders {
		buf.WriteString(sanitize(k) + ": " + sanitize(v) + "\r\n")
	}
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(htmlBody)

	return emailer.SendRawFromEnv(from, to, buf.Bytes())
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
	req, err := http.NewRequest(http.MethodPost, coreURL+"/internal/story/start", bytes.NewReader(payload))
	if err != nil {
		log.Printf("callStartStory: request build error: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	auth.AttachServiceAuth(req, "core") // API-001 signed service identity
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("callStartStory: HTTP error: %v", err)
		return
	}
	defer resp.Body.Close()
	log.Printf("callStartStory: story=%q user=%s → HTTP %d", storyName, userPublicId, resp.StatusCode)
}
