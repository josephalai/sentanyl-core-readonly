package routes

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/badges"
	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/plans"
	"github.com/josephalai/sentanyl/pkg/publicchannel"
	"github.com/josephalai/sentanyl/pkg/utils"
)

// RegisterStoryRoutes wires all story-builder CRUD endpoints onto a router group (expects /api/tenant prefix with auth middleware).
func RegisterStoryRoutes(r gin.IRouter) {
	// Stories
	r.GET("/stories", handleGetStories)
	r.GET("/story/:id", handleGetStory)
	r.POST("/story/", handleCreateStory)
	r.PUT("/story/:id", handleUpdateStory)
	r.DELETE("/story/:id", handleDeleteStory)
	r.DELETE("/stories", handleDeleteAllStories)
	r.DELETE("/story/:id/purge", handlePurgeStory)
	r.PUT("/story/:id/start", handleStartStory)
	r.PUT("/story/:id/stop", handleStopStory)
	r.POST("/story/:id/storylines/:storylineId", handleAddStorylineToStory)
	r.DELETE("/story/:id/storylines/:storylineId", handleRemoveStorylineFromStory)

	// Storylines
	r.GET("/storylines", handleGetStorylines)
	r.GET("/storyline/:id", handleGetStoryline)
	r.POST("/storyline/", handleCreateStoryline)
	r.PUT("/storyline/:id", handleUpdateStoryline)
	r.DELETE("/storyline/:id", handleDeleteStoryline)
	r.PUT("/storyline/:id/start", handleStartStoryline)
	r.PUT("/storyline/:id/stop", handleStopStoryline)
	r.POST("/storyline/:id/enactments", handleAddEnactmentToStoryline)
	r.DELETE("/storyline/:id/enactments/:enactmentId", handleRemoveEnactmentFromStoryline)

	// Enactments
	r.GET("/enactments", handleGetEnactments)
	r.GET("/enactment/:id", handleGetEnactment)
	r.POST("/enactment/", handleCreateEnactment)
	r.PUT("/enactment/:id", handleUpdateEnactment)
	r.DELETE("/enactment/:id", handleDeleteEnactment)
	r.PUT("/enactment/:id/start", handleStartEnactment)
	r.PUT("/enactment/:id/stop", handleStopEnactment)
	r.POST("/enactment/:id/trigger", handleAddTriggerToEnactment)
	r.DELETE("/enactment/:id/trigger/:triggerId", handleRemoveTriggerFromEnactment)
	r.POST("/enactment/:id/scene/:sceneId", handleSetEnactmentScene)
	r.DELETE("/enactment/:id/scene", handleRemoveEnactmentScene)

	// Scenes
	r.GET("/scenes", handleGetScenes)
	r.GET("/scene/:id", handleGetScene)
	r.POST("/scene/", handleCreateScene)
	r.PUT("/scene/:id", handleUpdateScene)
	r.DELETE("/scene/:id", handleDeleteScene)
	r.POST("/scene/:id/tag", handleAddTagToScene)
	r.DELETE("/scene/:id/tag/:tagId", handleRemoveTagFromScene)
	r.POST("/scene/:id/message/:messageId", handleSetSceneMessage)
	r.DELETE("/scene/:id/message", handleRemoveSceneMessage)

	// Messages
	r.GET("/messages", handleGetMessages)
	r.GET("/message/:id", handleGetMessage)
	r.POST("/message/", handleCreateMessage)
	r.PUT("/message/:id", handleUpdateMessage)
	r.DELETE("/message/:id", handleDeleteMessage)
	r.POST("/message/:id/tag", handleAddTagToMessage)
	r.DELETE("/message/:id/tag/:tagId", handleRemoveTagFromMessage)

	// Message Content
	r.GET("/message_contents", handleGetMessageContents)
	r.GET("/message_content/:id", handleGetMessageContent)
	r.POST("/message_content/", handleCreateMessageContent)
	r.PUT("/message_content/:id", handleUpdateMessageContent)
	r.DELETE("/message_content/:id", handleDeleteMessageContent)

	// Triggers
	r.GET("/triggers", handleGetTriggers)
	r.GET("/trigger/:id", handleGetTrigger)
	r.POST("/trigger/", handleCreateTrigger)
	r.PUT("/trigger/:id", handleUpdateTrigger)
	r.DELETE("/trigger/:id", handleDeleteTrigger)

	// Actions
	r.GET("/actions", handleGetActions)
	r.GET("/action/:id", handleGetAction)
	r.POST("/action/", handleCreateAction)
	r.PUT("/action/:id", handleUpdateAction)
	r.DELETE("/action/:id", handleDeleteAction)

	// Badges
	r.GET("/badges", handleGetBadges)
	r.GET("/badge/:id", handleGetBadge)
	r.POST("/badge/", handleCreateBadge)
	r.PUT("/badge/:id", handleUpdateBadge)
	r.DELETE("/badge/:id", handleDeleteBadge)
	r.PUT("/user_badge/user/:userId/badge/:badgeId", handleAssignBadgeToUser)
	r.DELETE("/user_badge/user/:userId/badge/:badgeId", handleRemoveBadgeFromUser)

	// Tags
	r.GET("/tags", handleGetTags)
	r.GET("/tag/:id", handleGetTag)
	r.POST("/tag/", handleCreateTag)
	r.PUT("/tag/:id", handleUpdateTag)
	r.DELETE("/tag/:id", handleDeleteTag)

	// Template Variables
	r.GET("/template_variables", handleGetTemplateVariables)
	r.GET("/template_variable/:id", handleGetTemplateVariable)
	r.POST("/template_variable/", handleCreateTemplateVariable)
	r.PUT("/template_variable/:id", handleUpdateTemplateVariable)
	r.DELETE("/template_variable/:id", handleDeleteTemplateVariable)

	// Users / CRM (tenant view of their subscribers)
	r.GET("/users", handleGetUsers)
	r.GET("/user/:id", handleGetUser)
	r.POST("/user/", handleCreateUser)
	r.PUT("/user/:id", handleUpdateUser)
	r.DELETE("/user/:id", handleDeleteUser)
	r.DELETE("/users", handleDeleteAllUsers)
	r.GET("/user/:id/detail", handleGetUserDetail)
	r.POST("/user/:id/story/:storyId", handleAddUserToStory)

	// Email lists
	r.GET("/creator/lists", handleGetEmailLists)
	r.GET("/creator/list/:id", handleGetEmailList)
	r.POST("/creator/list", handleCreateEmailList)
	r.DELETE("/creator/list/:id", handleDeleteEmailList)

	// Email queue / hot triggers
	r.GET("/emails/pending", handleGetPendingEmails)
	r.GET("/hot-triggers", handleGetHotTriggers)

	// Stats
	r.GET("/stats/", handleStatsOverview)
	r.GET("/stats/story", handleStatsStub)
	r.GET("/stats/story/:id", handleStoryStats)
	r.GET("/stats/storyline", handleStatsStub)
	r.GET("/stats/storyline/:id", handleStatsStub)
	r.GET("/stats/enactment", handleStatsStub)
	r.GET("/stats/enactment/:id", handleStatsStub)
	r.GET("/stats/message", handleStatsStub)
	r.GET("/stats/message/:id", handleStatsStub)
	r.GET("/stats/badge", handleStatsStub)
	r.GET("/stats/badge/:id", handleStatsStub)
	r.GET("/stats/trigger", handleStatsStub)
	r.GET("/stats/trigger/:id", handleStatsStub)
	r.GET("/stats/email", handleStatsStub)
	r.GET("/stats/email/:id", handleStatsStub)
	r.GET("/stats/user", handleStatsStub)
	r.GET("/stats/user/:id", handleStatsStub)
	r.GET("/stats/link", handleStatsStub)
	r.GET("/stats/link/:id", handleStatsStub)
	r.GET("/stats/spam", handleStatsStub)
	r.GET("/stats/fail", handleStatsStub)
	r.GET("/stats/validate", handleStatsStub)
	r.GET("/stats/ab", handleStatsStub)
	r.GET("/stats/ab/:id", handleStatsStub)

	// Admin — destructive reset is owner-only (ID-001).
	r.POST("/admin/reset", auth.RequirePermission(auth.PermDataDestroy), handleAdminReset)
}

// ─── Helpers ──────────────────────────────────────────────

func now() *time.Time { t := time.Now(); return &t }

// ─── Stories ──────────────────────────────────────────────

// hydrateStory re-fetches the full tree: storylines → enactments → scenes
// so anything added/updated after initial embedding is always current.
func hydrateStory(story *pkgmodels.Story) {
	// ID-004: every nested lookup is scoped to the story's own tenant. Public
	// IDs and ObjectIds are not an authorization boundary — a corrupted,
	// guessed, or injected reference must never hydrate another tenant's graph.
	tid := story.SubscriberId
	// Script-deployed stories persist storyline references in storyline_ids
	// with no embedded array (see hydrateStoryGraph in story_engine.go).
	// Merge those references in so the GUI sees the full graph regardless of
	// which representation the story was written with.
	if story.StorylineIds != nil {
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
			story.Storylines = append(story.Storylines, &sl)
			present[sl.PublicId] = true
		}
	}
	for i, sl := range story.Storylines {
		var freshSL pkgmodels.Storyline
		if err := db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": sl.PublicId, "subscriber_id": tid}).One(&freshSL); err != nil {
			continue
		}
		// Script-deployed storylines reference enactments by act_ids with no
		// embedded acts — resolve those first so the loop below sees them.
		if len(freshSL.Acts) == 0 && freshSL.ActIds != nil {
			for _, actID := range freshSL.ActIds.Ids {
				var en pkgmodels.Enactment
				if err := db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"_id": actID, "subscriber_id": tid}).One(&en); err != nil {
					continue
				}
				freshSL.Acts = append(freshSL.Acts, &en)
			}
			sort.Slice(freshSL.Acts, func(a, b int) bool { return freshSL.Acts[a].NaturalOrder < freshSL.Acts[b].NaturalOrder })
		}
		for j, en := range freshSL.Acts {
			var freshEn pkgmodels.Enactment
			if err := db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": en.PublicId, "subscriber_id": tid}).One(&freshEn); err != nil {
				continue
			}
			// Script-deployed enactments reference scenes by id — resolve them.
			hydrateEnactment(&freshEn, tid)
			// Re-fetch send_scene so messages set after scene was linked are visible
			if freshEn.SendScene != nil && freshEn.SendScene.PublicId != "" {
				var freshScene pkgmodels.Scene
				if err := db.GetCollection(pkgmodels.SceneCollection).Find(bson.M{"public_id": freshEn.SendScene.PublicId, "subscriber_id": tid}).One(&freshScene); err == nil {
					freshEn.SendScene = &freshScene
				}
			}
			// Re-fetch multi-scene send_scenes as well
			for k, sc := range freshEn.SendScenes {
				var freshScene pkgmodels.Scene
				if err := db.GetCollection(pkgmodels.SceneCollection).Find(bson.M{"public_id": sc.PublicId, "subscriber_id": tid}).One(&freshScene); err == nil {
					freshEn.SendScenes[k] = &freshScene
				}
			}
			freshSL.Acts[j] = &freshEn
		}
		story.Storylines[i] = &freshSL
	}
	sort.Slice(story.Storylines, func(i, j int) bool {
		return story.Storylines[i].NaturalOrder < story.Storylines[j].NaturalOrder
	})
}

func handleGetStories(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.Story
	db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}).All(&items)
	if items == nil {
		items = []pkgmodels.Story{}
	}
	for i := range items {
		hydrateStory(&items[i])
	}
	c.JSON(http.StatusOK, gin.H{"stories": items})
}

func handleGetStory(c *gin.Context) {
	var item pkgmodels.Story
	if err := db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "story not found"})
		return
	}
	hydrateStory(&item)
	c.JSON(http.StatusOK, gin.H{"story": item})
}

func handleCreateStory(c *gin.Context) {
	var item pkgmodels.Story
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
		return
	}
	item.SubscriberId = auth.GetTenantID(c)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.StoryCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"story": item})
}

func handleUpdateStory(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.StoryCollection, storyUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteStory(c *gin.Context) {
	db.GetCollection(pkgmodels.StoryCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteAllStories(c *gin.Context) {
	sid := auth.GetTenantID(c)
	db.GetCollection(pkgmodels.StoryCollection).UpdateAll(bson.M{"subscriber_id": sid}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handlePurgeStory resets a campaign: all story sessions (contact enrollment
// state) are removed so contacts can re-enter, but the story itself is kept —
// matching the GUI copy "Purge campaign and reset all enrolled contacts".
func handlePurgeStory(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var story pkgmodels.Story
	if err := db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": sid}).One(&story); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "story not found"})
		return
	}
	info, _ := db.GetCollection(storySessionCollection).RemoveAll(bson.M{"story_id": story.Id, "subscriber_id": sid})
	removed := 0
	if info != nil {
		removed = info.Removed
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "sessions_removed": removed})
}

func handleStartStory(c *gin.Context) {
	db.GetCollection(pkgmodels.StoryCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"status": "active"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStopStory(c *gin.Context) {
	db.GetCollection(pkgmodels.StoryCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"status": "stopped"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddStorylineToStory(c *gin.Context) {
	storyId := c.Param("id")
	storylineId := c.Param("storylineId")
	var sl pkgmodels.Storyline
	if err := db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": storylineId, "subscriber_id": auth.GetTenantID(c)}).One(&sl); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "storyline not found"})
		return
	}
	db.GetCollection(pkgmodels.StoryCollection).Update(
		bson.M{"public_id": storyId, "subscriber_id": auth.GetTenantID(c)},
		bson.M{"$push": bson.M{"storylines": sl}},
	)
	var updated pkgmodels.Story
	db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{"public_id": storyId, "subscriber_id": auth.GetTenantID(c)}).One(&updated)
	c.JSON(http.StatusOK, gin.H{"story": updated})
}

func handleRemoveStorylineFromStory(c *gin.Context) {
	storyId := c.Param("id")
	storylineId := c.Param("storylineId")
	// A storyline can be attached embedded (GUI) or by id reference (script
	// deploy) — pull from both representations.
	pull := bson.M{"storylines": bson.M{"public_id": storylineId, "subscriber_id": auth.GetTenantID(c)}}
	var sl pkgmodels.Storyline
	if err := db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": storylineId, "subscriber_id": auth.GetTenantID(c)}).One(&sl); err == nil {
		pull["storyline_ids.bson_ids"] = sl.Id
	}
	db.GetCollection(pkgmodels.StoryCollection).Update(
		bson.M{"public_id": storyId, "subscriber_id": auth.GetTenantID(c)},
		bson.M{"$pull": pull},
	)
	var updated pkgmodels.Story
	db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{"public_id": storyId, "subscriber_id": auth.GetTenantID(c)}).One(&updated)
	c.JSON(http.StatusOK, gin.H{"story": updated})
}

// ─── Storylines ───────────────────────────────────────────

func handleGetStorylines(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.Storyline
	db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}).All(&items)
	if items == nil {
		items = []pkgmodels.Storyline{}
	}
	c.JSON(http.StatusOK, gin.H{"storylines": items})
}

func handleGetStoryline(c *gin.Context) {
	var item pkgmodels.Storyline
	if err := db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "storyline not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"storyline": item})
}

func handleCreateStoryline(c *gin.Context) {
	var item pkgmodels.Storyline
	c.ShouldBindJSON(&item)
	item.SubscriberId = auth.GetTenantID(c)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.StorylineCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"storyline": item})
}

func handleUpdateStoryline(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.StorylineCollection, storylineUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteStoryline(c *gin.Context) {
	db.GetCollection(pkgmodels.StorylineCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStartStoryline(c *gin.Context) {
	db.GetCollection(pkgmodels.StorylineCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"status": "active"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStopStoryline(c *gin.Context) {
	db.GetCollection(pkgmodels.StorylineCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"status": "stopped"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddEnactmentToStoryline(c *gin.Context) {
	storylineId := c.Param("id")
	var body struct {
		EnactmentId string `json:"enactment_id"`
	}
	c.ShouldBindJSON(&body)
	var en pkgmodels.Enactment
	if err := db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": body.EnactmentId, "subscriber_id": auth.GetTenantID(c)}).One(&en); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "enactment not found"})
		return
	}
	db.GetCollection(pkgmodels.StorylineCollection).Update(
		bson.M{"public_id": storylineId, "subscriber_id": auth.GetTenantID(c)},
		bson.M{"$push": bson.M{"acts": en}},
	)
	var updated pkgmodels.Storyline
	db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": storylineId, "subscriber_id": auth.GetTenantID(c)}).One(&updated)
	c.JSON(http.StatusOK, gin.H{"storyline": updated})
}

func handleRemoveEnactmentFromStoryline(c *gin.Context) {
	storylineId := c.Param("id")
	enactmentId := c.Param("enactmentId")
	db.GetCollection(pkgmodels.StorylineCollection).Update(
		bson.M{"public_id": storylineId, "subscriber_id": auth.GetTenantID(c)},
		bson.M{"$pull": bson.M{"acts": bson.M{"public_id": enactmentId, "subscriber_id": auth.GetTenantID(c)}}},
	)
	var updated pkgmodels.Storyline
	db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": storylineId, "subscriber_id": auth.GetTenantID(c)}).One(&updated)
	c.JSON(http.StatusOK, gin.H{"storyline": updated})
}

// ─── Enactments ───────────────────────────────────────────

func handleGetEnactments(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.Enactment
	db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}).All(&items)
	if items == nil {
		items = []pkgmodels.Enactment{}
	}
	c.JSON(http.StatusOK, gin.H{"enactments": items})
}

func handleGetEnactment(c *gin.Context) {
	var item pkgmodels.Enactment
	if err := db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "enactment not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enactment": item})
}

func handleCreateEnactment(c *gin.Context) {
	var item pkgmodels.Enactment
	c.ShouldBindJSON(&item)
	item.SubscriberId = auth.GetTenantID(c)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.EnactmentCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"enactment": item})
}

func handleUpdateEnactment(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.EnactmentCollection, enactmentUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteEnactment(c *gin.Context) {
	db.GetCollection(pkgmodels.EnactmentCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStartEnactment(c *gin.Context) {
	db.GetCollection(pkgmodels.EnactmentCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"status": "active"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStopEnactment(c *gin.Context) {
	db.GetCollection(pkgmodels.EnactmentCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"status": "stopped"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddTriggerToEnactment(c *gin.Context) {
	enactmentId := c.Param("id")
	var body struct {
		TriggerId string `json:"trigger_id"`
	}
	c.ShouldBindJSON(&body)
	var tr pkgmodels.Trigger
	if err := db.GetCollection(pkgmodels.TriggerCollection).Find(bson.M{"public_id": body.TriggerId, "subscriber_id": auth.GetTenantID(c)}).One(&tr); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trigger not found"})
		return
	}
	eventKey := tr.TriggerType
	if eventKey == "" {
		eventKey = "default"
	}
	db.GetCollection(pkgmodels.EnactmentCollection).Update(
		bson.M{"public_id": enactmentId, "subscriber_id": auth.GetTenantID(c)},
		bson.M{"$push": bson.M{"trigger." + eventKey: tr}},
	)
	var updated pkgmodels.Enactment
	db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": enactmentId, "subscriber_id": auth.GetTenantID(c)}).One(&updated)
	c.JSON(http.StatusOK, gin.H{"enactment": updated})
}

func handleRemoveTriggerFromEnactment(c *gin.Context) {
	enactmentId := c.Param("id")
	triggerId := c.Param("triggerId")
	// Pull from all event keys in the trigger map
	var en pkgmodels.Enactment
	if err := db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": enactmentId, "subscriber_id": auth.GetTenantID(c)}).One(&en); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "enactment not found"})
		return
	}
	for key := range en.OnEvent {
		db.GetCollection(pkgmodels.EnactmentCollection).Update(
			bson.M{"public_id": enactmentId, "subscriber_id": auth.GetTenantID(c)},
			bson.M{"$pull": bson.M{"trigger." + key: bson.M{"public_id": triggerId}}},
		)
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleSetEnactmentScene(c *gin.Context) {
	enactmentId := c.Param("id")
	sceneId := c.Param("sceneId")
	var scene pkgmodels.Scene
	if err := db.GetCollection(pkgmodels.SceneCollection).Find(bson.M{"public_id": sceneId, "subscriber_id": auth.GetTenantID(c)}).One(&scene); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scene not found"})
		return
	}
	db.GetCollection(pkgmodels.EnactmentCollection).Update(
		bson.M{"public_id": enactmentId, "subscriber_id": auth.GetTenantID(c)},
		bson.M{"$set": bson.M{"send_scene": scene}},
	)
	var updated pkgmodels.Enactment
	db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": enactmentId, "subscriber_id": auth.GetTenantID(c)}).One(&updated)
	c.JSON(http.StatusOK, gin.H{"enactment": updated})
}

func handleRemoveEnactmentScene(c *gin.Context) {
	enactmentId := c.Param("id")
	db.GetCollection(pkgmodels.EnactmentCollection).Update(
		bson.M{"public_id": enactmentId, "subscriber_id": auth.GetTenantID(c)},
		bson.M{"$unset": bson.M{"send_scene": ""}},
	)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Scenes ───────────────────────────────────────────────

func handleGetScenes(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.Scene
	db.GetCollection(pkgmodels.SceneCollection).Find(bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}).All(&items)
	if items == nil {
		items = []pkgmodels.Scene{}
	}
	c.JSON(http.StatusOK, gin.H{"scenes": items})
}

func handleGetScene(c *gin.Context) {
	var item pkgmodels.Scene
	if err := db.GetCollection(pkgmodels.SceneCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scene not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"scene": item})
}

func handleCreateScene(c *gin.Context) {
	var item pkgmodels.Scene
	c.ShouldBindJSON(&item)
	item.SubscriberId = auth.GetTenantID(c)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.SceneCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"scene": item})
}

func handleUpdateScene(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.SceneCollection, sceneUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteScene(c *gin.Context) {
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddTagToScene(c *gin.Context) {
	var body struct {
		TagId string `json:"tag_id"`
	}
	c.ShouldBindJSON(&body)
	var tag pkgmodels.Tag
	if err := db.GetCollection(pkgmodels.TagCollection).Find(bson.M{"public_id": body.TagId, "subscriber_id": auth.GetTenantID(c)}).One(&tag); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$push": bson.M{"tags": tag}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
func handleRemoveTagFromScene(c *gin.Context) {
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$pull": bson.M{"tags": bson.M{"public_id": c.Param("tagId")}}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
func handleSetSceneMessage(c *gin.Context) {
	var msg pkgmodels.Message
	if err := db.GetCollection(pkgmodels.MessageCollection).Find(bson.M{"public_id": c.Param("messageId"), "subscriber_id": auth.GetTenantID(c)}).One(&msg); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"message": msg}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
func handleRemoveSceneMessage(c *gin.Context) {
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$unset": bson.M{"message": ""}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Messages ─────────────────────────────────────────────

func handleGetMessages(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.Message
	db.GetCollection(pkgmodels.MessageCollection).Find(bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}).All(&items)
	if items == nil {
		items = []pkgmodels.Message{}
	}
	c.JSON(http.StatusOK, gin.H{"messages": items})
}

func handleGetMessage(c *gin.Context) {
	var item pkgmodels.Message
	if err := db.GetCollection(pkgmodels.MessageCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": item})
}

func handleCreateMessage(c *gin.Context) {
	var raw struct {
		Name           string                    `json:"name"`
		Content        *pkgmodels.MessageContent `json:"content"`
		MessageContent *pkgmodels.MessageContent `json:"message_content"`
	}
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	content := raw.Content
	if content == nil {
		content = raw.MessageContent
	}
	// Content is optional: the Messages GUI creates the message first and
	// attaches content afterwards via the Content modal.
	item := pkgmodels.Message{
		SubscriberId: auth.GetTenantID(c),
		Name:         raw.Name,
		Content:      content,
	}
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.MessageCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"message": item})
}

func handleUpdateMessage(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.MessageCollection, messageUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteMessage(c *gin.Context) {
	db.GetCollection(pkgmodels.MessageCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddTagToMessage(c *gin.Context) {
	var body struct {
		TagId string `json:"tag_id"`
	}
	c.ShouldBindJSON(&body)
	var tag pkgmodels.Tag
	if err := db.GetCollection(pkgmodels.TagCollection).Find(bson.M{"public_id": body.TagId, "subscriber_id": auth.GetTenantID(c)}).One(&tag); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}
	db.GetCollection(pkgmodels.MessageCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$push": bson.M{"vars": tag}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
func handleRemoveTagFromMessage(c *gin.Context) {
	db.GetCollection(pkgmodels.MessageCollection).Update(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}, bson.M{"$pull": bson.M{"vars": bson.M{"public_id": c.Param("tagId")}}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Message Content ──────────────────────────────────────

func handleGetMessageContents(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.MessageContent
	db.GetCollection(pkgmodels.MessageContentCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.MessageContent{}
	}
	c.JSON(http.StatusOK, gin.H{"message_contents": items})
}

func handleGetMessageContent(c *gin.Context) {
	var item pkgmodels.MessageContent
	if err := db.GetCollection(pkgmodels.MessageContentCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message_content not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message_content": item})
}

func handleCreateMessageContent(c *gin.Context) {
	var item pkgmodels.MessageContent
	c.ShouldBindJSON(&item)
	item.SubscriberId = auth.GetTenantID(c)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.MessageContentCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"message_content": item})
}

func handleUpdateMessageContent(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.MessageContentCollection, messageContentUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteMessageContent(c *gin.Context) {
	db.GetCollection(pkgmodels.MessageContentCollection).Remove(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Triggers ─────────────────────────────────────────────

func handleGetTriggers(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.Trigger
	db.GetCollection(pkgmodels.TriggerCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.Trigger{}
	}
	c.JSON(http.StatusOK, gin.H{"triggers": items})
}

func handleGetTrigger(c *gin.Context) {
	var item pkgmodels.Trigger
	if err := db.GetCollection(pkgmodels.TriggerCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trigger not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"trigger": item})
}

func handleCreateTrigger(c *gin.Context) {
	var item pkgmodels.Trigger
	c.ShouldBindJSON(&item)
	item.SubscriberId = auth.GetTenantID(c)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.TriggerCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"trigger": item})
}

func handleUpdateTrigger(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.TriggerCollection, triggerUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteTrigger(c *gin.Context) {
	db.GetCollection(pkgmodels.TriggerCollection).Remove(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Actions ──────────────────────────────────────────────

func handleGetActions(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.Action
	db.GetCollection(pkgmodels.ActionCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.Action{}
	}
	c.JSON(http.StatusOK, gin.H{"actions": items})
}

func handleGetAction(c *gin.Context) {
	var item pkgmodels.Action
	if err := db.GetCollection(pkgmodels.ActionCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "action not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"action": item})
}

func handleCreateAction(c *gin.Context) {
	var item pkgmodels.Action
	c.ShouldBindJSON(&item)
	item.SubscriberId = auth.GetTenantID(c)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.ActionCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"action": item})
}

func handleUpdateAction(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.ActionCollection, actionUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteAction(c *gin.Context) {
	db.GetCollection(pkgmodels.ActionCollection).Remove(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Badges ───────────────────────────────────────────────

func handleGetBadges(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.Badge
	db.GetCollection(pkgmodels.BadgeCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.Badge{}
	}
	c.JSON(http.StatusOK, gin.H{"badges": items})
}

func handleGetBadge(c *gin.Context) {
	var item pkgmodels.Badge
	if err := db.GetCollection(pkgmodels.BadgeCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "badge not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"badge": item})
}

func handleCreateBadge(c *gin.Context) {
	var item pkgmodels.Badge
	c.ShouldBindJSON(&item)
	item.SubscriberId = auth.GetTenantID(c)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.BadgeCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"badge": item})
}

func handleUpdateBadge(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.BadgeCollection, badgeUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleDeleteBadge soft-deletes a badge definition (ID-013). Deletion is
// blocked (409) while the badge is still referenced: assigned to contacts or
// named by an offer's granted_badges — hard-removing those definitions
// silently changed access and broke provenance.
func handleDeleteBadge(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var badge pkgmodels.Badge
	if err := db.GetCollection(pkgmodels.BadgeCollection).Find(bson.M{
		"public_id": c.Param("id"), "subscriber_id": sid, "timestamps.deleted_at": nil,
	}).One(&badge); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "badge not found"})
		return
	}
	holders, _ := db.GetCollection(pkgmodels.UserCollection).Find(bson.M{
		"badges": badge.Id, "timestamps.deleted_at": nil,
	}).Count()
	offers, _ := db.GetCollection(pkgmodels.OfferCollection).Find(bson.M{
		"granted_badges": badge.Name, "subscriber_id": sid, "timestamps.deleted_at": nil,
	}).Count()
	if offers == 0 {
		offers, _ = db.GetCollection(pkgmodels.OfferCollection).Find(bson.M{
			"granted_badges": badge.Name, "tenant_id": badge.TenantID, "timestamps.deleted_at": nil,
		}).Count()
	}
	if holders > 0 || offers > 0 {
		c.JSON(http.StatusConflict, gin.H{
			"error":            "badge is still referenced; remove it from contacts and offers first",
			"contacts_holding": holders,
			"offers_granting":  offers,
		})
		return
	}
	_ = db.GetCollection(pkgmodels.BadgeCollection).Update(
		bson.M{"_id": badge.Id},
		bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}},
	)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleAssignBadgeToUser pushes badgeId onto User.Badges, dedupe-safe.
// Both :userId and :badgeId are public_ids (the rest of this file follows
// the same convention for cross-entity routes).
func handleAssignBadgeToUser(c *gin.Context) {
	userPub := c.Param("userId")
	badgePub := c.Param("badgeId")
	var badge pkgmodels.Badge
	if err := db.GetCollection(pkgmodels.BadgeCollection).Find(bson.M{"public_id": badgePub, "subscriber_id": auth.GetTenantID(c)}).One(&badge); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "badge not found"})
		return
	}
	var user pkgmodels.User
	if err := db.GetCollection(pkgmodels.UserCollection).Find(
		bson.M{"public_id": userPub, "tenant_id": auth.GetTenantObjectID(c)}).Select(bson.M{"_id": 1, "tenant_id": 1}).One(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	tenantOID := user.TenantID
	// ID-012: durable provenance with the acting account user as the actor.
	if _, err := badges.Assign(tenantOID, user.Id, badge.Id, "manual", "", auth.GetAccountUserID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "badge assignment failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "badge_id": badge.Id.Hex()})
}

// handleRemoveBadgeFromUser pulls badgeId from User.Badges. No-op if absent.
func handleRemoveBadgeFromUser(c *gin.Context) {
	userPub := c.Param("userId")
	badgePub := c.Param("badgeId")
	var badge pkgmodels.Badge
	if err := db.GetCollection(pkgmodels.BadgeCollection).Find(bson.M{"public_id": badgePub, "subscriber_id": auth.GetTenantID(c)}).One(&badge); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "badge not found"})
		return
	}
	var user pkgmodels.User
	if err := db.GetCollection(pkgmodels.UserCollection).Find(
		bson.M{"public_id": userPub, "tenant_id": auth.GetTenantObjectID(c)}).Select(bson.M{"_id": 1, "tenant_id": 1}).One(&user); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	tenantOID := user.TenantID
	if err := badges.Remove(tenantOID, user.Id, badge.Id, "manual", "", auth.GetAccountUserID(c)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "badge removal failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Tags ─────────────────────────────────────────────────

func handleGetTags(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.Tag
	db.GetCollection(pkgmodels.TagCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.Tag{}
	}
	c.JSON(http.StatusOK, gin.H{"tags": items})
}

func handleGetTag(c *gin.Context) {
	var item pkgmodels.Tag
	if err := db.GetCollection(pkgmodels.TagCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tag": item})
}

func handleCreateTag(c *gin.Context) {
	var item pkgmodels.Tag
	c.ShouldBindJSON(&item)
	item.SubscriberId = auth.GetTenantID(c)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.TagCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"tag": item})
}

func handleUpdateTag(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.TagCollection, tagUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// handleDeleteTag soft-deletes a tag definition (ID-013): the definition and
// any historical references stay auditable instead of vanishing.
func handleDeleteTag(c *gin.Context) {
	if err := db.GetCollection(pkgmodels.TagCollection).Update(
		bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c), "timestamps.deleted_at": nil},
		bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}},
	); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Template Variables ───────────────────────────────────

func handleGetTemplateVariables(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []bson.M
	db.GetCollection(pkgmodels.TemplateVariablesCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []bson.M{}
	}
	c.JSON(http.StatusOK, gin.H{"template_variables": items})
}

func handleGetTemplateVariable(c *gin.Context) {
	var item bson.M
	if err := db.GetCollection(pkgmodels.TemplateVariablesCollection).Find(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "template_variable not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"template_variable": item})
}

func handleCreateTemplateVariable(c *gin.Context) {
	var item bson.M
	c.ShouldBindJSON(&item)
	item["_id"] = bson.NewObjectId()
	item["public_id"] = utils.GeneratePublicId()
	item["subscriber_id"] = auth.GetTenantID(c)
	db.GetCollection(pkgmodels.TemplateVariablesCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"template_variable": item})
}

func handleUpdateTemplateVariable(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.TemplateVariablesCollection, templateVarFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteTemplateVariable(c *gin.Context) {
	db.GetCollection(pkgmodels.TemplateVariablesCollection).Remove(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Users ────────────────────────────────────────────────

type contactResponse struct {
	pkgmodels.User
	ConsentPending bool `json:"consent_pending"`
}

func toContactResponse(item pkgmodels.User) contactResponse {
	requestedConsent := item.ConsentSubscribed != nil && *item.ConsentSubscribed
	return contactResponse{
		User:           item,
		ConsentPending: !item.Subscribed && !requestedConsent && item.ConsentOptInDigest != "",
	}
}

func handleGetUsers(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	var items []pkgmodels.User
	db.GetCollection(pkgmodels.UserCollection).Find(bson.M{"tenant_id": tenantID, "timestamps.deleted_at": nil}).All(&items)
	out := make([]contactResponse, 0, len(items))
	for _, item := range items {
		out = append(out, toContactResponse(item))
	}
	c.JSON(http.StatusOK, gin.H{"users": out})
}

func handleGetUser(c *gin.Context) {
	var item pkgmodels.User
	if err := db.GetCollection(pkgmodels.UserCollection).Find(bson.M{"public_id": c.Param("id"), "tenant_id": auth.GetTenantObjectID(c), "timestamps.deleted_at": nil}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": toContactResponse(item)})
}

func handleCreateUser(c *gin.Context) {
	var item pkgmodels.User
	c.ShouldBindJSON(&item)
	// Admin-driven creation gets an explicit refusal (public capture paths
	// hold the contact instead) — the admin can act on the upgrade prompt.
	if tid := auth.GetTenantObjectID(c); tid != "" && item.Subscribed && plans.ContactCreationBlocked(tid) {
		c.JSON(http.StatusPaymentRequired, gin.H{
			"error": "contact limit reached — upgrade your plan to add more contacts",
			"code":  "contact_limit_reached",
		})
		return
	}
	// tenant_id is the single contact ownership key (ID-007).
	item.TenantID = auth.GetTenantObjectID(c)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.UserCollection).Insert(item)
	if item.TenantID != "" {
		plans.Invalidate(item.TenantID)
	}
	c.JSON(http.StatusOK, gin.H{"user": item})
}

func handleUpdateUser(c *gin.Context) {
	if !applySanitizedUpdate(c, pkgmodels.UserCollection, contactUpdateFields) {
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteUser(c *gin.Context) {
	db.GetCollection(pkgmodels.UserCollection).Update(bson.M{"public_id": c.Param("id"), "tenant_id": auth.GetTenantObjectID(c)}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteAllUsers(c *gin.Context) {
	tenantID := auth.GetTenantObjectID(c)
	db.GetCollection(pkgmodels.UserCollection).UpdateAll(bson.M{"tenant_id": tenantID}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleGetUserDetail(c *gin.Context) {
	var item pkgmodels.User
	db.GetCollection(pkgmodels.UserCollection).Find(bson.M{"public_id": c.Param("id"), "tenant_id": auth.GetTenantObjectID(c)}).One(&item)
	c.JSON(http.StatusOK, gin.H{"user": toContactResponse(item), "campaign": nil, "hot_triggers": []interface{}{}, "pending_emails": []interface{}{}})
}

func handleAddUserToStory(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

// registerUserRequest is the narrow, server-validated contact-creation DTO
// (ID-002). It carries only client-editable contact fields — never tenant_id,
// subscriber_id, badges, story state, password hashes, or Stripe IDs, which
// are server-owned. `subscriber_id`/`domain` are accepted for routing only:
// the tenant is resolved from verified context, and a body subscriber_id that
// disagrees with that context is rejected rather than trusted.
type registerUserRequest struct {
	Email        string `json:"email"`
	FirstName    string `json:"first_name"`
	MiddleName   string `json:"middle_name"`
	LastName     string `json:"last_name"`
	Phone        string `json:"phone"`
	ListID       string `json:"list_id"`
	Subscribed   bool   `json:"subscribed"`
	SubscriberID string `json:"subscriber_id"`
	Domain       string `json:"domain"`
}

// resolveRegistrationTenant resolves the tenant for a public contact-creation
// request from verified context only (ID-002): a tenant JWT (admin dashboard),
// an X-API-Key (documented API clients), or a public channel/domain. It never
// trusts a caller-supplied subscriber_id/tenant_id as authority.
func resolveRegistrationTenant(c *gin.Context, bodyDomain string) (bson.ObjectId, bool) {
	if tid, ok := auth.ResolveTenantByJWT(c); ok {
		return tid, true
	}
	if tid, ok := auth.ResolveTenantByAPIKey(c); ok {
		return tid, true
	}
	if pctx, err := publicchannel.ResolvePublicRequestWithDomain(c, bodyDomain); err == nil && pctx.TenantID != "" {
		return pctx.TenantID, true
	}
	return "", false
}

// HandleRegisterUser registers a new end-user/subscriber. Public surface, but
// the owning tenant is resolved from verified context — a JWT, an API key, or a
// verified channel/domain — not from a caller-chosen subscriber_id (ID-002).
func HandleRegisterUser(c *gin.Context) {
	var req registerUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	tenantID, ok := resolveRegistrationTenant(c, req.Domain)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unresolved tenant: provide a tenant token, API key, or verified domain"})
		return
	}
	// A body subscriber_id is honored only as a routing hint; if it disagrees
	// with the verified tenant it is a cross-tenant injection attempt.
	if req.SubscriberID != "" && req.SubscriberID != tenantID.Hex() {
		c.JSON(http.StatusForbidden, gin.H{"error": "subscriber_id does not match authenticated tenant"})
		return
	}
	if req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "email is required"})
		return
	}

	// Reject duplicate contacts within the tenant rather than silently
	// creating a second record (ID-008 groundwork).
	existing, _ := db.GetCollection(pkgmodels.UserCollection).Find(bson.M{
		"tenant_id":             tenantID,
		"email":                 req.Email,
		"timestamps.deleted_at": nil,
	}).Count()
	if existing > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "contact already exists for this tenant"})
		return
	}

	var item pkgmodels.User
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.TenantID = tenantID
	item.Email = pkgmodels.EmailAddress(req.Email)
	item.Name.First = req.FirstName
	item.Name.Middle = req.MiddleName
	item.Name.Last = req.LastName
	item.Phone = req.Phone
	item.Subscribed = req.Subscribed
	if bson.IsObjectIdHex(req.ListID) {
		item.EmailList = bson.ObjectIdHex(req.ListID)
	}
	item.SoftDeletes.CreatedAt = now()
	plans.ApplyHold(&item)
	if err := db.GetCollection(pkgmodels.UserCollection).Insert(item); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create contact"})
		return
	}
	plans.Invalidate(tenantID)
	c.JSON(http.StatusOK, gin.H{"user": item})
}

// ─── Email Lists ──────────────────────────────────────────

func handleGetEmailLists(c *gin.Context) {
	sid := auth.GetTenantID(c)
	var items []pkgmodels.EmailList
	db.GetCollection(pkgmodels.EmailListCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.EmailList{}
	}
	c.JSON(http.StatusOK, gin.H{"lists": items})
}

func handleGetEmailList(c *gin.Context) {
	id := c.Param("id")
	if !bson.IsObjectIdHex(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "list not found"})
		return
	}
	var item pkgmodels.EmailList
	if err := db.GetCollection(pkgmodels.EmailListCollection).Find(bson.M{"_id": bson.ObjectIdHex(id), "subscriber_id": auth.GetTenantID(c)}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "list not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"list": item})
}

func handleCreateEmailList(c *gin.Context) {
	var item pkgmodels.EmailList
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SubscriberId = auth.GetTenantID(c)
	db.GetCollection(pkgmodels.EmailListCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"list": item})
}

func handleDeleteEmailList(c *gin.Context) {
	db.GetCollection(pkgmodels.EmailListCollection).Remove(bson.M{"public_id": c.Param("id"), "subscriber_id": auth.GetTenantID(c)})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Email Queue / Hot Triggers ───────────────────────────

func handleGetPendingEmails(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"scheduled_emails": []interface{}{}, "instant_emails": []interface{}{}, "total": 0})
}

func handleGetHotTriggers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"hot_triggers": []interface{}{}, "total": 0})
}

// ─── Stats ────────────────────────────────────────────────

func handleStatsOverview(c *gin.Context) {
	sid := auth.GetTenantID(c)
	alive := bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}
	count := func(coll string) int {
		n, _ := db.GetCollection(coll).Find(alive).Count()
		return n
	}
	activeSessions, _ := db.GetCollection(storySessionCollection).Find(bson.M{"subscriber_id": sid, "status": "active"}).Count()
	completedSessions, _ := db.GetCollection(storySessionCollection).Find(bson.M{"subscriber_id": sid, "status": "completed"}).Count()
	c.JSON(http.StatusOK, gin.H{"status": "OK", "stats": gin.H{
		"campaigns":          count(pkgmodels.StoryCollection),
		"storylines":         count(pkgmodels.StorylineCollection),
		"messages":           count(pkgmodels.MessageCollection),
		"contacts":           count(pkgmodels.UserCollection),
		"badges":             count(pkgmodels.BadgeCollection),
		"triggers":           count(pkgmodels.TriggerCollection),
		"active_enrollments": activeSessions,
		"completed_stories":  completedSessions,
	}})
}

func handleStatsStub(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "OK", "data": []interface{}{}})
}

// ─── Admin ────────────────────────────────────────────────

func handleAdminReset(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
