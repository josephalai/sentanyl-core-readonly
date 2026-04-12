package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/utils"
)

// RegisterStoryRoutes wires all story-builder CRUD endpoints onto the root engine.
func RegisterStoryRoutes(r *gin.Engine) {
	// Stories
	r.GET("/api/stories", handleGetStories)
	r.GET("/api/story/:id", handleGetStory)
	r.POST("/api/story/", handleCreateStory)
	r.PUT("/api/story/:id", handleUpdateStory)
	r.DELETE("/api/story/:id", handleDeleteStory)
	r.DELETE("/api/stories", handleDeleteAllStories)
	r.DELETE("/api/story/:id/purge", handlePurgeStory)
	r.PUT("/api/story/:id/start", handleStartStory)
	r.PUT("/api/story/:id/stop", handleStopStory)
	r.POST("/api/story/:id/storylines/:storylineId", handleAddStorylineToStory)
	r.DELETE("/api/story/:id/storylines/:storylineId", handleRemoveStorylineFromStory)

	// Storylines
	r.GET("/api/storylines", handleGetStorylines)
	r.GET("/api/storyline/:id", handleGetStoryline)
	r.POST("/api/storyline/", handleCreateStoryline)
	r.PUT("/api/storyline/:id", handleUpdateStoryline)
	r.DELETE("/api/storyline/:id", handleDeleteStoryline)
	r.PUT("/api/storyline/:id/start", handleStartStoryline)
	r.PUT("/api/storyline/:id/stop", handleStopStoryline)
	r.POST("/api/storyline/:id/enactments", handleAddEnactmentToStoryline)
	r.DELETE("/api/storyline/:id/enactments/:enactmentId", handleRemoveEnactmentFromStoryline)

	// Enactments
	r.GET("/api/enactments", handleGetEnactments)
	r.GET("/api/enactment/:id", handleGetEnactment)
	r.POST("/api/enactment/", handleCreateEnactment)
	r.PUT("/api/enactment/:id", handleUpdateEnactment)
	r.DELETE("/api/enactment/:id", handleDeleteEnactment)
	r.PUT("/api/enactment/:id/start", handleStartEnactment)
	r.PUT("/api/enactment/:id/stop", handleStopEnactment)
	r.POST("/api/enactment/:id/trigger", handleAddTriggerToEnactment)
	r.DELETE("/api/enactment/:id/trigger/:triggerId", handleRemoveTriggerFromEnactment)
	r.POST("/api/enactment/:id/scene/:sceneId", handleSetEnactmentScene)
	r.DELETE("/api/enactment/:id/scene", handleRemoveEnactmentScene)

	// Scenes
	r.GET("/api/scenes", handleGetScenes)
	r.GET("/api/scene/:id", handleGetScene)
	r.POST("/api/scene/", handleCreateScene)
	r.PUT("/api/scene/:id", handleUpdateScene)
	r.DELETE("/api/scene/:id", handleDeleteScene)
	r.POST("/api/scene/:id/tag", handleAddTagToScene)
	r.DELETE("/api/scene/:id/tag/:tagId", handleRemoveTagFromScene)
	r.POST("/api/scene/:id/message/:messageId", handleSetSceneMessage)
	r.DELETE("/api/scene/:id/message", handleRemoveSceneMessage)

	// Messages
	r.GET("/api/messages", handleGetMessages)
	r.GET("/api/message/:id", handleGetMessage)
	r.POST("/api/message/", handleCreateMessage)
	r.PUT("/api/message/:id", handleUpdateMessage)
	r.DELETE("/api/message/:id", handleDeleteMessage)
	r.POST("/api/message/:id/tag", handleAddTagToMessage)
	r.DELETE("/api/message/:id/tag/:tagId", handleRemoveTagFromMessage)

	// Message Content
	r.GET("/api/message_contents", handleGetMessageContents)
	r.GET("/api/message_content/:id", handleGetMessageContent)
	r.POST("/api/message_content/", handleCreateMessageContent)
	r.PUT("/api/message_content/:id", handleUpdateMessageContent)
	r.DELETE("/api/message_content/:id", handleDeleteMessageContent)

	// Triggers
	r.GET("/api/triggers", handleGetTriggers)
	r.GET("/api/trigger/:id", handleGetTrigger)
	r.POST("/api/trigger/", handleCreateTrigger)
	r.PUT("/api/trigger/:id", handleUpdateTrigger)
	r.DELETE("/api/trigger/:id", handleDeleteTrigger)

	// Actions
	r.GET("/api/actions", handleGetActions)
	r.GET("/api/action/:id", handleGetAction)
	r.POST("/api/action/", handleCreateAction)
	r.PUT("/api/action/:id", handleUpdateAction)
	r.DELETE("/api/action/:id", handleDeleteAction)

	// Badges
	r.GET("/api/badges", handleGetBadges)
	r.GET("/api/badge/:id", handleGetBadge)
	r.POST("/api/badge/", handleCreateBadge)
	r.PUT("/api/badge/:id", handleUpdateBadge)
	r.DELETE("/api/badge/:id", handleDeleteBadge)
	r.PUT("/api/user_badge/user/:userId/badge/:badgeId", handleAssignBadgeToUser)
	r.DELETE("/api/user_badge/user/:userId/badge/:badgeId", handleRemoveBadgeFromUser)

	// Tags
	r.GET("/api/tags", handleGetTags)
	r.GET("/api/tag/:id", handleGetTag)
	r.POST("/api/tag/", handleCreateTag)
	r.PUT("/api/tag/:id", handleUpdateTag)
	r.DELETE("/api/tag/:id", handleDeleteTag)

	// Template Variables
	r.GET("/api/template_variables", handleGetTemplateVariables)
	r.GET("/api/template_variable/:id", handleGetTemplateVariable)
	r.POST("/api/template_variable/", handleCreateTemplateVariable)
	r.PUT("/api/template_variable/:id", handleUpdateTemplateVariable)
	r.DELETE("/api/template_variable/:id", handleDeleteTemplateVariable)

	// Users / CRM
	r.GET("/api/users", handleGetUsers)
	r.GET("/api/user/:id", handleGetUser)
	r.POST("/api/user/", handleCreateUser)
	r.PUT("/api/user/:id", handleUpdateUser)
	r.DELETE("/api/user/:id", handleDeleteUser)
	r.DELETE("/api/users", handleDeleteAllUsers)
	r.GET("/api/user/:id/detail", handleGetUserDetail)
	r.POST("/api/user/:id/story/:storyId", handleAddUserToStory)
	r.POST("/api/register/user", handleRegisterUser)

	// Email lists
	r.GET("/api/creator/lists", handleGetEmailLists)
	r.GET("/api/creator/list/:id", handleGetEmailList)
	r.POST("/api/creator/list", handleCreateEmailList)
	r.DELETE("/api/creator/list/:id", handleDeleteEmailList)

	// Email queue / hot triggers
	r.GET("/api/emails/pending", handleGetPendingEmails)
	r.GET("/api/hot-triggers", handleGetHotTriggers)

	// Stats (stubs — returns empty but won't 404)
	r.GET("/api/stats/", handleStatsOverview)
	r.GET("/api/stats/story", handleStatsStub)
	r.GET("/api/stats/story/:id", handleStatsStub)
	r.GET("/api/stats/storyline", handleStatsStub)
	r.GET("/api/stats/storyline/:id", handleStatsStub)
	r.GET("/api/stats/enactment", handleStatsStub)
	r.GET("/api/stats/enactment/:id", handleStatsStub)
	r.GET("/api/stats/message", handleStatsStub)
	r.GET("/api/stats/message/:id", handleStatsStub)
	r.GET("/api/stats/badge", handleStatsStub)
	r.GET("/api/stats/badge/:id", handleStatsStub)
	r.GET("/api/stats/trigger", handleStatsStub)
	r.GET("/api/stats/trigger/:id", handleStatsStub)
	r.GET("/api/stats/email", handleStatsStub)
	r.GET("/api/stats/email/:id", handleStatsStub)
	r.GET("/api/stats/user", handleStatsStub)
	r.GET("/api/stats/user/:id", handleStatsStub)
	r.GET("/api/stats/link", handleStatsStub)
	r.GET("/api/stats/link/:id", handleStatsStub)
	r.GET("/api/stats/spam", handleStatsStub)
	r.GET("/api/stats/fail", handleStatsStub)
	r.GET("/api/stats/validate", handleStatsStub)
	r.GET("/api/stats/ab", handleStatsStub)
	r.GET("/api/stats/ab/:id", handleStatsStub)

	// Admin
	r.POST("/api/admin/reset", handleAdminReset)
}

// ─── Helpers ──────────────────────────────────────────────

func subID(c *gin.Context) string {
	if s := c.Query("subscriber_id"); s != "" {
		return s
	}
	var body struct {
		SubscriberID string `json:"subscriber_id"`
	}
	_ = c.ShouldBindJSON(&body)
	return body.SubscriberID
}

func now() *time.Time { t := time.Now(); return &t }

// ─── Stories ──────────────────────────────────────────────

func handleGetStories(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.Story
	db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}).All(&items)
	if items == nil {
		items = []pkgmodels.Story{}
	}
	c.JSON(http.StatusOK, gin.H{"stories": items})
}

func handleGetStory(c *gin.Context) {
	var item pkgmodels.Story
	if err := db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "story not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"story": item})
}

func handleCreateStory(c *gin.Context) {
	var item pkgmodels.Story
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
		return
	}
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.StoryCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"story": item})
}

func handleUpdateStory(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.StoryCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteStory(c *gin.Context) {
	db.GetCollection(pkgmodels.StoryCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteAllStories(c *gin.Context) {
	var body struct{ SubscriberID string `json:"subscriber_id"` }
	c.ShouldBindJSON(&body)
	db.GetCollection(pkgmodels.StoryCollection).UpdateAll(bson.M{"subscriber_id": body.SubscriberID}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handlePurgeStory(c *gin.Context) {
	db.GetCollection(pkgmodels.StoryCollection).Remove(bson.M{"public_id": c.Param("id")})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStartStory(c *gin.Context) {
	db.GetCollection(pkgmodels.StoryCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"status": "active"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStopStory(c *gin.Context) {
	db.GetCollection(pkgmodels.StoryCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"status": "stopped"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddStorylineToStory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleRemoveStorylineFromStory(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Storylines ───────────────────────────────────────────

func handleGetStorylines(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.Storyline
	db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}).All(&items)
	if items == nil {
		items = []pkgmodels.Storyline{}
	}
	c.JSON(http.StatusOK, gin.H{"storylines": items})
}

func handleGetStoryline(c *gin.Context) {
	var item pkgmodels.Storyline
	if err := db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "storyline not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"storyline": item})
}

func handleCreateStoryline(c *gin.Context) {
	var item pkgmodels.Storyline
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.StorylineCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"storyline": item})
}

func handleUpdateStoryline(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.StorylineCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteStoryline(c *gin.Context) {
	db.GetCollection(pkgmodels.StorylineCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStartStoryline(c *gin.Context) {
	db.GetCollection(pkgmodels.StorylineCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"status": "active"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStopStoryline(c *gin.Context) {
	db.GetCollection(pkgmodels.StorylineCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"status": "stopped"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddEnactmentToStoryline(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
func handleRemoveEnactmentFromStoryline(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

// ─── Enactments ───────────────────────────────────────────

func handleGetEnactments(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.Enactment
	db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}).All(&items)
	if items == nil {
		items = []pkgmodels.Enactment{}
	}
	c.JSON(http.StatusOK, gin.H{"enactments": items})
}

func handleGetEnactment(c *gin.Context) {
	var item pkgmodels.Enactment
	if err := db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "enactment not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enactment": item})
}

func handleCreateEnactment(c *gin.Context) {
	var item pkgmodels.Enactment
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.EnactmentCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"enactment": item})
}

func handleUpdateEnactment(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.EnactmentCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteEnactment(c *gin.Context) {
	db.GetCollection(pkgmodels.EnactmentCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStartEnactment(c *gin.Context) {
	db.GetCollection(pkgmodels.EnactmentCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"status": "active"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleStopEnactment(c *gin.Context) {
	db.GetCollection(pkgmodels.EnactmentCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"status": "stopped"}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddTriggerToEnactment(c *gin.Context)      { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
func handleRemoveTriggerFromEnactment(c *gin.Context)  { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
func handleSetEnactmentScene(c *gin.Context)           { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
func handleRemoveEnactmentScene(c *gin.Context)        { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

// ─── Scenes ───────────────────────────────────────────────

func handleGetScenes(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.Scene
	db.GetCollection(pkgmodels.SceneCollection).Find(bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}).All(&items)
	if items == nil {
		items = []pkgmodels.Scene{}
	}
	c.JSON(http.StatusOK, gin.H{"scenes": items})
}

func handleGetScene(c *gin.Context) {
	var item pkgmodels.Scene
	if err := db.GetCollection(pkgmodels.SceneCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scene not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"scene": item})
}

func handleCreateScene(c *gin.Context) {
	var item pkgmodels.Scene
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.SceneCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"scene": item})
}

func handleUpdateScene(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteScene(c *gin.Context) {
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddTagToScene(c *gin.Context)      { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
func handleRemoveTagFromScene(c *gin.Context)  { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
func handleSetSceneMessage(c *gin.Context)     { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
func handleRemoveSceneMessage(c *gin.Context)  { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

// ─── Messages ─────────────────────────────────────────────

func handleGetMessages(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.Message
	db.GetCollection(pkgmodels.MessageCollection).Find(bson.M{"subscriber_id": sid, "timestamps.deleted_at": nil}).All(&items)
	if items == nil {
		items = []pkgmodels.Message{}
	}
	c.JSON(http.StatusOK, gin.H{"messages": items})
}

func handleGetMessage(c *gin.Context) {
	var item pkgmodels.Message
	if err := db.GetCollection(pkgmodels.MessageCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": item})
}

func handleCreateMessage(c *gin.Context) {
	var item pkgmodels.Message
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.MessageCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"message": item})
}

func handleUpdateMessage(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.MessageCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteMessage(c *gin.Context) {
	db.GetCollection(pkgmodels.MessageCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddTagToMessage(c *gin.Context)     { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
func handleRemoveTagFromMessage(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

// ─── Message Content ──────────────────────────────────────

func handleGetMessageContents(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.MessageContent
	db.GetCollection(pkgmodels.MessageContentCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.MessageContent{}
	}
	c.JSON(http.StatusOK, gin.H{"message_contents": items})
}

func handleGetMessageContent(c *gin.Context) {
	var item pkgmodels.MessageContent
	if err := db.GetCollection(pkgmodels.MessageContentCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message_content not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message_content": item})
}

func handleCreateMessageContent(c *gin.Context) {
	var item pkgmodels.MessageContent
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.MessageContentCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"message_content": item})
}

func handleUpdateMessageContent(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.MessageContentCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteMessageContent(c *gin.Context) {
	db.GetCollection(pkgmodels.MessageContentCollection).Remove(bson.M{"public_id": c.Param("id")})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Triggers ─────────────────────────────────────────────

func handleGetTriggers(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.Trigger
	db.GetCollection(pkgmodels.TriggerCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.Trigger{}
	}
	c.JSON(http.StatusOK, gin.H{"triggers": items})
}

func handleGetTrigger(c *gin.Context) {
	var item pkgmodels.Trigger
	if err := db.GetCollection(pkgmodels.TriggerCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trigger not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"trigger": item})
}

func handleCreateTrigger(c *gin.Context) {
	var item pkgmodels.Trigger
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.TriggerCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"trigger": item})
}

func handleUpdateTrigger(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.TriggerCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteTrigger(c *gin.Context) {
	db.GetCollection(pkgmodels.TriggerCollection).Remove(bson.M{"public_id": c.Param("id")})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Actions ──────────────────────────────────────────────

func handleGetActions(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.Action
	db.GetCollection(pkgmodels.ActionCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.Action{}
	}
	c.JSON(http.StatusOK, gin.H{"actions": items})
}

func handleGetAction(c *gin.Context) {
	var item pkgmodels.Action
	if err := db.GetCollection(pkgmodels.ActionCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "action not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"action": item})
}

func handleCreateAction(c *gin.Context) {
	var item pkgmodels.Action
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.ActionCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"action": item})
}

func handleUpdateAction(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.ActionCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteAction(c *gin.Context) {
	db.GetCollection(pkgmodels.ActionCollection).Remove(bson.M{"public_id": c.Param("id")})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Badges ───────────────────────────────────────────────

func handleGetBadges(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.Badge
	db.GetCollection(pkgmodels.BadgeCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.Badge{}
	}
	c.JSON(http.StatusOK, gin.H{"badges": items})
}

func handleGetBadge(c *gin.Context) {
	var item pkgmodels.Badge
	if err := db.GetCollection(pkgmodels.BadgeCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "badge not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"badge": item})
}

func handleCreateBadge(c *gin.Context) {
	var item pkgmodels.Badge
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.BadgeCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"badge": item})
}

func handleUpdateBadge(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.BadgeCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteBadge(c *gin.Context) {
	db.GetCollection(pkgmodels.BadgeCollection).Remove(bson.M{"public_id": c.Param("id")})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAssignBadgeToUser(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
func handleRemoveBadgeFromUser(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

// ─── Tags ─────────────────────────────────────────────────

func handleGetTags(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.Tag
	db.GetCollection(pkgmodels.TagCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.Tag{}
	}
	c.JSON(http.StatusOK, gin.H{"tags": items})
}

func handleGetTag(c *gin.Context) {
	var item pkgmodels.Tag
	if err := db.GetCollection(pkgmodels.TagCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tag": item})
}

func handleCreateTag(c *gin.Context) {
	var item pkgmodels.Tag
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.TagCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"tag": item})
}

func handleUpdateTag(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.TagCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteTag(c *gin.Context) {
	db.GetCollection(pkgmodels.TagCollection).Remove(bson.M{"public_id": c.Param("id")})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Template Variables ───────────────────────────────────

func handleGetTemplateVariables(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []bson.M
	db.GetCollection(pkgmodels.TemplateVariablesCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []bson.M{}
	}
	c.JSON(http.StatusOK, gin.H{"template_variables": items})
}

func handleGetTemplateVariable(c *gin.Context) {
	var item bson.M
	if err := db.GetCollection(pkgmodels.TemplateVariablesCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
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
	db.GetCollection(pkgmodels.TemplateVariablesCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"template_variable": item})
}

func handleUpdateTemplateVariable(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.TemplateVariablesCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteTemplateVariable(c *gin.Context) {
	db.GetCollection(pkgmodels.TemplateVariablesCollection).Remove(bson.M{"public_id": c.Param("id")})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ─── Users ────────────────────────────────────────────────

func handleGetUsers(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.User
	db.GetCollection(pkgmodels.UserCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.User{}
	}
	c.JSON(http.StatusOK, gin.H{"users": items})
}

func handleGetUser(c *gin.Context) {
	var item pkgmodels.User
	if err := db.GetCollection(pkgmodels.UserCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": item})
}

func handleCreateUser(c *gin.Context) {
	var item pkgmodels.User
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.UserCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"user": item})
}

func handleUpdateUser(c *gin.Context) {
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.UserCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteUser(c *gin.Context) {
	db.GetCollection(pkgmodels.UserCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteAllUsers(c *gin.Context) {
	var body struct{ SubscriberID string `json:"subscriber_id"` }
	c.ShouldBindJSON(&body)
	db.GetCollection(pkgmodels.UserCollection).UpdateAll(bson.M{"subscriber_id": body.SubscriberID}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleGetUserDetail(c *gin.Context) {
	var item pkgmodels.User
	db.GetCollection(pkgmodels.UserCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item)
	c.JSON(http.StatusOK, gin.H{"user": item, "campaign": nil, "hot_triggers": []interface{}{}, "pending_emails": []interface{}{}})
}

func handleAddUserToStory(c *gin.Context)  { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

func handleRegisterUser(c *gin.Context) {
	var item pkgmodels.User
	c.ShouldBindJSON(&item)
	item.Id = bson.NewObjectId()
	item.PublicId = utils.GeneratePublicId()
	item.SoftDeletes.CreatedAt = now()
	db.GetCollection(pkgmodels.UserCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"user": item})
}

// ─── Email Lists ──────────────────────────────────────────

func handleGetEmailLists(c *gin.Context) {
	sid := c.Query("subscriber_id")
	var items []pkgmodels.EmailList
	db.GetCollection(pkgmodels.EmailListCollection).Find(bson.M{"subscriber_id": sid}).All(&items)
	if items == nil {
		items = []pkgmodels.EmailList{}
	}
	c.JSON(http.StatusOK, gin.H{"lists": items})
}

func handleGetEmailList(c *gin.Context) {
	var item pkgmodels.EmailList
	if err := db.GetCollection(pkgmodels.EmailListCollection).FindId(bson.ObjectIdHex(c.Param("id"))).One(&item); err != nil {
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
	db.GetCollection(pkgmodels.EmailListCollection).Insert(item)
	c.JSON(http.StatusOK, gin.H{"list": item})
}

func handleDeleteEmailList(c *gin.Context) {
	db.GetCollection(pkgmodels.EmailListCollection).Remove(bson.M{"public_id": c.Param("id")})
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
	c.JSON(http.StatusOK, gin.H{"status": "OK", "stats": gin.H{}})
}

func handleStatsStub(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "OK", "data": []interface{}{}})
}

// ─── Admin ────────────────────────────────────────────────

func handleAdminReset(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
