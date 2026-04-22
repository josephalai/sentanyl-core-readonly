package routes

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/mgo.v2/bson"

	"github.com/josephalai/sentanyl/pkg/auth"
	"github.com/josephalai/sentanyl/pkg/db"
	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
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
	r.GET("/stats/story/:id", handleStatsStub)
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

	// Admin
	r.POST("/admin/reset", handleAdminReset)
}

// ─── Helpers ──────────────────────────────────────────────

func now() *time.Time { t := time.Now(); return &t }

// ─── Stories ──────────────────────────────────────────────

func handleGetStories(c *gin.Context) {
	sid := auth.GetTenantID(c)
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
	item.SubscriberId = auth.GetTenantID(c)
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
	sid := auth.GetTenantID(c)
	db.GetCollection(pkgmodels.StoryCollection).UpdateAll(bson.M{"subscriber_id": sid}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
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
	storyId := c.Param("id")
	storylineId := c.Param("storylineId")
	var sl pkgmodels.Storyline
	if err := db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": storylineId}).One(&sl); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "storyline not found"})
		return
	}
	db.GetCollection(pkgmodels.StoryCollection).Update(
		bson.M{"public_id": storyId},
		bson.M{"$push": bson.M{"storylines": sl}},
	)
	var updated pkgmodels.Story
	db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{"public_id": storyId}).One(&updated)
	c.JSON(http.StatusOK, gin.H{"story": updated})
}

func handleRemoveStorylineFromStory(c *gin.Context) {
	storyId := c.Param("id")
	storylineId := c.Param("storylineId")
	db.GetCollection(pkgmodels.StoryCollection).Update(
		bson.M{"public_id": storyId},
		bson.M{"$pull": bson.M{"storylines": bson.M{"public_id": storylineId}}},
	)
	var updated pkgmodels.Story
	db.GetCollection(pkgmodels.StoryCollection).Find(bson.M{"public_id": storyId}).One(&updated)
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
	if err := db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
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

func handleAddEnactmentToStoryline(c *gin.Context) {
	storylineId := c.Param("id")
	var body struct {
		EnactmentId string `json:"enactment_id"`
	}
	c.ShouldBindJSON(&body)
	var en pkgmodels.Enactment
	if err := db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": body.EnactmentId}).One(&en); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "enactment not found"})
		return
	}
	db.GetCollection(pkgmodels.StorylineCollection).Update(
		bson.M{"public_id": storylineId},
		bson.M{"$push": bson.M{"acts": en}},
	)
	var updated pkgmodels.Storyline
	db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": storylineId}).One(&updated)
	c.JSON(http.StatusOK, gin.H{"storyline": updated})
}

func handleRemoveEnactmentFromStoryline(c *gin.Context) {
	storylineId := c.Param("id")
	enactmentId := c.Param("enactmentId")
	db.GetCollection(pkgmodels.StorylineCollection).Update(
		bson.M{"public_id": storylineId},
		bson.M{"$pull": bson.M{"acts": bson.M{"public_id": enactmentId}}},
	)
	var updated pkgmodels.Storyline
	db.GetCollection(pkgmodels.StorylineCollection).Find(bson.M{"public_id": storylineId}).One(&updated)
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
	if err := db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
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

func handleAddTriggerToEnactment(c *gin.Context) {
	enactmentId := c.Param("id")
	var body struct {
		TriggerId string `json:"trigger_id"`
	}
	c.ShouldBindJSON(&body)
	var tr pkgmodels.Trigger
	if err := db.GetCollection(pkgmodels.TriggerCollection).Find(bson.M{"public_id": body.TriggerId}).One(&tr); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trigger not found"})
		return
	}
	eventKey := tr.TriggerType
	if eventKey == "" {
		eventKey = "default"
	}
	db.GetCollection(pkgmodels.EnactmentCollection).Update(
		bson.M{"public_id": enactmentId},
		bson.M{"$push": bson.M{"trigger." + eventKey: tr}},
	)
	var updated pkgmodels.Enactment
	db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": enactmentId}).One(&updated)
	c.JSON(http.StatusOK, gin.H{"enactment": updated})
}

func handleRemoveTriggerFromEnactment(c *gin.Context) {
	enactmentId := c.Param("id")
	triggerId := c.Param("triggerId")
	// Pull from all event keys in the trigger map
	var en pkgmodels.Enactment
	if err := db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": enactmentId}).One(&en); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "enactment not found"})
		return
	}
	for key := range en.OnEvent {
		db.GetCollection(pkgmodels.EnactmentCollection).Update(
			bson.M{"public_id": enactmentId},
			bson.M{"$pull": bson.M{"trigger." + key: bson.M{"public_id": triggerId}}},
		)
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleSetEnactmentScene(c *gin.Context) {
	enactmentId := c.Param("id")
	sceneId := c.Param("sceneId")
	var scene pkgmodels.Scene
	if err := db.GetCollection(pkgmodels.SceneCollection).Find(bson.M{"public_id": sceneId}).One(&scene); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "scene not found"})
		return
	}
	db.GetCollection(pkgmodels.EnactmentCollection).Update(
		bson.M{"public_id": enactmentId},
		bson.M{"$set": bson.M{"send_scene": scene}},
	)
	var updated pkgmodels.Enactment
	db.GetCollection(pkgmodels.EnactmentCollection).Find(bson.M{"public_id": enactmentId}).One(&updated)
	c.JSON(http.StatusOK, gin.H{"enactment": updated})
}

func handleRemoveEnactmentScene(c *gin.Context) {
	enactmentId := c.Param("id")
	db.GetCollection(pkgmodels.EnactmentCollection).Update(
		bson.M{"public_id": enactmentId},
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
	if err := db.GetCollection(pkgmodels.SceneCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
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
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteScene(c *gin.Context) {
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddTagToScene(c *gin.Context) {
	var body struct{ TagId string `json:"tag_id"` }
	c.ShouldBindJSON(&body)
	var tag pkgmodels.Tag
	if err := db.GetCollection(pkgmodels.TagCollection).Find(bson.M{"public_id": body.TagId}).One(&tag); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"}); return
	}
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$push": bson.M{"tags": tag}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
func handleRemoveTagFromScene(c *gin.Context) {
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$pull": bson.M{"tags": bson.M{"public_id": c.Param("tagId")}}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
func handleSetSceneMessage(c *gin.Context) {
	var msg pkgmodels.Message
	if err := db.GetCollection(pkgmodels.MessageCollection).Find(bson.M{"public_id": c.Param("messageId")}).One(&msg); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "message not found"}); return
	}
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"message": msg}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
func handleRemoveSceneMessage(c *gin.Context) {
	db.GetCollection(pkgmodels.SceneCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$unset": bson.M{"message": ""}})
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
	if err := db.GetCollection(pkgmodels.MessageCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
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
	if content == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message content is required"})
		return
	}
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
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.MessageCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteMessage(c *gin.Context) {
	db.GetCollection(pkgmodels.MessageCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAddTagToMessage(c *gin.Context) {
	var body struct{ TagId string `json:"tag_id"` }
	c.ShouldBindJSON(&body)
	var tag pkgmodels.Tag
	if err := db.GetCollection(pkgmodels.TagCollection).Find(bson.M{"public_id": body.TagId}).One(&tag); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tag not found"}); return
	}
	db.GetCollection(pkgmodels.MessageCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$push": bson.M{"vars": tag}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
func handleRemoveTagFromMessage(c *gin.Context) {
	db.GetCollection(pkgmodels.MessageCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$pull": bson.M{"vars": bson.M{"public_id": c.Param("tagId")}}})
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
	if err := db.GetCollection(pkgmodels.MessageContentCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
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
	if err := db.GetCollection(pkgmodels.TriggerCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
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
	if err := db.GetCollection(pkgmodels.ActionCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
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
	if err := db.GetCollection(pkgmodels.BadgeCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
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
	var updates bson.M
	c.ShouldBindJSON(&updates)
	db.GetCollection(pkgmodels.BadgeCollection).Update(bson.M{"public_id": c.Param("id")}, bson.M{"$set": updates})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleDeleteBadge(c *gin.Context) {
	db.GetCollection(pkgmodels.BadgeCollection).Remove(bson.M{"public_id": c.Param("id")})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleAssignBadgeToUser(c *gin.Context)  { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }
func handleRemoveBadgeFromUser(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

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
	if err := db.GetCollection(pkgmodels.TagCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item); err != nil {
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
	item["subscriber_id"] = auth.GetTenantID(c)
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
	sid := auth.GetTenantID(c)
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
	item.SubscriberId = auth.GetTenantID(c)
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
	sid := auth.GetTenantID(c)
	db.GetCollection(pkgmodels.UserCollection).UpdateAll(bson.M{"subscriber_id": sid}, bson.M{"$set": bson.M{"timestamps.deleted_at": time.Now()}})
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func handleGetUserDetail(c *gin.Context) {
	var item pkgmodels.User
	db.GetCollection(pkgmodels.UserCollection).Find(bson.M{"public_id": c.Param("id")}).One(&item)
	c.JSON(http.StatusOK, gin.H{"user": item, "campaign": nil, "hot_triggers": []interface{}{}, "pending_emails": []interface{}{}})
}

func handleAddUserToStory(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) }

// HandleRegisterUser registers a new end-user/subscriber (public endpoint, no tenant JWT needed).
func HandleRegisterUser(c *gin.Context) {
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
	sid := auth.GetTenantID(c)
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
