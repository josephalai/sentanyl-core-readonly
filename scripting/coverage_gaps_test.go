package scripting

import (
	"testing"

	pkgmodels "github.com/josephalai/sentanyl/pkg/models"

	"gopkg.in/mgo.v2/bson"
)

// Helper: compile a fixture and assert no errors
func compileFixtureCoverage(t *testing.T, src string) *CompileResult {
	t.Helper()
	ResetIDCounter()
	result := CompileScript(src, "sub_coverage", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error at %s: %s", d.Pos, d.Message)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// 1. FixtureNextStoryHopping — next_story in on_complete & on_fail
// ---------------------------------------------------------------------------

func TestCoverageNextStoryHopping(t *testing.T) {
	result := compileFixtureCoverage(t, FixtureNextStoryHopping)

	// Must produce 2 stories
	if len(result.Stories) != 2 {
		t.Fatalf("expected 2 stories, got %d", len(result.Stories))
	}

	story1 := result.Stories[0]
	story2 := result.Stories[1]

	if story1.Name != "Welcome Series" {
		t.Errorf("expected story 1 name 'Welcome Series', got %q", story1.Name)
	}
	if story2.Name != "Post-Purchase Follow-Up" {
		t.Errorf("expected story 2 name 'Post-Purchase Follow-Up', got %q", story2.Name)
	}

	// Verify on_complete has next_story wired to story 2
	if story1.OnComplete.NextStory == nil {
		t.Fatal("expected on_complete.next_story to be wired")
	}
	if story1.OnComplete.NextStory.Name != "Post-Purchase Follow-Up" {
		t.Errorf("expected on_complete.next_story name 'Post-Purchase Follow-Up', got %q", story1.OnComplete.NextStory.Name)
	}
	if story1.OnComplete.NextStoryId == nil {
		t.Fatal("expected on_complete.next_story_id to be set")
	}
	if story1.OnComplete.NextStoryId.Id != story2.Id {
		t.Errorf("on_complete next_story_id mismatch: %v != %v", story1.OnComplete.NextStoryId.Id, story2.Id)
	}

	// Verify on_fail has next_story wired to story 2
	if story1.OnFail.NextStory == nil {
		t.Fatal("expected on_fail.next_story to be wired")
	}
	if story1.OnFail.NextStory.Name != "Post-Purchase Follow-Up" {
		t.Errorf("expected on_fail.next_story name 'Post-Purchase Follow-Up', got %q", story1.OnFail.NextStory.Name)
	}
	if story1.OnFail.NextStoryId == nil {
		t.Fatal("expected on_fail.next_story_id to be set")
	}

	// Verify badges
	if _, ok := result.Badges["welcome_done"]; !ok {
		t.Error("expected 'welcome_done' badge")
	}
	if _, ok := result.Badges["welcome_failed"]; !ok {
		t.Error("expected 'welcome_failed' badge")
	}

	// Each story has 1 storyline, 1 enactment, 1 scene
	if len(story1.Storylines) != 1 {
		t.Fatalf("expected 1 storyline in story 1, got %d", len(story1.Storylines))
	}
	if len(story1.Storylines[0].Acts) != 1 {
		t.Fatalf("expected 1 enactment in story 1, got %d", len(story1.Storylines[0].Acts))
	}

	if len(story2.Storylines) != 1 {
		t.Fatalf("expected 1 storyline in story 2, got %d", len(story2.Storylines))
	}
	if len(story2.Storylines[0].Acts) != 1 {
		t.Fatalf("expected 1 enactment in story 2, got %d", len(story2.Storylines[0].Acts))
	}
}

// ---------------------------------------------------------------------------
// 2. FixtureStorylineOnFailRoutes — storyline on_fail conditional routes
// ---------------------------------------------------------------------------

func TestCoverageStorylineOnFailRoutes(t *testing.T) {
	result := compileFixtureCoverage(t, FixtureStorylineOnFailRoutes)

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	story := result.Stories[0]

	// 3 storylines
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	sl1 := story.Storylines[0]
	sl2 := story.Storylines[1]
	sl3 := story.Storylines[2]

	if sl1.Name != "Main Flow" {
		t.Errorf("expected SL1 name 'Main Flow', got %q", sl1.Name)
	}
	if sl2.Name != "Premium Recovery" {
		t.Errorf("expected SL2 name 'Premium Recovery', got %q", sl2.Name)
	}
	if sl3.Name != "Standard Recovery" {
		t.Errorf("expected SL3 name 'Standard Recovery', got %q", sl3.Name)
	}

	// SL1 on_fail should have badge transaction
	if sl1.OnFail.BadgeTransaction == nil {
		t.Fatal("expected on_fail badge transaction")
	}
	if len(sl1.OnFail.BadgeTransaction.GiveBadges) != 1 {
		t.Fatalf("expected 1 give_badge in on_fail, got %d", len(sl1.OnFail.BadgeTransaction.GiveBadges))
	}
	if sl1.OnFail.BadgeTransaction.GiveBadges[0].Name != "sl1_failed" {
		t.Errorf("expected give_badge 'sl1_failed', got %q", sl1.OnFail.BadgeTransaction.GiveBadges[0].Name)
	}

	// SL1 on_fail should have 2 conditional routes
	if len(sl1.OnFail.ConditionalRoutes) != 2 {
		t.Fatalf("expected 2 conditional routes in on_fail, got %d", len(sl1.OnFail.ConditionalRoutes))
	}

	// Route 1: must_have "premium" -> Premium Recovery (priority 2)
	route1 := sl1.OnFail.ConditionalRoutes[0]
	if route1.Priority != 2 {
		t.Errorf("expected route 1 priority 2, got %d", route1.Priority)
	}
	if len(route1.RequiredBadges.MustHave) != 1 {
		t.Errorf("expected 1 must_have on route 1, got %d", len(route1.RequiredBadges.MustHave))
	}
	if route1.NextStoryline == nil {
		t.Error("expected route 1 to have next_storyline wired")
	} else if route1.NextStoryline.Name != "Premium Recovery" {
		t.Errorf("expected route 1 next_storyline 'Premium Recovery', got %q", route1.NextStoryline.Name)
	}

	// Route 2: must_not_have "premium" -> Standard Recovery (priority 1)
	route2 := sl1.OnFail.ConditionalRoutes[1]
	if route2.Priority != 1 {
		t.Errorf("expected route 2 priority 1, got %d", route2.Priority)
	}
	if len(route2.RequiredBadges.MustNotHave) != 1 {
		t.Errorf("expected 1 must_not_have on route 2, got %d", len(route2.RequiredBadges.MustNotHave))
	}
	if route2.NextStoryline == nil {
		t.Error("expected route 2 to have next_storyline wired")
	} else if route2.NextStoryline.Name != "Standard Recovery" {
		t.Errorf("expected route 2 next_storyline 'Standard Recovery', got %q", route2.NextStoryline.Name)
	}

	// Verify badge
	if _, ok := result.Badges["sl1_failed"]; !ok {
		t.Error("expected 'sl1_failed' badge")
	}
	if _, ok := result.Badges["premium"]; !ok {
		t.Error("expected 'premium' badge")
	}
}

// ---------------------------------------------------------------------------
// 3. FixtureSceneTemplateName — template keyword in scenes
// ---------------------------------------------------------------------------

func TestCoverageSceneTemplateName(t *testing.T) {
	result := compileFixtureCoverage(t, FixtureSceneTemplateName)

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	story := result.Stories[0]

	if len(story.Storylines) != 1 {
		t.Fatalf("expected 1 storyline, got %d", len(story.Storylines))
	}
	sl := story.Storylines[0]

	// 2 enactments
	if len(sl.Acts) != 2 {
		t.Fatalf("expected 2 enactments, got %d", len(sl.Acts))
	}

	// Enactment 1: template "welcome_template"
	en1 := sl.Acts[0]
	scene1 := en1.SendScene
	if scene1 == nil || scene1.Message == nil || scene1.Message.Content == nil {
		t.Fatal("expected scene 1 to have message content")
	}
	if scene1.Message.Content.Template == nil {
		t.Fatal("expected scene 1 to have template")
	}
	if scene1.Message.Content.Template.Name != "welcome_template" {
		t.Errorf("expected template name 'welcome_template', got %q", scene1.Message.Content.Template.Name)
	}
	if scene1.Message.Content.TemplateId == nil {
		t.Fatal("expected scene 1 to have template_id")
	}

	// Enactment 2: template "followup_template"
	en2 := sl.Acts[1]
	scene2 := en2.SendScene
	if scene2 == nil || scene2.Message == nil || scene2.Message.Content == nil {
		t.Fatal("expected scene 2 to have message content")
	}
	if scene2.Message.Content.Template == nil {
		t.Fatal("expected scene 2 to have template")
	}
	if scene2.Message.Content.Template.Name != "followup_template" {
		t.Errorf("expected template name 'followup_template', got %q", scene2.Message.Content.Template.Name)
	}

	// Vars should be populated
	if len(scene1.Message.Content.GivenVars) == 0 {
		t.Error("expected scene 1 to have given_vars")
	}
	if scene1.Message.Content.GivenVars["company_name"] != "Acme Corp" {
		t.Errorf("expected company_name 'Acme Corp', got %q", scene1.Message.Content.GivenVars["company_name"])
	}
	if len(scene2.Message.Content.GivenVars) == 0 {
		t.Error("expected scene 2 to have given_vars")
	}
	if scene2.Message.Content.GivenVars["sender_name"] != "Jane" {
		t.Errorf("expected sender_name 'Jane', got %q", scene2.Message.Content.GivenVars["sender_name"])
	}
}

// ---------------------------------------------------------------------------
// 4. FixtureSceneDefaultsTriggers — scene_defaults trigger injection
// ---------------------------------------------------------------------------

func TestCoverageSceneDefaultsTriggers(t *testing.T) {
	result := compileFixtureCoverage(t, FixtureSceneDefaultsTriggers)

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	story := result.Stories[0]

	if len(story.Storylines) != 1 {
		t.Fatalf("expected 1 storyline, got %d", len(story.Storylines))
	}
	sl := story.Storylines[0]

	// 2 enactments
	if len(sl.Acts) != 2 {
		t.Fatalf("expected 2 enactments, got %d", len(sl.Acts))
	}

	// Each enactment should have the scene_defaults trigger (not_open)
	// PLUS its own explicit click trigger = 2 triggers each
	for i, en := range sl.Acts {
		totalTriggers := 0
		for _, triggers := range en.OnEvent {
			totalTriggers += len(triggers)
		}
		// 1 explicit click trigger + 1 inherited not_open trigger = 2
		if totalTriggers < 2 {
			t.Errorf("enactment %d (%s): expected >= 2 triggers (1 click + 1 inherited not_open), got %d", i, en.Name, totalTriggers)
		}

		// Verify not_open trigger exists (from scene_defaults)
		notOpenTriggers := en.OnEvent[pkgmodels.OnNotOpen]
		if len(notOpenTriggers) == 0 {
			t.Errorf("enactment %d (%s): expected OnNotOpen trigger from scene_defaults", i, en.Name)
		}
	}
}

// ---------------------------------------------------------------------------
// 5. FixtureHandlebarsVars — Handlebars {{ var }} rendering
// ---------------------------------------------------------------------------

func TestCoverageHandlebarsVars(t *testing.T) {
	result := compileFixtureCoverage(t, FixtureHandlebarsVars)

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	story := result.Stories[0]

	if len(story.Storylines) != 1 {
		t.Fatalf("expected 1 storyline, got %d", len(story.Storylines))
	}

	en := story.Storylines[0].Acts[0]
	scene := en.SendScene

	if scene.Message == nil || scene.Message.Content == nil {
		t.Fatal("expected message content")
	}

	content := scene.Message.Content

	// Check vars are stored
	expectedVars := map[string]string{
		"first_name":    "John",
		"product_name":  "Premium Widget",
		"discount_code": "SAVE20",
		"company_name":  "Acme Corp",
	}
	for k, v := range expectedVars {
		if content.GivenVars[k] != v {
			t.Errorf("expected var %s=%q, got %q", k, v, content.GivenVars[k])
		}
	}

	// Subject should contain handlebars placeholders
	if content.Subject == "" {
		t.Error("expected non-empty subject")
	}
	// Body should contain handlebars
	if content.Body == "" {
		t.Error("expected non-empty body")
	}
}

// ---------------------------------------------------------------------------
// All coverage gap fixtures must compile
// ---------------------------------------------------------------------------

func TestCoverageAllFixturesCompile(t *testing.T) {
	fixtures := map[string]string{
		"NextStoryHopping":       FixtureNextStoryHopping,
		"StorylineOnFailRoutes":  FixtureStorylineOnFailRoutes,
		"SceneTemplateName":      FixtureSceneTemplateName,
		"SceneDefaultsTriggers":  FixtureSceneDefaultsTriggers,
		"HandlebarsVars":         FixtureHandlebarsVars,
	}

	for name, src := range fixtures {
		t.Run(name, func(t *testing.T) {
			ResetIDCounter()
			result := CompileScript(src, "sub_coverage", bson.NewObjectId())
			for _, d := range result.Diagnostics {
				if d.Level == DiagError {
					t.Errorf("fixture %s: compile error at %s: %s", name, d.Pos, d.Message)
				}
			}
			if len(result.Stories) == 0 {
				t.Errorf("fixture %s: expected at least 1 story", name)
			}
		})
	}
}
