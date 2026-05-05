package scripting

import (
	"os"
	"path/filepath"
	"testing"

	pkgmodels "github.com/josephalai/sentanyl/pkg/models"
	"gopkg.in/mgo.v2/bson"
)

// ---------- Compiler Tests ----------

func TestCompilerMinimalCampaign(t *testing.T) {
	ResetIDCounter()
	src := `
story "Welcome" {
	priority 5
	storyline "Main" {
		order 1
		enactment "Hook" {
			level 1
			order 1
			scene "Email 1" {
				subject "Welcome!"
				body "<h1>Hello</h1>"
				from_email "team@example.com"
				from_name "Team"
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	story := result.Stories[0]
	if story.Name != "Welcome" {
		t.Errorf("expected story name 'Welcome', got %q", story.Name)
	}
	if story.Priority != 5 {
		t.Errorf("expected priority 5, got %d", story.Priority)
	}
	if story.SubscriberId != "sub_123" {
		t.Errorf("expected subscriber_id 'sub_123', got %q", story.SubscriberId)
	}
	if len(story.Storylines) != 1 {
		t.Fatalf("expected 1 storyline, got %d", len(story.Storylines))
	}
	sl := story.Storylines[0]
	if sl.NaturalOrder != 1 {
		t.Errorf("expected storyline order 1, got %d", sl.NaturalOrder)
	}
	if len(sl.Acts) != 1 {
		t.Fatalf("expected 1 enactment, got %d", len(sl.Acts))
	}
	en := sl.Acts[0]
	if en.Level != 1 {
		t.Errorf("expected level 1, got %d", en.Level)
	}
	if en.SendScene == nil {
		t.Fatal("expected send scene")
	}
	if en.SendScene.Name != "Email 1" {
		t.Errorf("expected scene name 'Email 1', got %q", en.SendScene.Name)
	}
	if en.SendScene.Message == nil {
		t.Fatal("expected message on scene")
	}
	if en.SendScene.Message.Content == nil {
		t.Fatal("expected message content")
	}
	if en.SendScene.Message.Content.Subject != "Welcome!" {
		t.Errorf("expected subject 'Welcome!', got %q", en.SendScene.Message.Content.Subject)
	}
	if en.SendScene.Message.Content.Body != "<h1>Hello</h1>" {
		t.Errorf("expected body '<h1>Hello</h1>', got %q", en.SendScene.Message.Content.Body)
	}
	if en.SendScene.Message.Content.FromEmail != "team@example.com" {
		t.Errorf("expected from_email 'team@example.com', got %q", en.SendScene.Message.Content.FromEmail)
	}
}

func TestCompilerBadgeTransactions(t *testing.T) {
	ResetIDCounter()
	src := `
story "Badges" {
	on_begin { give_badge "entered" }
	on_complete {
		give_badge "completed"
		remove_badge "entered"
	}
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	story := result.Stories[0]

	// Check badges were created
	if len(result.Badges) < 2 {
		t.Errorf("expected at least 2 badges, got %d", len(result.Badges))
	}
	if _, ok := result.Badges["entered"]; !ok {
		t.Error("expected badge 'entered'")
	}
	if _, ok := result.Badges["completed"]; !ok {
		t.Error("expected badge 'completed'")
	}

	// Check on_begin
	if story.OnBegin.BadgeTransaction == nil {
		t.Fatal("expected badge transaction on on_begin")
	}
	if len(story.OnBegin.BadgeTransaction.GiveBadges) != 1 {
		t.Errorf("expected 1 give badge, got %d", len(story.OnBegin.BadgeTransaction.GiveBadges))
	}

	// Check on_complete
	if story.OnComplete.BadgeTransaction == nil {
		t.Fatal("expected badge transaction on on_complete")
	}
	if len(story.OnComplete.BadgeTransaction.GiveBadges) != 1 {
		t.Errorf("expected 1 give badge, got %d", len(story.OnComplete.BadgeTransaction.GiveBadges))
	}
	if len(story.OnComplete.BadgeTransaction.RemoveBadges) != 1 {
		t.Errorf("expected 1 remove badge, got %d", len(story.OnComplete.BadgeTransaction.RemoveBadges))
	}
}

func TestCompilerTriggerGeneration(t *testing.T) {
	ResetIDCounter()
	src := `
story "Triggers" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on click "link_1" {
				within 1d
				do jump_to_enactment "Offer"
				do give_badge "clicked"
			}
			on not_open {
				within 1d
				do retry_scene up_to 2
			}
		}
		enactment "Offer" {
			scene "Offer" { subject "Buy" body "Body" }
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	if len(en.OnEvent) == 0 {
		t.Fatal("expected triggers on enactment")
	}

	// Check OnClick triggers
	clickTriggers := en.OnEvent["OnClick"]
	if len(clickTriggers) != 1 {
		t.Fatalf("expected 1 OnClick trigger, got %d", len(clickTriggers))
	}
	click := clickTriggers[0]
	if click.UserActionValue != "link_1" {
		t.Errorf("expected user action value 'link_1', got %q", click.UserActionValue)
	}
	if click.DoAction == nil {
		t.Fatal("expected action on click trigger")
	}

	// Check NotOpen triggers
	notOpenTriggers := en.OnEvent["NotOpen"]
	if len(notOpenTriggers) != 1 {
		t.Fatalf("expected 1 NotOpen trigger, got %d", len(notOpenTriggers))
	}
}

func TestCompilerRequiredBadges(t *testing.T) {
	ResetIDCounter()
	src := `
story "Badge Guard" {
	required_badges {
		must_have "vip"
		must_not_have "banned"
	}
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	story := result.Stories[0]
	if len(story.RequiredUserBadges.MustHave) != 1 {
		t.Errorf("expected 1 must_have badge, got %d", len(story.RequiredUserBadges.MustHave))
	}
	if len(story.RequiredUserBadges.MustNotHave) != 1 {
		t.Errorf("expected 1 must_not_have badge, got %d", len(story.RequiredUserBadges.MustNotHave))
	}
	if story.RequiredUserBadges.MustHave[0].Name != "vip" {
		t.Errorf("expected must_have badge 'vip', got %q", story.RequiredUserBadges.MustHave[0].Name)
	}
}

func TestCompilerMultiSceneEnactment(t *testing.T) {
	ResetIDCounter()
	src := `
story "Multi" {
	storyline "Main" {
		enactment "Triple" {
			scene "Scene 1" { subject "First" body "Body 1" }
			scene "Scene 2" { subject "Second" body "Body 2" }
			scene "Scene 3" { subject "Third" body "Body 3" }
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	if len(en.SendScenes) != 3 {
		t.Errorf("expected 3 send scenes, got %d", len(en.SendScenes))
	}
	if en.SendScenesIds == nil || len(en.SendScenesIds.Ids) != 3 {
		t.Error("expected 3 scene IDs in SendScenesIds")
	}
	// First scene should also be set as SendScene for backward compat
	if en.SendScene == nil {
		t.Error("expected SendScene set for backward compat")
	}
}

func TestCompilerConditionalRoutes(t *testing.T) {
	ResetIDCounter()
	src := `
story "Routes" {
	storyline "Main" {
		order 1
		on_complete {
			conditional_route {
				required_badges { must_have "vip" }
				next_storyline "VIP"
				priority 1
			}
		}
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
	storyline "VIP" {
		order 2
		enactment "VIP Hook" {
			scene "VIP Email" { subject "VIP" body "Body" }
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	sl := result.Stories[0].Storylines[0]
	if len(sl.OnComplete.ConditionalRoutes) != 1 {
		t.Fatalf("expected 1 conditional route, got %d", len(sl.OnComplete.ConditionalRoutes))
	}
	cr := sl.OnComplete.ConditionalRoutes[0]
	if cr.Priority != 1 {
		t.Errorf("expected priority 1, got %d", cr.Priority)
	}
	if len(cr.RequiredBadges.MustHave) != 1 {
		t.Error("expected 1 must_have badge on conditional route")
	}
	// Check that NextStoryline is wired
	if cr.NextStoryline == nil {
		t.Error("expected NextStoryline to be wired")
	} else if cr.NextStoryline.Name != "VIP" {
		t.Errorf("expected NextStoryline 'VIP', got %q", cr.NextStoryline.Name)
	}
}

func TestCompilerConditionGuards(t *testing.T) {
	ResetIDCounter()
	src := `
story "Conditions" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on click "link" {
				when has_badge "warm"
				when not_has_badge "cold"
				do mark_complete
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	clickTriggers := en.OnEvent["OnClick"]
	if len(clickTriggers) != 1 {
		t.Fatalf("expected 1 click trigger, got %d", len(clickTriggers))
	}
	tr := clickTriggers[0]
	if len(tr.RequiredBadges.MustHave) < 1 {
		t.Error("expected at least 1 must_have badge from condition")
	}
	if len(tr.RequiredBadges.MustNotHave) < 1 {
		t.Error("expected at least 1 must_not_have badge from condition")
	}
}

func TestCompilerAdvanceToNextStoryline(t *testing.T) {
	ResetIDCounter()
	src := `
story "Advance" {
	storyline "Main" {
		order 1
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on click "buy" {
				do advance_to_next_storyline
			}
		}
	}
	storyline "Followup" {
		order 2
		enactment "Follow" {
			scene "Follow Email" { subject "Follow up" body "Body" }
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	clickTriggers := en.OnEvent["OnClick"]
	if len(clickTriggers) != 1 {
		t.Fatalf("expected 1 click trigger, got %d", len(clickTriggers))
	}
	action := clickTriggers[0].DoAction
	if action == nil {
		t.Fatal("expected action")
	}
	if !action.AdvanceToNextStoryline {
		t.Error("expected AdvanceToNextStoryline to be true")
	}
}

func TestCompilerEndStory(t *testing.T) {
	ResetIDCounter()
	src := `
story "End" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on click "stop" {
				do end_story
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	clickTriggers := en.OnEvent["OnClick"]
	action := clickTriggers[0].DoAction
	if !action.EndStory {
		t.Error("expected EndStory to be true")
	}
}

func TestCompilerUnsubscribe(t *testing.T) {
	ResetIDCounter()
	src := `
story "Unsub" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on unsubscribe {
				do unsubscribe
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	unsubTriggers := en.OnEvent["OnUnsubscribe"]
	if len(unsubTriggers) != 1 {
		t.Fatalf("expected 1 unsubscribe trigger, got %d", len(unsubTriggers))
	}
	action := unsubTriggers[0].DoAction
	if !action.Unsubscribe {
		t.Error("expected Unsubscribe to be true")
	}
}

func TestCompilerSkipToNextStorylineOnExpiry(t *testing.T) {
	ResetIDCounter()
	src := `
story "Skip" {
	storyline "Main" {
		enactment "Hook" {
			skip_to_next_storyline_on_expiry true
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	if !en.SkipToNextStorylineOnExpiry {
		t.Error("expected SkipToNextStorylineOnExpiry to be true")
	}
}

func TestCompilerPersistScope(t *testing.T) {
	ResetIDCounter()
	src := `
story "Scoped" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on click "link" {
				persist_scope "story"
				do mark_complete
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	clickTriggers := en.OnEvent["OnClick"]
	if len(clickTriggers) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(clickTriggers))
	}
	if string(clickTriggers[0].PersistScope) != "story" {
		t.Errorf("expected persist scope 'story', got %q", clickTriggers[0].PersistScope)
	}
}

func TestCompilerEntityIDsAreValid(t *testing.T) {
	ResetIDCounter()
	src := `
story "IDs" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	story := result.Stories[0]
	if !story.Id.Valid() {
		t.Error("story ID should be valid")
	}
	if story.PublicId == "" {
		t.Error("story PublicId should not be empty")
	}
	sl := story.Storylines[0]
	if !sl.Id.Valid() {
		t.Error("storyline ID should be valid")
	}
	en := sl.Acts[0]
	if !en.Id.Valid() {
		t.Error("enactment ID should be valid")
	}
	if en.SendScene == nil {
		t.Fatal("expected scene")
	}
	if !en.SendScene.Id.Valid() {
		t.Error("scene ID should be valid")
	}
}

func TestCompilerStorylineIds(t *testing.T) {
	ResetIDCounter()
	src := `
story "StorylineIds" {
	storyline "SL1" {
		order 1
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
	storyline "SL2" {
		order 2
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	story := result.Stories[0]
	if story.StorylineIds == nil || len(story.StorylineIds.Ids) != 2 {
		t.Error("expected 2 storyline IDs in StorylineIds")
	}
}

func TestCompilerElseTrigger(t *testing.T) {
	ResetIDCounter()
	src := `
story "Else" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on click "link" {
				do mark_complete
				else {
					do next_scene
				}
			}
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	// Should have OnClick trigger AND OnElse trigger
	clickTriggers := en.OnEvent["OnClick"]
	if len(clickTriggers) != 1 {
		t.Errorf("expected 1 OnClick trigger, got %d", len(clickTriggers))
	}
	elseTriggers := en.OnEvent["OnElse"]
	if len(elseTriggers) != 1 {
		t.Errorf("expected 1 OnElse trigger, got %d", len(elseTriggers))
	}
}

func TestCompilerStartAndCompleteTriggers(t *testing.T) {
	ResetIDCounter()
	src := `
story "Triggers" {
	start_trigger "enrolled"
	complete_trigger "graduated"
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := CompileScript(src, "sub_123", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	story := result.Stories[0]
	if story.StartTrigger == nil {
		t.Error("expected start trigger")
	} else {
		if story.StartTrigger.Name != "enrolled" {
			t.Errorf("expected start trigger badge 'enrolled', got %q", story.StartTrigger.Name)
		}
		if story.StartTrigger.Badge == nil {
			t.Error("expected badge on start trigger")
		}
	}
	if story.CompleteTrigger == nil {
		t.Error("expected complete trigger")
	} else if story.CompleteTrigger.Name != "graduated" {
		t.Errorf("expected complete trigger badge 'graduated', got %q", story.CompleteTrigger.Name)
	}
}

func TestCompilerCampaignBasicFixture(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("fixtures", "campaign_basic.ss"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	ResetIDCounter()
	result := CompileScript(string(src), "sub_test", bson.NewObjectId())

	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("compile error: %s", d)
		}
	}
	if len(result.Campaigns) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(result.Campaigns))
	}
	c := result.Campaigns[0]
	if c.Status != pkgmodels.CampaignStatusDraft {
		t.Errorf("expected status %q, got %q", pkgmodels.CampaignStatusDraft, c.Status)
	}
	if c.EmailGenConfig == nil {
		t.Fatal("expected EmailGenConfig populated from subject_gen / body_gen / context_pack")
	}
	if c.EmailGenConfig.SubjectInstruction == "" || c.EmailGenConfig.BodyInstruction == "" {
		t.Errorf("expected subject + body instructions, got %+v", c.EmailGenConfig)
	}
	if len(c.EmailGenConfig.ContextPackRefs) != 1 || c.EmailGenConfig.ContextPackRefs[0] != "brand-tone-v1" {
		t.Errorf("expected context pack ref, got %v", c.EmailGenConfig.ContextPackRefs)
	}
	if len(c.Audience.MustHave) != 1 || c.Audience.MustHave[0] != "paid_subscriber" {
		t.Errorf("expected must_have, got %v", c.Audience.MustHave)
	}
	if len(c.Audience.MustNotHave) != 1 || c.Audience.MustNotHave[0] != "churned" {
		t.Errorf("expected must_not_have, got %v", c.Audience.MustNotHave)
	}
	if len(c.ClickRules) != 1 {
		t.Fatalf("expected 1 click rule, got %d", len(c.ClickRules))
	}
	if c.ClickRules[0].URLPattern != "https://acme.com/v2" || c.ClickRules[0].AwardBadge != "v2_announce_click" {
		t.Errorf("unexpected click rule: %+v", c.ClickRules[0])
	}
	if c.PublicId == "" || c.Id == "" {
		t.Errorf("expected PublicId + Id populated")
	}
}
