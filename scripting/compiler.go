package scripting

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	pkgmodels "github.com/josephalai/sentanyl/pkg/models"

	"gopkg.in/mgo.v2/bson"
)

// CompileResult holds the output of the compiler.
type CompileResult struct {
	// The top-level Story entities ready for persistence.
	Stories []*pkgmodels.Story

	// The top-level Funnel entities ready for persistence.
	Funnels []*pkgmodels.Funnel

	// The top-level Site entities ready for persistence.
	Sites []*pkgmodels.Site

	// E-Commerce entities
	Products []*pkgmodels.Product
	Offers   []*pkgmodels.Offer

	// Pending asset generation jobs (lead magnets, worksheets, etc.)
	Assets []*pkgmodels.Asset

	// Video Intelligence entities
	MediaEntities   []*pkgmodels.Media
	PlayerPresets   []*pkgmodels.PlayerPreset
	Channels        []*pkgmodels.MediaChannel
	MediaWebhooks   []*pkgmodels.MediaWebhook

	// LMS entities
	Quizzes            []*pkgmodels.LMSQuiz
	CertificateTemplates []*pkgmodels.CertificateTemplate

	// All entities generated, keyed by name for reference.
	Badges         map[string]*pkgmodels.Badge
	Tags           map[string]*pkgmodels.Tag

	// Diagnostics from the compilation pass.
	Diagnostics Diagnostics
}

// Compiler maps a validated AST into the existing Sentanyl entity model.
type Compiler struct {
	ast          *ScriptAST
	symbols      *SymbolTable
	subscriberID string
	creatorID    bson.ObjectId
	errors       Diagnostics

	// Generated entity maps
	badges       map[string]*pkgmodels.Badge
	tags         map[string]*pkgmodels.Tag

	// Named reference maps for wiring
	storyMap     map[string]*pkgmodels.Story        // "story_name" -> entity (for next_story resolution)
	storylineMap map[string]*pkgmodels.Storyline   // "story:storyline" -> entity
	enactmentMap map[string]*pkgmodels.Enactment   // "story:storyline:enactment" -> entity
	sceneMap     map[string]*pkgmodels.Scene        // "story:storyline:enactment:scene" -> entity
	funnelMap    map[string]*pkgmodels.Funnel       // "funnel_name" -> entity
	stageMap     map[string]*pkgmodels.FunnelStage  // "funnel:route:stage" -> entity
	productMap   map[string]*pkgmodels.Product      // "product_name" -> entity
	offerMap     map[string]*pkgmodels.Offer        // "offer_name" -> entity
	mediaMap     map[string]*pkgmodels.Media        // "media_name" -> entity
	presetMap    map[string]*pkgmodels.PlayerPreset  // "preset_name" -> entity

	// Pending asset generation jobs accumulated during compilation
	pendingAssets []*pkgmodels.Asset

	// Pending quiz entities accumulated during compilation
	pendingQuizzes []*pkgmodels.LMSQuiz

	// Pending certificate templates accumulated during compilation
	certTemplates []*pkgmodels.CertificateTemplate

	// Retry metadata embedded in triggers
	retryCounters map[string]int // keyed by enactment path
}

// NewCompiler creates a Compiler.
func NewCompiler(ast *ScriptAST, symbols *SymbolTable, subscriberID string, creatorID bson.ObjectId) *Compiler {
	return &Compiler{
		ast:          ast,
		symbols:      symbols,
		subscriberID: subscriberID,
		creatorID:    creatorID,
		badges:       make(map[string]*pkgmodels.Badge),
		tags:         make(map[string]*pkgmodels.Tag),
		storyMap:     make(map[string]*pkgmodels.Story),
		storylineMap: make(map[string]*pkgmodels.Storyline),
		enactmentMap: make(map[string]*pkgmodels.Enactment),
		sceneMap:     make(map[string]*pkgmodels.Scene),
		funnelMap:    make(map[string]*pkgmodels.Funnel),
		stageMap:     make(map[string]*pkgmodels.FunnelStage),
		productMap:   make(map[string]*pkgmodels.Product),
		offerMap:     make(map[string]*pkgmodels.Offer),
		mediaMap:     make(map[string]*pkgmodels.Media),
		presetMap:    make(map[string]*pkgmodels.PlayerPreset),
		retryCounters: make(map[string]int),
	}
}

// Compile maps the AST into Sentanyl entities.
func (c *Compiler) Compile() *CompileResult {
	result := &CompileResult{
		Badges: make(map[string]*pkgmodels.Badge),
		Tags:   make(map[string]*pkgmodels.Tag),
	}

	// Pre-create all badges and tags
	c.precreateBadges()
	c.precreateTags()

	// Compile each story
	for _, storyNode := range c.ast.Stories {
		story := c.compileStory(storyNode)
		if story != nil {
			result.Stories = append(result.Stories, story)
			c.storyMap[storyNode.Name] = story
		}
	}

	// Second pass: resolve cross-story references (next_story in on_complete / on_fail)
	for i, storyNode := range c.ast.Stories {
		if i >= len(result.Stories) {
			break
		}
		story := result.Stories[i]

		if storyNode.OnComplete != nil && storyNode.OnComplete.NextStory != "" {
			if nextStory, ok := c.storyMap[storyNode.OnComplete.NextStory]; ok {
				story.OnComplete.NextStory = nextStory
				story.OnComplete.NextStoryId = &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.StoryCollection,
					Id:             nextStory.Id,
				}
			} else {
				c.errorf(storyNode.OnComplete.Pos, "next_story %q not found", storyNode.OnComplete.NextStory)
			}
		}

		if storyNode.OnFail != nil && storyNode.OnFail.NextStory != "" {
			if nextStory, ok := c.storyMap[storyNode.OnFail.NextStory]; ok {
				story.OnFail.NextStory = nextStory
				story.OnFail.NextStoryId = &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.StoryCollection,
					Id:             nextStory.Id,
				}
			} else {
				c.errorf(storyNode.OnFail.Pos, "next_story %q not found", storyNode.OnFail.NextStory)
			}
		}
	}

	// Compile each funnel
	for _, funnelNode := range c.ast.Funnels {
		funnel := c.compileFunnel(funnelNode)
		if funnel != nil {
			result.Funnels = append(result.Funnels, funnel)
		}
	}

	// Compile each site (global website)
	for _, siteNode := range c.ast.Sites {
		site := c.compileSite(siteNode)
		if site != nil {
			result.Sites = append(result.Sites, site)
		}
	}

	// Compile each product (no price - deliverable only)
	for _, productNode := range c.ast.Products {
		product := c.compileProduct(productNode)
		if product != nil {
			result.Products = append(result.Products, product)
			c.productMap[productNode.Name] = product
		}
	}

	// Compile each course (sugar for product type=course)
	for _, courseNode := range c.ast.Courses {
		product := c.compileCourseDecl(courseNode)
		if product != nil {
			result.Products = append(result.Products, product)
			c.productMap[courseNode.Name] = product
		}
	}

	// Compile each offer (pricing + badge grants)
	for _, offerNode := range c.ast.Offers {
		offer := c.compileOffer(offerNode)
		if offer != nil {
			result.Offers = append(result.Offers, offer)
			c.offerMap[offerNode.Name] = offer
		}
	}

	// Compile video intelligence entities
	for _, mediaNode := range c.ast.MediaDecls {
		media := c.compileMediaDecl(mediaNode)
		if media != nil {
			result.MediaEntities = append(result.MediaEntities, media)
			c.mediaMap[mediaNode.Name] = media
		}
	}

	for _, presetNode := range c.ast.PlayerPresets {
		preset := c.compilePlayerPresetDecl(presetNode)
		if preset != nil {
			result.PlayerPresets = append(result.PlayerPresets, preset)
			c.presetMap[presetNode.Name] = preset
		}
	}

	for _, channelNode := range c.ast.ChannelDecls {
		channel := c.compileChannelDecl(channelNode)
		if channel != nil {
			result.Channels = append(result.Channels, channel)
		}
	}

	for _, webhookNode := range c.ast.MediaWebhookDecls {
		webhook := c.compileMediaWebhookDecl(webhookNode)
		if webhook != nil {
			result.MediaWebhooks = append(result.MediaWebhooks, webhook)
		}
	}

	result.Badges = c.badges
	result.Tags = c.tags
	result.Assets = c.pendingAssets
	result.Quizzes = c.pendingQuizzes
	result.CertificateTemplates = c.certTemplates
	result.Diagnostics = c.errors
	return result
}

func (c *Compiler) errorf(pos Pos, format string, args ...interface{}) {
	c.errors = append(c.errors, Diagnostic{
		Pos:     pos,
		Message: fmt.Sprintf(format, args...),
		Level:   DiagError,
	})
}

// ---------- Badge / Tag Pre-creation ----------

func (c *Compiler) precreateBadges() {
	for name := range c.symbols.BadgeNames {
		badge := &pkgmodels.Badge{
			Id:           bson.NewObjectId(),
			PublicId:     generatePublicID("badge"),
			SubscriberId: c.subscriberID,
			CreatorId:    c.creatorID,
			Name:         name,
			Description:  fmt.Sprintf("Auto-generated badge: %s", name),
		}
		c.badges[name] = badge
	}
}

func (c *Compiler) precreateTags() {
	for name := range c.symbols.TagNames {
		tag := &pkgmodels.Tag{
			Id:           bson.NewObjectId(),
			PublicId:     generatePublicID("tag"),
			SubscriberId: c.subscriberID,
			CreatorId:    c.creatorID,
			Name:         name,
		}
		c.tags[name] = tag
	}
}

// ---------- Story Compilation ----------

func (c *Compiler) compileStory(node *StoryNode) *pkgmodels.Story {
	story := &pkgmodels.Story{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("story"),
		SubscriberId: c.subscriberID,
		CreatorId:    c.creatorID,
		Name:         node.Name,
	}

	if node.Priority != nil {
		story.Priority = *node.Priority
	}
	if node.AllowInterruption != nil {
		story.AllowInterruption = *node.AllowInterruption
	}

	// OnBegin
	if node.OnBegin != nil {
		story.OnBegin.BadgeTransaction = c.compileBadgeTransaction(node.OnBegin.BadgeTransaction)
	}

	// OnComplete
	if node.OnComplete != nil {
		story.OnComplete.BadgeTransaction = c.compileBadgeTransaction(node.OnComplete.BadgeTransaction)
		// NextStory reference resolved in Compile() second pass
	}

	// OnFail
	if node.OnFail != nil {
		story.OnFail.BadgeTransaction = c.compileBadgeTransaction(node.OnFail.BadgeTransaction)
	}

	// Required badges
	if node.RequiredBadges != nil {
		story.RequiredUserBadges.MustHave = c.compileRequiredBadges(node.RequiredBadges.MustHave)
		story.RequiredUserBadges.MustNotHave = c.compileRequiredBadges(node.RequiredBadges.MustNotHave)
	}

	// Start trigger
	if node.StartTrigger != nil {
		badge := c.badges[*node.StartTrigger]
		if badge != nil {
			story.StartTrigger = &pkgmodels.RequiredBadge{
				Id:   bson.NewObjectId(),
				Name: badge.Name,
				BadgeID: &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.BadgeCollection,
					Id:             badge.Id,
				},
				Badge: badge,
			}
		}
	}

	// Complete trigger
	if node.CompleteTrigger != nil {
		badge := c.badges[*node.CompleteTrigger]
		if badge != nil {
			story.CompleteTrigger = &pkgmodels.RequiredBadge{
				Id:   bson.NewObjectId(),
				Name: badge.Name,
				BadgeID: &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.BadgeCollection,
					Id:             badge.Id,
				},
				Badge: badge,
			}
		}
	}

	// Compile storylines
	storylineIds := &pkgmodels.BsonCollectionIds{
		CollectionName: pkgmodels.StorylineCollection,
	}
	for i, slNode := range node.Storylines {
		sl := c.compileStoryline(node, slNode, i+1)
		if sl != nil {
			story.Storylines = append(story.Storylines, sl)
			storylineIds.Ids = append(storylineIds.Ids, sl.Id)
			c.storylineMap[node.Name+":"+slNode.Name] = sl
		}
	}
	story.StorylineIds = storylineIds

	// Second pass: resolve storyline references in on_complete/on_fail
	for _, slNode := range node.Storylines {
		slKey := node.Name + ":" + slNode.Name
		sl := c.storylineMap[slKey]
		if sl == nil {
			continue
		}

		if slNode.OnComplete != nil && slNode.OnComplete.NextStoryline != "" {
			nextKey := node.Name + ":" + slNode.OnComplete.NextStoryline
			if nextSL, ok := c.storylineMap[nextKey]; ok {
				sl.OnComplete.NextStoryline = nextSL
				sl.OnComplete.NextStorylineId = &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.StorylineCollection,
					Id:             nextSL.Id,
				}
			}
		}

		if slNode.OnComplete != nil {
			for i, crNode := range slNode.OnComplete.ConditionalRoutes {
				if i < len(sl.OnComplete.ConditionalRoutes) {
					cr := sl.OnComplete.ConditionalRoutes[i]
					nextKey := node.Name + ":" + crNode.NextStoryline
					if nextSL, ok := c.storylineMap[nextKey]; ok {
						cr.NextStoryline = nextSL
						cr.NextStorylineId = &pkgmodels.BsonCollectionId{
							CollectionName: pkgmodels.StorylineCollection,
							Id:             nextSL.Id,
						}
					}
				}
			}
		}

		if slNode.OnFail != nil && slNode.OnFail.NextStoryline != "" {
			nextKey := node.Name + ":" + slNode.OnFail.NextStoryline
			if nextSL, ok := c.storylineMap[nextKey]; ok {
				sl.OnFail.NextStoryline = nextSL
				sl.OnFail.NextStorylineId = &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.StorylineCollection,
					Id:             nextSL.Id,
				}
			}
		}

		if slNode.OnFail != nil {
			for i, crNode := range slNode.OnFail.ConditionalRoutes {
				if i < len(sl.OnFail.ConditionalRoutes) {
					cr := sl.OnFail.ConditionalRoutes[i]
					nextKey := node.Name + ":" + crNode.NextStoryline
					if nextSL, ok := c.storylineMap[nextKey]; ok {
						cr.NextStoryline = nextSL
						cr.NextStorylineId = &pkgmodels.BsonCollectionId{
							CollectionName: pkgmodels.StorylineCollection,
							Id:             nextSL.Id,
						}
					}
				}
			}
		}
	}

	return story
}

// ---------- Storyline Compilation ----------

func (c *Compiler) compileStoryline(storyNode *StoryNode, node *StorylineNode, defaultOrder int) *pkgmodels.Storyline {
	sl := &pkgmodels.Storyline{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("storyline"),
		SubscriberId: c.subscriberID,
		CreatorId:    c.creatorID,
		Name:         node.Name,
	}

	if node.Order != nil {
		sl.NaturalOrder = *node.Order
	} else {
		sl.NaturalOrder = defaultOrder
	}

	// Required badges
	if node.RequiredBadges != nil {
		sl.RequiredUserBadges.MustHave = c.compileRequiredBadges(node.RequiredBadges.MustHave)
		sl.RequiredUserBadges.MustNotHave = c.compileRequiredBadges(node.RequiredBadges.MustNotHave)
	}

	// OnBegin
	if node.OnBegin != nil {
		sl.OnBegin.BadgeTransaction = c.compileBadgeTransaction(node.OnBegin.BadgeTransaction)
	}

	// OnComplete
	if node.OnComplete != nil {
		sl.OnComplete.BadgeTransaction = c.compileBadgeTransaction(node.OnComplete.BadgeTransaction)
		// Conditional routes
		for _, crNode := range node.OnComplete.ConditionalRoutes {
			cr := c.compileConditionalRoute(storyNode, crNode)
			sl.OnComplete.ConditionalRoutes = append(sl.OnComplete.ConditionalRoutes, cr)
		}
	}

	// OnFail
	if node.OnFail != nil {
		sl.OnFail.BadgeTransaction = c.compileBadgeTransaction(node.OnFail.BadgeTransaction)
		for _, crNode := range node.OnFail.ConditionalRoutes {
			cr := c.compileConditionalRoute(storyNode, crNode)
			sl.OnFail.ConditionalRoutes = append(sl.OnFail.ConditionalRoutes, cr)
		}
	}

	// Compile enactments
	actIds := &pkgmodels.BsonCollectionIds{
		CollectionName: pkgmodels.EnactmentCollection,
	}
	for i, enNode := range node.Enactments {
		en := c.compileEnactment(storyNode, node, enNode, i+1)
		if en != nil {
			sl.Acts = append(sl.Acts, en)
			actIds.Ids = append(actIds.Ids, en.Id)
			enKey := storyNode.Name + ":" + node.Name + ":" + enNode.Name
			c.enactmentMap[enKey] = en
		}
	}
	sl.ActIds = actIds

	// Linking pass: resolve forward enactment references in trigger actions.
	// At this point all enactments for this storyline are in c.enactmentMap.
	for _, enNode := range node.Enactments {
		enKey := storyNode.Name + ":" + node.Name + ":" + enNode.Name
		en := c.enactmentMap[enKey]
		if en == nil {
			continue
		}
		for _, triggers := range en.OnEvent {
			for _, tr := range triggers {
				if tr != nil && tr.DoAction != nil {
					c.resolveEnactmentRef(storyNode.Name, node.Name, tr.DoAction)
				}
			}
		}
	}

	return sl
}

// resolveEnactmentRef resolves a deferred jump_to_enactment reference on an
// Action. During initial compilation forward references store the target name
// in ActionName as "jump_to_enactment:<name>". This pass replaces that with
// the actual NextEnactment pointer now that all enactments are compiled.
func (c *Compiler) resolveEnactmentRef(storyName, slName string, action *pkgmodels.Action) {
	const prefix = "jump_to_enactment:"
	if !strings.HasPrefix(action.ActionName, prefix) {
		return
	}
	targetName := strings.TrimPrefix(action.ActionName, prefix)
	enKey := storyName + ":" + slName + ":" + targetName
	if targetEn, ok := c.enactmentMap[enKey]; ok {
		// Only set the ID reference, NOT the full NextEnactment pointer.
		// Action.ReadyMongoStore() recursively persists NextEnactment, so
		// setting the pointer would cause duplicate inserts when multiple
		// triggers reference the same target enactment.  The runtime
		// (ExecuteAction) already hydrates NextEnactment from NextEnactmentId.
		action.NextEnactmentId = &pkgmodels.BsonCollectionId{
			CollectionName: pkgmodels.EnactmentCollection,
			Id:             targetEn.Id,
		}
		action.ActionName = ""
	}
}

// ---------- Enactment Compilation ----------

func (c *Compiler) compileEnactment(storyNode *StoryNode, slNode *StorylineNode, node *EnactmentNode, defaultOrder int) *pkgmodels.Enactment {
	en := &pkgmodels.Enactment{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("enactment"),
		SubscriberId: c.subscriberID,
		CreatorId:    c.creatorID,
		Name:         node.Name,
	}

	if node.Level != nil {
		en.Level = *node.Level
	} else {
		en.Level = defaultOrder
	}
	if node.Order != nil {
		en.NaturalOrder = *node.Order
	} else {
		en.NaturalOrder = defaultOrder
	}
	if node.SkipToNextStorylineOnExpiry != nil {
		en.SkipToNextStorylineOnExpiry = *node.SkipToNextStorylineOnExpiry
	}

	// Compile scenes
	if len(node.Scenes) == 1 {
		// Single-scene enactment → uses SendScene
		scene := c.compileScene(storyNode, slNode, node, node.Scenes[0])
		en.SendScene = scene
		en.SendSceneId = &pkgmodels.BsonCollectionId{
			CollectionName: pkgmodels.SceneCollection,
			Id:             scene.Id,
		}
		scKey := storyNode.Name + ":" + slNode.Name + ":" + node.Name + ":" + node.Scenes[0].Name
		c.sceneMap[scKey] = scene
	} else if len(node.Scenes) > 1 {
		// Multi-scene enactment → uses SendScenes
		sceneIds := &pkgmodels.BsonCollectionIds{
			CollectionName: pkgmodels.SceneCollection,
		}
		for _, scNode := range node.Scenes {
			scene := c.compileScene(storyNode, slNode, node, scNode)
			en.SendScenes = append(en.SendScenes, scene)
			sceneIds.Ids = append(sceneIds.Ids, scene.Id)
			scKey := storyNode.Name + ":" + slNode.Name + ":" + node.Name + ":" + scNode.Name
			c.sceneMap[scKey] = scene
		}
		en.SendScenesIds = sceneIds
		// Also set first scene as SendScene for backward compat
		en.SendScene = en.SendScenes[0]
		en.SendSceneId = &pkgmodels.BsonCollectionId{
			CollectionName: pkgmodels.SceneCollection,
			Id:             en.SendScenes[0].Id,
		}
	}

	// Compile triggers
	en.OnEvent = make(map[string][]*pkgmodels.Trigger)
	triggerIds := &pkgmodels.BsonCollectionIds{
		CollectionName: pkgmodels.TriggerCollection,
	}
	for _, trNode := range node.Triggers {
		triggers := c.compileTrigger(storyNode, slNode, node, trNode)
		for _, tr := range triggers {
			triggerType := tr.TriggerType
			en.OnEvent[triggerType] = append(en.OnEvent[triggerType], tr)
			triggerIds.Ids = append(triggerIds.Ids, tr.Id)
		}
	}
	en.OnEventIds = triggerIds

	return en
}

// ---------- Scene Compilation ----------

func (c *Compiler) compileScene(storyNode *StoryNode, slNode *StorylineNode, enNode *EnactmentNode, node *SceneNode) *pkgmodels.Scene {
	scene := &pkgmodels.Scene{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("scene"),
		SubscriberId: c.subscriberID,
		CreatorId:    c.creatorID,
		Name:         node.Name,
	}

	// Message
	msg := &pkgmodels.Message{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("message"),
		SubscriberId: c.subscriberID,
		CreatorId:    c.creatorID,
	}

	content := &pkgmodels.MessageContent{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("content"),
		SubscriberId: c.subscriberID,
		CreatorId:    c.creatorID,
		Subject:      node.Subject,
		Body:         node.Body,
		FromEmail:    node.FromEmail,
		FromName:     node.FromName,
		ReplyTo:      node.ReplyTo,
	}

	// Vars
	if len(node.Vars) > 0 {
		content.GivenVars = node.Vars
	}

	// TemplateName — create a Template entity with the given name so that the
	// platform can look it up during email compilation.
	if node.TemplateName != "" {
		tmpl := &pkgmodels.Template{
			Id:           bson.NewObjectId(),
			PublicId:     generatePublicID("template"),
			SubscriberId: c.subscriberID,
			CreatorId:    c.creatorID,
			Name:         node.TemplateName,
		}
		content.Template = tmpl
		content.TemplateId = &pkgmodels.BsonCollectionId{
			CollectionName: pkgmodels.TemplateCollection,
			Id:             tmpl.Id,
		}
	}

	msg.Content = content
	scene.Message = msg
	scene.MessageId = &pkgmodels.BsonCollectionId{
		CollectionName: pkgmodels.MessageCollection,
		Id:             msg.Id,
	}

	// Tags
	if len(node.Tags) > 0 {
		tagIds := &pkgmodels.BsonCollectionIds{
			CollectionName: pkgmodels.TagCollection,
		}
		for _, tagName := range node.Tags {
			if tag, ok := c.tags[tagName]; ok {
				scene.Tags = append(scene.Tags, tag)
				tagIds.Ids = append(tagIds.Ids, tag.Id)
			}
		}
		scene.TagsIds = tagIds
	}

	return scene
}

// ---------- Trigger Compilation ----------

func (c *Compiler) compileTrigger(storyNode *StoryNode, slNode *StorylineNode, enNode *EnactmentNode, node *TriggerNode) []*pkgmodels.Trigger {
	triggers := []*pkgmodels.Trigger{}

	// Map trigger type to existing constants
	triggerType, userAction, userActionValue := c.mapTriggerType(node)

	// Main trigger with primary actions
	mainTrigger := &pkgmodels.Trigger{
		Id:              bson.NewObjectId(),
		PublicId:        generatePublicID("trigger"),
		SubscriberId:    c.subscriberID,
		CreatorId:       c.creatorID,
		TriggerType:     triggerType,
		UserAction:      userAction,
		UserActionValue: userActionValue,
		MarkComplete:    node.MarkComplete,
		MarkFailed:      node.MarkFailed,
	}

	// Priority
	if node.Priority != nil {
		mainTrigger.Priority = pkgmodels.TriggerPriority(*node.Priority)
	}

	// Persist scope
	if node.PersistScope != "" {
		mainTrigger.PersistScope = pkgmodels.TriggerScope(node.PersistScope)
	}

	// Required badges on trigger
	if node.RequiredBadges != nil {
		mainTrigger.RequiredBadges.MustHave = c.compileRequiredBadges(node.RequiredBadges.MustHave)
		mainTrigger.RequiredBadges.MustNotHave = c.compileRequiredBadges(node.RequiredBadges.MustNotHave)
	}

	// Compile condition guards into required badges
	for _, cond := range node.Conditions {
		c.applyConditionToTrigger(cond, mainTrigger)
	}

	// Watch trigger fields
	if node.TriggerType == "watch" || node.TriggerType == "progress" {
		mainTrigger.WatchBlockID = node.WatchBlockID
		mainTrigger.WatchOperator = node.WatchOperator
		mainTrigger.WatchPercent = node.WatchPercent
	}

	// Compile primary action
	if len(node.Actions) > 0 {
		action := c.compileActions(storyNode, slNode, enNode, node.Actions)
		mainTrigger.DoAction = action
		mainTrigger.DoActionId = &pkgmodels.BsonCollectionId{
			CollectionName: pkgmodels.ActionCollection,
			Id:             action.Id,
		}

		// Apply within timing to action
		if node.Within != nil && action.When == nil {
			action.When = c.compileActionWhen(node.Within)
		}

		// Propagate action-level flags to the trigger so the runtime sees them.
		// The DSL allows "do mark_complete" / "do mark_failed" as actions, but
		// ExecuteAction checks the Trigger's MarkComplete/MarkFailed booleans.
		for _, a := range node.Actions {
			if a.MarkComplete {
				mainTrigger.MarkComplete = true
			}
			if a.MarkFailed {
				mainTrigger.MarkFailed = true
			}
		}
	}

	triggers = append(triggers, mainTrigger)

	// Compile else actions as a separate trigger with lower priority.
	// ElseActions come from trigger-level "else" blocks; RetryFallback comes
	// from action-level "else" inside retry_scene/retry_enactment modifiers.
	var elseActions []*ActionNode
	if len(node.ElseActions) > 0 {
		elseActions = node.ElseActions
	} else {
		// Check if any primary action has RetryFallback (e.g. retry_scene ... else do ...)
		for _, a := range node.Actions {
			if len(a.RetryFallback) > 0 {
				elseActions = a.RetryFallback
				break
			}
		}
	}

	if len(elseActions) > 0 {
		elseTrigger := &pkgmodels.Trigger{
			Id:              bson.NewObjectId(),
			PublicId:        generatePublicID("trigger"),
			SubscriberId:    c.subscriberID,
			CreatorId:       c.creatorID,
			TriggerType:     pkgmodels.Else,
			UserAction:      userAction,
			UserActionValue: userActionValue,
		}

		if node.Priority != nil {
			// Else gets lower priority
			elseTrigger.Priority = pkgmodels.TriggerPriority(*node.Priority + 1)
		} else {
			elseTrigger.Priority = pkgmodels.SecondPriority
		}

		action := c.compileActions(storyNode, slNode, enNode, elseActions)
		elseTrigger.DoAction = action
		elseTrigger.DoActionId = &pkgmodels.BsonCollectionId{
			CollectionName: pkgmodels.ActionCollection,
			Id:             action.Id,
		}

		// Propagate action-level flags to the else trigger.
		for _, a := range elseActions {
			if a.MarkComplete {
				elseTrigger.MarkComplete = true
			}
			if a.MarkFailed {
				elseTrigger.MarkFailed = true
			}
		}

		triggers = append(triggers, elseTrigger)
	}

	return triggers
}

func (c *Compiler) mapTriggerType(node *TriggerNode) (string, string, string) {
	switch node.TriggerType {
	case "click":
		return pkgmodels.OnClick, pkgmodels.OnClick, node.UserActionValue
	case "not_click":
		return pkgmodels.OnNotClick, pkgmodels.OnNotClick, node.UserActionValue
	case "open":
		return pkgmodels.OnOpen, pkgmodels.OnOpen, ""
	case "not_open":
		return pkgmodels.OnNotOpen, pkgmodels.OnNotOpen, ""
	case "sent":
		return pkgmodels.OnSent, "", ""
	case "webhook":
		return pkgmodels.OnWebhook, "", node.UserActionValue
	case "nothing":
		return pkgmodels.OnNothing, "", ""
	case "else":
		return pkgmodels.Else, "", ""
	case "bounce":
		return pkgmodels.OnBounce, "", ""
	case "spam":
		return pkgmodels.OnSpam, "", ""
	case "unsubscribe":
		return pkgmodels.OnUnsubscribe, "", ""
	case "failure":
		return pkgmodels.OnFailure, "", ""
	case "email_validated":
		return pkgmodels.OnEmailAddressValidated, "", ""
	case "user_has_tag":
		return pkgmodels.UserHasTag, "", node.UserActionValue
	case "badge":
		return pkgmodels.OnBadge, "", node.UserActionValue
	case "submit":
		return pkgmodels.OnSubmit, pkgmodels.OnSubmit, node.UserActionValue
	case "abandon":
		return pkgmodels.OnAbandon, pkgmodels.OnAbandon, ""
	case "purchase":
		return pkgmodels.OnPurchase, pkgmodels.OnPurchase, node.UserActionValue
	case "watch":
		return pkgmodels.OnWatch, pkgmodels.OnWatch, node.UserActionValue
	case "play":
		return pkgmodels.OnPlay, pkgmodels.OnPlay, node.UserActionValue
	case "pause":
		return pkgmodels.OnPause, pkgmodels.OnPause, node.UserActionValue
	case "progress":
		return pkgmodels.OnProgress, pkgmodels.OnProgress, node.UserActionValue
	case "complete":
		return pkgmodels.OnComplete, pkgmodels.OnComplete, node.UserActionValue
	case "rewatch":
		return pkgmodels.OnRewatch, pkgmodels.OnRewatch, node.UserActionValue
	case "cta_click":
		return pkgmodels.OnCTAClick, pkgmodels.OnCTAClick, node.UserActionValue
	case "turnstile_submit":
		return pkgmodels.OnTurnstileSubmit, pkgmodels.OnTurnstileSubmit, node.UserActionValue
	case "chapter_click":
		return pkgmodels.OnChapterClick, pkgmodels.OnChapterClick, node.UserActionValue
	default:
		return node.TriggerType, "", node.UserActionValue
	}
}

func (c *Compiler) applyConditionToTrigger(cond *ConditionNode, trigger *pkgmodels.Trigger) {
	if cond == nil {
		return
	}
	switch cond.ConditionType {
	case "has_badge":
		if badge, ok := c.badges[cond.Value]; ok {
			rb := &pkgmodels.RequiredBadge{
				Id:   bson.NewObjectId(),
				Name: badge.Name,
				BadgeID: &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.BadgeCollection,
					Id:             badge.Id,
				},
				Badge: badge,
			}
			trigger.RequiredBadges.MustHave = append(trigger.RequiredBadges.MustHave, rb)
		}
	case "not_has_badge":
		if badge, ok := c.badges[cond.Value]; ok {
			rb := &pkgmodels.RequiredBadge{
				Id:   bson.NewObjectId(),
				Name: badge.Name,
				BadgeID: &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.BadgeCollection,
					Id:             badge.Id,
				},
				Badge: badge,
			}
			trigger.RequiredBadges.MustNotHave = append(trigger.RequiredBadges.MustNotHave, rb)
		}
	case "and":
		for _, child := range cond.Children {
			c.applyConditionToTrigger(child, trigger)
		}
	case "or":
		// OR conditions map to BadgeConditions
		for _, child := range cond.Children {
			c.applyConditionToTrigger(child, trigger)
		}
	case "not":
		if len(cond.Children) > 0 {
			// Flip the condition
			child := cond.Children[0]
			flipped := &ConditionNode{
				NodeBase: child.NodeBase,
				Value:    child.Value,
				Children: child.Children,
			}
			switch child.ConditionType {
			case "has_badge":
				flipped.ConditionType = "not_has_badge"
			case "not_has_badge":
				flipped.ConditionType = "has_badge"
			default:
				flipped.ConditionType = child.ConditionType
			}
			c.applyConditionToTrigger(flipped, trigger)
		}
	}
}

// ---------- Action Compilation ----------

// addActionName sets action.ActionName if empty, otherwise appends to ExtraActions.
func addActionName(action *pkgmodels.Action, name string) {
	if action.ActionName == "" {
		action.ActionName = name
	} else {
		action.ExtraActions = append(action.ExtraActions, name)
	}
}

func (c *Compiler) compileActions(storyNode *StoryNode, slNode *StorylineNode, enNode *EnactmentNode, actions []*ActionNode) *pkgmodels.Action {
	if len(actions) == 0 {
		return nil
	}

	// Build a single Action entity that captures the combined behavior.
	// The existing model uses a single Action per trigger, so we merge.
	action := &pkgmodels.Action{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("action"),
		SubscriberId: c.subscriberID,
		CreatorId:    c.creatorID,
	}

	for _, an := range actions {
		switch an.ActionType {
		case "next_scene":
			// Next scene is implicit in the enactment flow; just set name.
			action.ActionName = "next_scene"
		case "prev_scene":
			action.ActionName = "prev_scene"
		case "jump_to_enactment", "next_enactment":
			enKey := storyNode.Name + ":" + slNode.Name + ":" + an.Target
			if targetEn, ok := c.enactmentMap[enKey]; ok {
				action.NextEnactment = targetEn
				action.NextEnactmentId = &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.EnactmentCollection,
					Id:             targetEn.Id,
				}
			} else {
				// Will be resolved in linking pass; store the name for now.
				action.ActionName = "jump_to_enactment:" + an.Target
			}
		case "loop_to_enactment":
			enKey := storyNode.Name + ":" + slNode.Name + ":" + an.Target
			if targetEn, ok := c.enactmentMap[enKey]; ok {
				// Only set the ID reference, NOT the full NextEnactment pointer.
				// Action.ReadyMongoStore() recursively persists NextEnactment, so
				// setting the pointer causes duplicate inserts of the target
				// enactment's scenes/messages/content.
				action.NextEnactmentId = &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.EnactmentCollection,
					Id:             targetEn.Id,
				}
			}
			action.ActionName = "loop_to_enactment"
			if an.RetryMaxCount != nil {
				action.ActionName = fmt.Sprintf("loop_to_enactment:max_%d", *an.RetryMaxCount)
			}
		case "loop_to_start_enactment":
			// Loop back to the current enactment's first scene
			action.ActionName = "loop_to_start_enactment"
		case "retry_scene":
			action.ActionName = "retry_scene"
			if an.RetryMaxCount != nil {
				action.ActionName = fmt.Sprintf("retry_scene:max_%d", *an.RetryMaxCount)
			}
		case "retry_enactment":
			action.ActionName = "retry_enactment"
			if an.RetryMaxCount != nil {
				action.ActionName = fmt.Sprintf("retry_enactment:max_%d", *an.RetryMaxCount)
			}
		case "advance_to_next_storyline":
			action.AdvanceToNextStoryline = true
		case "jump_to_storyline", "loop_to_storyline":
			action.AdvanceToNextStoryline = true
			action.ActionName = "jump_to_storyline:" + an.Target
		case "loop_to_start_storyline":
			action.ActionName = "loop_to_start_storyline"
		case "end_story":
			action.EndStory = true
		case "mark_complete":
			// Handled at trigger level
			action.ActionName = "mark_complete"
		case "mark_failed":
			action.ActionName = "mark_failed"
		case "unsubscribe":
			action.Unsubscribe = true
		case "give_badge", "remove_badge":
			if an.BadgeTransaction != nil {
				bt := c.compileBadgeTransaction(an.BadgeTransaction)
				action.BadgeTransaction = bt
				action.BadgeTransactionIds = &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.BadgeTransactionCollection,
					Id:             bt.Id,
				}
			}
		case "wait":
			if an.Wait != nil {
				action.When = c.compileActionWhen(an.Wait)
			}
		case "send_immediate":
			action.SendImmediate = an.SendImmediate
		case "jump_to_stage":
			addActionName(action, pkgmodels.ActionJumpToStage+":"+an.Target)
		case "start_story":
			addActionName(action, pkgmodels.ActionStartStory+":"+an.Target)
		case "send_email":
			addActionName(action, pkgmodels.ActionSendEmail+":"+an.Target)
		case "redirect":
			addActionName(action, pkgmodels.ActionRedirect+":"+an.Target)
		case "provide_download":
			addActionName(action, pkgmodels.ActionProvideDownload+":"+an.Target)
		}

		// Apply send_immediate if set
		if an.SendImmediate != nil {
			action.SendImmediate = an.SendImmediate
		}

		// Apply wait
		if an.Wait != nil && action.When == nil {
			action.When = c.compileActionWhen(an.Wait)
		}

		// Handle badge transactions from any action
		if an.BadgeTransaction != nil && action.BadgeTransaction == nil {
			bt := c.compileBadgeTransaction(an.BadgeTransaction)
			action.BadgeTransaction = bt
			action.BadgeTransactionIds = &pkgmodels.BsonCollectionId{
				CollectionName: pkgmodels.BadgeTransactionCollection,
				Id:             bt.Id,
			}
		}
	}

	return action
}

func (c *Compiler) compileActionWhen(dur *DurationNode) *pkgmodels.ActionWhen {
	if dur == nil {
		return nil
	}
	return &pkgmodels.ActionWhen{
		WaitType: pkgmodels.ActionTaken,
		WaitUntil: &pkgmodels.Timeframe{
			Amount:   dur.Amount,
			TimeUnit: c.mapDurationUnit(dur.Unit),
		},
	}
}

func (c *Compiler) mapDurationUnit(unit string) string {
	switch unit {
	case "d", "days":
		return pkgmodels.Days
	case "h", "hours":
		return pkgmodels.Hours
	case "m", "minutes":
		return pkgmodels.Minutes
	case "s", "seconds":
		return pkgmodels.Seconds
	default:
		return pkgmodels.Days
	}
}

// ---------- Badge / Required Badge Compilation ----------

func (c *Compiler) compileBadgeTransaction(node *BadgeTransactionNode) *pkgmodels.BadgeTransaction {
	if node == nil {
		return nil
	}

	bt := &pkgmodels.BadgeTransaction{
		Id: bson.NewObjectId(),
	}

	if len(node.GiveBadges) > 0 {
		giveIds := &pkgmodels.BsonCollectionIds{
			CollectionName: pkgmodels.BadgeCollection,
		}
		for _, name := range node.GiveBadges {
			if badge, ok := c.badges[name]; ok {
				bt.GiveBadges = append(bt.GiveBadges, badge)
				giveIds.Ids = append(giveIds.Ids, badge.Id)
			}
		}
		bt.GiveBadgesIds = giveIds
	}

	if len(node.RemoveBadges) > 0 {
		removeIds := &pkgmodels.BsonCollectionIds{
			CollectionName: pkgmodels.BadgeCollection,
		}
		for _, name := range node.RemoveBadges {
			if badge, ok := c.badges[name]; ok {
				bt.RemoveBadges = append(bt.RemoveBadges, badge)
				removeIds.Ids = append(removeIds.Ids, badge.Id)
			}
		}
		bt.RemoveBadgesIds = removeIds
	}

	return bt
}

func (c *Compiler) compileRequiredBadges(badgeNames []string) []*pkgmodels.RequiredBadge {
	var result []*pkgmodels.RequiredBadge
	for _, name := range badgeNames {
		if badge, ok := c.badges[name]; ok {
			result = append(result, &pkgmodels.RequiredBadge{
				Id:   bson.NewObjectId(),
				Name: badge.Name,
				BadgeID: &pkgmodels.BsonCollectionId{
					CollectionName: pkgmodels.BadgeCollection,
					Id:             badge.Id,
				},
				Badge: badge,
			})
		}
	}
	return result
}

func (c *Compiler) compileConditionalRoute(storyNode *StoryNode, node *ConditionalRouteNode) *pkgmodels.ConditionalRoute {
	cr := &pkgmodels.ConditionalRoute{}
	if node.RequiredBadges != nil {
		cr.RequiredBadges.MustHave = c.compileRequiredBadges(node.RequiredBadges.MustHave)
		cr.RequiredBadges.MustNotHave = c.compileRequiredBadges(node.RequiredBadges.MustNotHave)
	}
	if node.Priority != nil {
		cr.Priority = *node.Priority
	}
	// NextStoryline wired in linking pass
	return cr
}

// ---------- Funnel Compilation ----------

func (c *Compiler) compileFunnel(node *FunnelNode) *pkgmodels.Funnel {
	funnel := &pkgmodels.Funnel{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("funnel"),
		SubscriberId: c.subscriberID,
		CreatorId:    c.creatorID,
		Name:         node.Name,
		Domain:       node.Domain,
	}

	if node.AIContext != nil {
		funnel.AIContext = c.compileAIContext(node.AIContext)
	}

	routeIds := &pkgmodels.BsonCollectionIds{
		CollectionName: pkgmodels.FunnelRouteCollection,
	}
	for i, rNode := range node.Routes {
		route := c.compileRoute(node, rNode, i+1)
		if route != nil {
			route.FunnelId = funnel.Id
			funnel.Routes = append(funnel.Routes, route)
			routeIds.Ids = append(routeIds.Ids, route.Id)
		}
	}
	funnel.RouteIds = routeIds

	c.funnelMap[node.Name] = funnel
	return funnel
}

func (c *Compiler) compileRoute(funnelNode *FunnelNode, node *RouteNode, defaultOrder int) *pkgmodels.FunnelRoute {
	route := &pkgmodels.FunnelRoute{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("route"),
		SubscriberId: c.subscriberID,
	}
	route.Name = node.Name

	if node.Order != nil {
		route.Order = *node.Order
	} else {
		route.Order = defaultOrder
	}

	if node.RequiredBadges != nil {
		route.RequiredUserBadges.MustHave = c.compileRequiredBadges(node.RequiredBadges.MustHave)
		route.RequiredUserBadges.MustNotHave = c.compileRequiredBadges(node.RequiredBadges.MustNotHave)
	}

	stageIds := &pkgmodels.BsonCollectionIds{
		CollectionName: pkgmodels.FunnelStageCollection,
	}
	for i, sNode := range node.Stages {
		stage := c.compileStage(funnelNode, node, sNode, i+1)
		if stage != nil {
			stage.RouteId = route.Id
			route.Stages = append(route.Stages, stage)
			stageIds.Ids = append(stageIds.Ids, stage.Id)

			sKey := funnelNode.Name + ":" + node.Name + ":" + sNode.Name
			c.stageMap[sKey] = stage
		}
	}
	route.StageIds = stageIds

	return route
}

func (c *Compiler) compileStage(funnelNode *FunnelNode, routeNode *RouteNode, node *StageNode, defaultOrder int) *pkgmodels.FunnelStage {
	stage := &pkgmodels.FunnelStage{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("stage"),
		SubscriberId: c.subscriberID,
		Name:         node.Name,
		Path:         node.Path,
	}

	if node.Order != nil {
		stage.Order = *node.Order
	} else {
		stage.Order = defaultOrder
	}

	// Compile pages
	pageIds := &pkgmodels.BsonCollectionIds{
		CollectionName: pkgmodels.FunnelPageCollection,
	}
	for _, pgNode := range node.Pages {
		page := c.compilePage(pgNode)
		if page != nil {
			page.StageId = stage.Id
			stage.Pages = append(stage.Pages, page)
			pageIds.Ids = append(pageIds.Ids, page.Id)
		}
	}
	stage.PageIds = pageIds

	// Compile triggers
	triggerIds := &pkgmodels.BsonCollectionIds{
		CollectionName: pkgmodels.TriggerCollection,
	}
	for _, trNode := range node.Triggers {
		triggers := c.compileTrigger(nil, nil, nil, trNode)
		for _, tr := range triggers {
			stage.Triggers = append(stage.Triggers, tr)
			triggerIds.Ids = append(triggerIds.Ids, tr.Id)
		}
	}
	stage.TriggerIds = triggerIds

	// PDF config
	if node.PDFConfig != nil {
		stage.PDFConfig = &pkgmodels.PDFConfig{
			AIGenerated: node.PDFConfig.AIGenerated,
		}
		if node.PDFConfig.AIContext != nil {
			stage.PDFConfig.AIContext = c.compileAIContext(node.PDFConfig.AIContext)
		}
	}

	// Lead magnet declarations → pending Asset generation jobs
	for _, lmNode := range node.LeadMagnets {
		assetType := lmNode.AssetType
		if assetType == "" {
			assetType = "guide"
		}
		theme := lmNode.Theme
		if theme == "" {
			theme = "minimal"
		}
		fileName := assetType + "-" + stage.PublicId + ".pdf"
		asset := &pkgmodels.Asset{
			Id:           bson.NewObjectId(),
			PublicId:     generatePublicID("asset"),
			SubscriberId: c.subscriberID,
			FileName:     fileName,
			FileType:     "application/pdf",
			GenConfig: &pkgmodels.AssetGenConfig{
				AssetType:   assetType,
				References:  lmNode.References,
				Instruction: lmNode.Instruction,
				Theme:       theme,
				Status:      "pending",
				StageID:     stage.PublicId,
			},
		}
		c.pendingAssets = append(c.pendingAssets, asset)
	}

	// Lead magnets declared inside page blocks are also processed here (same logic)
	for _, pgNode := range node.Pages {
		for _, lmNode := range pgNode.LeadMagnets {
			assetType := lmNode.AssetType
			if assetType == "" {
				assetType = "guide"
			}
			theme := lmNode.Theme
			if theme == "" {
				theme = "minimal"
			}
			fileName := assetType + "-" + stage.PublicId + ".pdf"
			asset := &pkgmodels.Asset{
				Id:           bson.NewObjectId(),
				PublicId:     generatePublicID("asset"),
				SubscriberId: c.subscriberID,
				FileName:     fileName,
				FileType:     "application/pdf",
				GenConfig: &pkgmodels.AssetGenConfig{
					AssetType:   assetType,
					References:  lmNode.References,
					Instruction: lmNode.Instruction,
					Theme:       theme,
					Status:      "pending",
					StageID:     stage.PublicId,
				},
			}
			c.pendingAssets = append(c.pendingAssets, asset)
		}
	}

	return stage
}

func (c *Compiler) compilePage(node *PageNode) *pkgmodels.FunnelPage {
	page := &pkgmodels.FunnelPage{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("page"),
		SubscriberId: c.subscriberID,
		Name:         node.Name,
		TemplateName: node.TemplateName,
	}

	if node.AIContext != nil {
		page.AIContext = c.compileAIContext(node.AIContext)
	}

	// Compile blocks
	blockIds := &pkgmodels.BsonCollectionIds{
		CollectionName: pkgmodels.PageBlockCollection,
	}
	for _, bNode := range node.Blocks {
		block := c.compileBlock(bNode)
		if block != nil {
			block.PageId = page.Id
			page.Blocks = append(page.Blocks, block)
			blockIds.Ids = append(blockIds.Ids, block.Id)
		}
	}
	page.BlockIds = blockIds

	// Compile forms
	formIds := &pkgmodels.BsonCollectionIds{
		CollectionName: pkgmodels.PageFormCollection,
	}
	for _, fNode := range node.Forms {
		form := c.compileForm(fNode)
		if form != nil {
			form.PageId = page.Id
			page.Forms = append(page.Forms, form)
			formIds.Ids = append(formIds.Ids, form.Id)
		}
	}
	page.FormIds = formIds

	return page
}

func (c *Compiler) compileBlock(node *BlockNode) *pkgmodels.PageBlock {
	block := &pkgmodels.PageBlock{
		Id:             bson.NewObjectId(),
		PublicId:       generatePublicID("block"),
		SubscriberId:   c.subscriberID,
		SectionID:      node.SectionID,
		BlockType:      node.BlockType,
		SourceURL:      node.SourceURL,
		MediaPublicId:  node.MediaPublicId,
		PlayerPresetId: node.PlayerPresetId,
		Autoplay:       node.Autoplay,
	}

	if node.ContentGen != nil {
		block.ContentGen = c.compileContentGen(node.ContentGen)
	}
	if node.AIContext != nil {
		block.AIContext = c.compileAIContext(node.AIContext)
	}

	return block
}

func (c *Compiler) compileForm(node *FormNode) *pkgmodels.PageForm {
	form := &pkgmodels.PageForm{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("form"),
		SubscriberId: c.subscriberID,
		Name:         node.Name,
		FormType:     node.FormType,
		ProductId:    node.ProductID,
		OfferID:      node.OfferID,
	}

	for _, fNode := range node.Fields {
		field := &pkgmodels.FormField{
			FieldName:   fNode.FieldName,
			FieldType:   fNode.FieldType,
			Required:    fNode.Required,
			CustomField: fNode.CustomField,
		}
		form.Fields = append(form.Fields, field)
	}

	// Compile order bumps
	for _, obNode := range node.OrderBumps {
		ob := &pkgmodels.OrderBump{
			Text: obNode.Text,
		}
		// Resolve offer by name
		if offer, ok := c.offerMap[obNode.OfferID]; ok {
			ob.OfferID = offer.Id
		}
		form.OrderBumps = append(form.OrderBumps, ob)
	}

	return form
}

func (c *Compiler) compileContentGen(node *ContentGenNode) *pkgmodels.ContentGenConfig {
	return &pkgmodels.ContentGenConfig{
		Length:       node.Length,
		ContextURLs:  node.ContextURLs,
		PromptAppend: node.PromptAppend,
		Status:       "pending",
	}
}

func (c *Compiler) compileAIContext(node *AIContextNode) *pkgmodels.AIContextBlock {
	return &pkgmodels.AIContextBlock{
		ContextURLs:  node.ContextURLs,
		ContextRefs:  node.ContextRefs,
		ContextMode:  node.Mode,
	}
}

// ---------- ID Generation ----------

var idCounter int

// idSessionPrefix is a short random string regenerated each time the counter
// is reset so that public_ids are unique across server restarts and separate
// compilation sessions.  Format: "sscript_{prefix}_{session}_{counter}"
var idSessionPrefix string

func init() {
	refreshSessionPrefix()
}

func refreshSessionPrefix() {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, 6)
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	idSessionPrefix = string(b)
}

func generatePublicID(prefix string) string {
	idCounter++
	return fmt.Sprintf("sscript_%s_%s_%d", prefix, idSessionPrefix, idCounter)
}

// ResetIDCounter resets the ID counter and generates a fresh session prefix
// so that each compilation produces globally unique public_ids.
func ResetIDCounter() {
	idCounter = 0
	refreshSessionPrefix()
}

// ---------- Site Compilation ----------

// compileSite compiles a SiteNode into a Site entity.
func (c *Compiler) compileSite(node *SiteNode) *pkgmodels.Site {
	site := &pkgmodels.Site{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("site"),
		SubscriberId: c.subscriberID,
		CreatorId:    c.creatorID,
		Name:         node.Name,
		Domain:       node.Domain,
		Theme:        node.Theme,
	}

	if node.SEO != nil {
		site.SEO = &pkgmodels.SEOConfig{
			MetaTitle:         node.SEO.MetaTitle,
			MetaDescription:   node.SEO.MetaDescription,
			OpenGraphImageURL: node.SEO.OpenGraphImageURL,
		}
	}

	if node.Navigation != nil {
		site.Navigation = &pkgmodels.NavigationConfig{
			HeaderLinks: node.Navigation.HeaderLinks,
			FooterLinks: node.Navigation.FooterLinks,
		}
	}

	pageIds := &pkgmodels.BsonCollectionIds{
		CollectionName: pkgmodels.FunnelPageCollection,
	}
	for _, pgNode := range node.Pages {
		page := c.compilePage(pgNode)
		if page != nil {
			site.Pages = append(site.Pages, page)
			pageIds.Ids = append(pageIds.Ids, page.Id)
		}
	}
	site.PageIds = pageIds

	return site
}

// ---------- E-Commerce Compilation ----------

// compileProduct compiles a ProductDeclNode into a Product entity (no price).
func (c *Compiler) compileProduct(node *ProductDeclNode) *pkgmodels.Product {
	product := &pkgmodels.Product{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("product"),
		SubscriberId: c.subscriberID,
		Name:         node.Name,
		Description:  node.Description,
		ProductType:  node.ProductType,
		Status:       "published",
	}

	if node.Title != "" {
		product.Name = node.Title
	}
	if node.Instructor != "" {
		product.InstructorName = node.Instructor
	}
	if node.ThumbnailURL != "" {
		product.ThumbnailURL = node.ThumbnailURL
	}
	if node.Status != "" {
		product.Status = node.Status
	}

	// Handle description_gen for course products
	if node.DescriptionGen != nil {
		product.DescriptionGenStatus = "pending"
		product.DescriptionGenConfig = &pkgmodels.GenConfig{
			Instruction: node.DescriptionGen.Instruction,
			References:  node.DescriptionGen.References,
		}
	}

	// Compile old-style modules (backward compatibility)
	for i, modNode := range node.Modules {
		mod := c.compileModule(modNode, i+1)
		product.Modules = append(product.Modules, mod)
	}

	// Compile course modules for LMS courses
	if node.ProductType == "course" {
		totalLessons := 0
		for i, modNode := range node.Modules {
			courseMod := c.compileCourseModule(modNode, i+1, product)
			product.CourseModules = append(product.CourseModules, courseMod)
			totalLessons += len(courseMod.Lessons)
		}
		product.TotalLessons = totalLessons
	}

	return product
}

// compileModule compiles a ModuleNode into a Module entity.
func (c *Compiler) compileModule(node *ModuleNode, order int) *pkgmodels.Module {
	mod := &pkgmodels.Module{
		Id:    bson.NewObjectId(),
		Title: node.Title,
		Order: order,
	}

	for i, lessonNode := range node.Lessons {
		lesson := c.compileLesson(lessonNode, i+1)
		mod.Lessons = append(mod.Lessons, lesson)
	}

	return mod
}

// compileLesson compiles a LessonNode into a Lesson entity.
func (c *Compiler) compileLesson(node *LessonNode, order int) *pkgmodels.Lesson {
	return &pkgmodels.Lesson{
		Id:            bson.NewObjectId(),
		Title:         node.Title,
		VideoURL:      node.VideoURL,
		MediaPublicId: node.MediaPublicId,
		ContentHTML:   node.ContentHTML,
		IsDraft:       node.IsDraft,
		Order:         order,
	}
}

// compileCourseModule compiles a ModuleNode into a CourseModule entity for LMS courses.
func (c *Compiler) compileCourseModule(node *ModuleNode, defaultOrder int, product *pkgmodels.Product) *pkgmodels.CourseModule {
	slug := node.Slug
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(node.Title, " ", "-"))
	}
	mod := &pkgmodels.CourseModule{
		Slug:  slug,
		Title: node.Title,
		Order: node.Order,
	}
	if mod.Order == 0 {
		mod.Order = defaultOrder
	}
	if mod.Title == "" {
		mod.Title = slug
	}

	for i, lessonNode := range node.Lessons {
		lesson := c.compileCourseLesson(lessonNode, i+1)
		mod.Lessons = append(mod.Lessons, lesson)
	}

	// Compile quizzes if present
	for _, quizNode := range node.Quizzes {
		quiz := c.compileLMSQuiz(quizNode, mod.Slug, product)
		if quiz != nil {
			if mod.QuizSlug == "" {
				mod.QuizSlug = quiz.Slug
			}
			c.pendingQuizzes = append(c.pendingQuizzes, quiz)
		}
	}

	return mod
}

// compileCourseLesson compiles a LessonNode into a CourseLesson entity for LMS courses.
func (c *Compiler) compileCourseLesson(node *LessonNode, defaultOrder int) *pkgmodels.CourseLesson {
	lesson := &pkgmodels.CourseLesson{
		Slug:          node.Slug,
		Title:         node.Title,
		Order:         node.Order,
		VideoURL:      node.VideoURL,
		MediaPublicId: node.MediaPublicId,
		Duration:      node.Duration,
		ContentHTML:   node.ContentHTML,
		IsFree:        node.IsFree,
		IsDraft:       node.IsDraft,
		DripDays:      node.DripDays,
	}
	if lesson.Order == 0 {
		lesson.Order = defaultOrder
	}
	if lesson.Title == "" {
		lesson.Title = node.Slug
	}

	// Handle content_gen
	if node.ContentGen != nil {
		lesson.ContentGenStatus = "pending"
		lesson.ContentGenConfig = &pkgmodels.GenConfig{
			Instruction: node.ContentGen.Instruction,
			References:  node.ContentGen.References,
			Theme:       node.ContentGen.Theme,
		}
	}

	// Handle video stub fields
	if node.VideoMode != "" {
		lesson.VideoMode = pkgmodels.VideoMode(node.VideoMode)
	}
	if node.VideoStubScript != "" {
		lesson.VideoStubScript = node.VideoStubScript
	}
	if node.VideoStubDescription != "" {
		lesson.VideoStubDescription = node.VideoStubDescription
	}
	if node.ContentMarkdown != "" {
		lesson.ContentMarkdown = node.ContentMarkdown
	}

	return lesson
}

// compileCourseDecl compiles a CourseDeclNode into a Product entity (syntactic sugar).
// Supports both "authored" mode (inline modules/lessons) and "generate" mode
// (emits a generation job config instead of a materialized course tree).
func (c *Compiler) compileCourseDecl(node *CourseDeclNode) *pkgmodels.Product {
	productNode := &ProductDeclNode{
		NodeBase:       node.NodeBase,
		Name:           node.Name,
		ProductType:    "course",
		Description:    node.Description,
		Instructor:     node.Instructor,
		ThumbnailURL:   node.ThumbnailURL,
		Status:         node.Status,
		DescriptionGen: node.DescriptionGen,
		Modules:        node.Modules,
	}
	product := c.compileProduct(productNode)

	mode := node.Mode
	if mode == "" {
		mode = "authored"
	}

	if node.Audience != "" || node.Outcome != "" || node.Tone != "" || node.ExtraContext != "" || node.DefaultMedia != "" || len(node.References) > 0 || mode != "authored" {
		product.CourseGenConfig = &pkgmodels.CourseGenConfig{
			Mode:         mode,
			Audience:     node.Audience,
			Outcome:      node.Outcome,
			Tone:         node.Tone,
			ExtraContext: node.ExtraContext,
			DefaultMedia: node.DefaultMedia,
			References:   node.References,
		}
	}

	// Persist certificate config if specified
	if node.CertConfig != nil {
		certTemplate := &pkgmodels.CertificateTemplate{
			Id:              bson.NewObjectId(),
			PublicId:        generatePublicID("cert-tmpl"),
			ProductPublicId: product.PublicId,
			SubscriberId:    c.subscriberID,
			Title:           node.CertConfig.Title,
			TemplateName:    node.CertConfig.TemplateName,
			LogoURL:         node.CertConfig.LogoURL,
			AccentColor:     node.CertConfig.AccentColor,
			Enabled:         node.CertConfig.Enabled,
		}
		if certTemplate.Title == "" {
			certTemplate.Title = product.Name + " Certificate"
		}
		if certTemplate.TemplateName == "" {
			certTemplate.TemplateName = "default"
		}
		c.certTemplates = append(c.certTemplates, certTemplate)
	}

	return product
}

// resolveCourseRef resolves a course_ref("name") to the course's PublicId.
// When projection is provided, resolves to specific course properties.
func (c *Compiler) resolveCourseRef(name string) string {
	if product, ok := c.productMap[name]; ok {
		return product.PublicId
	}
	return ""
}

// resolveCourseRefProjection resolves a course_ref("name").projection to a value.
func (c *Compiler) resolveCourseRefProjection(name, projection string) interface{} {
	product, ok := c.productMap[name]
	if !ok {
		return nil
	}
	switch projection {
	case "title":
		return product.Name
	case "description":
		return product.Description
	case "outline":
		// Return a structural summary of modules + lessons
		var outline []map[string]interface{}
		for _, mod := range product.CourseModules {
			modMap := map[string]interface{}{
				"title": mod.Title,
				"slug":  mod.Slug,
				"order": mod.Order,
			}
			var lessons []string
			for _, l := range mod.Lessons {
				lessons = append(lessons, l.Title)
			}
			modMap["lessons"] = lessons
			outline = append(outline, modMap)
		}
		return outline
	case "lesson_titles":
		var titles []string
		for _, mod := range product.CourseModules {
			for _, l := range mod.Lessons {
				titles = append(titles, l.Title)
			}
		}
		return titles
	case "certificate_enabled":
		for _, tmpl := range c.certTemplates {
			if tmpl.ProductPublicId == product.PublicId {
				return tmpl.Enabled
			}
		}
		return false
	case "module_count":
		return len(product.CourseModules)
	case "total_lessons":
		return product.TotalLessons
	default:
		return product.PublicId
	}
}

// compileLMSQuiz compiles an LMSQuizNode into an LMSQuiz entity.
func (c *Compiler) compileLMSQuiz(node *LMSQuizNode, moduleSlug string, product *pkgmodels.Product) *pkgmodels.LMSQuiz {
	slug := node.Slug
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(node.Title, " ", "-"))
	}
	quiz := &pkgmodels.LMSQuiz{
		Id:            bson.NewObjectId(),
		PublicId:      generatePublicID("quiz"),
		SubscriberId:  c.subscriberID,
		Slug:          slug,
		Title:         node.Title,
		ModuleSlug:    moduleSlug,
		PassThreshold: node.PassThreshold,
		MaxAttempts:   node.MaxAttempts,
	}
	if product != nil {
		quiz.ProductID = product.Id
	}
	if quiz.Title == "" {
		quiz.Title = slug
	}
	if quiz.PassThreshold == 0 {
		quiz.PassThreshold = 70 // default
	}
	if quiz.MaxAttempts == 0 {
		quiz.MaxAttempts = 3 // default
	}

	for i, qNode := range node.Questions {
		question := c.compileLMSQuestion(qNode, i+1)
		quiz.Questions = append(quiz.Questions, question)
	}

	return quiz
}

// compileLMSQuestion compiles an LMSQuestionNode into an LMSQuizQuestion.
func (c *Compiler) compileLMSQuestion(node *LMSQuestionNode, order int) *pkgmodels.LMSQuizQuestion {
	q := &pkgmodels.LMSQuizQuestion{
		Slug:  node.Slug,
		Type:  node.Type,
		Title: node.Title,
		Order: order,
	}
	if q.Title == "" {
		q.Title = node.Slug
	}

	if node.Options != nil {
		q.Options = node.Options
	}

	// Handle answer based on type
	switch ansVal := node.Answer.(type) {
	case int:
		q.CorrectAnswer = ansVal
	case string:
		q.CorrectText = ansVal
	}

	return q
}

// compileOffer compiles an OfferDeclNode into an Offer entity.
func (c *Compiler) compileOffer(node *OfferDeclNode) *pkgmodels.Offer {
	offer := &pkgmodels.Offer{
		Id:           bson.NewObjectId(),
		PublicId:     generatePublicID("offer"),
		SubscriberId: c.subscriberID,
		Title:        node.Name,
		PricingModel: node.PricingModel,
		Amount:       int64(node.Price * 100), // Convert dollars to cents
		Currency:     node.Currency,
		GrantedBadges: node.GrantedBadges,
	}

	// Resolve included products by name
	for _, productName := range node.IncludedProducts {
		if product, ok := c.productMap[productName]; ok {
			offer.IncludedProducts = append(offer.IncludedProducts, product.Id)
		} else {
			c.errorf(node.Pos, "included product %q not found", productName)
		}
	}

	// Pre-create badges referenced in the offer
	for _, badgeName := range node.GrantedBadges {
		if _, exists := c.badges[badgeName]; !exists {
			badge := &pkgmodels.Badge{
				Id:           bson.NewObjectId(),
				PublicId:     generatePublicID("badge"),
				SubscriberId: c.subscriberID,
				CreatorId:    c.creatorID,
				Name:         badgeName,
				Description:  fmt.Sprintf("Auto-generated badge: %s", badgeName),
			}
			c.badges[badgeName] = badge
		}
	}

	return offer
}

// ---------- Video Intelligence Compilation ----------

// compileMediaDecl compiles a MediaDeclNode into a Media entity.
func (c *Compiler) compileMediaDecl(node *MediaDeclNode) *pkgmodels.Media {
media := &pkgmodels.Media{
Id:          bson.NewObjectId(),
PublicId:    generatePublicID("media"),
SubscriberId: c.subscriberID,
Title:       node.Title,
Description: node.Description,
Kind:        node.Kind,
SourceURL:   node.SourceURL,
PosterURL:   node.PosterURL,
Status:      "draft",
Tags:        node.Tags,
Folder:      node.Folder,
}

if media.Title == "" {
media.Title = node.Name
}
if media.Kind == "" {
media.Kind = "video"
}

// Compile chapters
for _, ch := range node.Chapters {
chapter := &pkgmodels.MediaChapter{
PublicId: generatePublicID("chapter"),
Title:    ch.Title,
StartSec: ch.StartSec,
EndSec:   ch.EndSec,
}
media.Chapters = append(media.Chapters, chapter)
}

// Compile interactions
for _, inter := range node.Interactions {
interaction := &pkgmodels.MediaInteraction{
PublicId: generatePublicID("interaction"),
Kind:     inter.Kind,
StartSec: inter.StartSec,
EndSec:   inter.EndSec,
}
switch inter.Kind {
case "turnstile":
interaction.Config = pkgmodels.TurnstileConfig{
Required: inter.Required,
Fields:   inter.Fields,
}
case "cta":
interaction.Config = pkgmodels.CTAConfig{
Text:       inter.Text,
URL:        inter.URL,
ButtonText: inter.ButtonText,
}
case "annotation":
interaction.Config = pkgmodels.AnnotationConfig{
Text: inter.Text,
URL:  inter.URL,
}
}
media.Interactions = append(media.Interactions, interaction)
}

// Compile badge rules
for _, br := range node.BadgeRules {
rule := &pkgmodels.MediaBadgeRule{
PublicId:      generatePublicID("badge_rule"),
EventName:     br.EventName,
Operator:      br.Operator,
Threshold:     br.Threshold,
BadgePublicId: br.BadgeName,
Enabled:       br.Enabled,
}

// Pre-create referenced badge
if br.BadgeName != "" {
if _, exists := c.badges[br.BadgeName]; !exists {
badge := &pkgmodels.Badge{
Id:           bson.NewObjectId(),
PublicId:     generatePublicID("badge"),
SubscriberId: c.subscriberID,
CreatorId:    c.creatorID,
Name:         br.BadgeName,
Description:  fmt.Sprintf("Auto-generated badge: %s", br.BadgeName),
}
c.badges[br.BadgeName] = badge
}
}

media.BadgeRules = append(media.BadgeRules, rule)
}

// Resolve player preset reference
if node.PlayerPreset != "" {
if preset, ok := c.presetMap[node.PlayerPreset]; ok {
media.PlayerPresetID = &preset.Id
}
}

return media
}

// compilePlayerPresetDecl compiles a PlayerPresetDeclNode into a PlayerPreset entity.
func (c *Compiler) compilePlayerPresetDecl(node *PlayerPresetDeclNode) *pkgmodels.PlayerPreset {
	return &pkgmodels.PlayerPreset{
		Id:                bson.NewObjectId(),
		PublicId:          generatePublicID("preset"),
		SubscriberId:      c.subscriberID,
		Name:              node.Name,
		// Appearance
		PlayerColor:       node.PlayerColor,
		ShowControls:      node.ShowControls,
		// Show Buttons
		ShowRewind:        node.ShowRewind,
		ShowFastForward:   node.ShowFastForward,
		ShowSkip:          node.ShowSkip,
		ShowDownload:      node.ShowDownload,
		HideProgressBar:   node.HideProgressBar,
		// Other controls
		ShowBigPlayButton: node.ShowBigPlayButton,
		AllowFullscreen:   node.AllowFullscreen,
		AllowPlaybackRate: node.AllowPlaybackRate,
		AllowSeeking:      node.AllowSeeking,
		// Behaviour
		Autoplay:          node.Autoplay,
		MutedDefault:      node.MutedDefault,
		DisablePause:      node.DisablePause,
		// End behaviour
		EndBehavior:       node.EndBehavior,
		// Player style
		RoundedPlayer:     node.RoundedPlayer,
		// Chapter controls
		ChapterStyle:      node.ChapterStyle,
		ChapterPosition:   node.ChapterPosition,
		ChapterClickJump:  node.ChapterClickJump,
	}
}

// compileChannelDecl compiles a ChannelDeclNode into a MediaChannel entity.
func (c *Compiler) compileChannelDecl(node *ChannelDeclNode) *pkgmodels.MediaChannel {
channel := &pkgmodels.MediaChannel{
Id:          bson.NewObjectId(),
PublicId:    generatePublicID("channel"),
SubscriberId: c.subscriberID,
Title:       node.Title,
Description: node.Description,
Layout:      node.Layout,
Theme:       node.Theme,
}
if channel.Title == "" {
channel.Title = node.Name
}

// Resolve media items by name
for i, mediaName := range node.Items {
item := &pkgmodels.MediaChannelItem{
MediaPublicId: mediaName,
Order:         i + 1,
}
// If we compiled a media with this name, use its public_id
if media, ok := c.mediaMap[mediaName]; ok {
item.MediaPublicId = media.PublicId
}
channel.Items = append(channel.Items, item)
}

return channel
}

// compileMediaWebhookDecl compiles a MediaWebhookDeclNode into a MediaWebhook entity.
func (c *Compiler) compileMediaWebhookDecl(node *MediaWebhookDeclNode) *pkgmodels.MediaWebhook {
return &pkgmodels.MediaWebhook{
Id:           bson.NewObjectId(),
PublicId:     generatePublicID("webhook"),
SubscriberId: c.subscriberID,
Name:         node.Name,
URL:          node.URL,
EventTypes:   node.EventTypes,
Enabled:      node.Enabled,
}
}
