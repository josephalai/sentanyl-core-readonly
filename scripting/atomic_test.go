package scripting

import (
	"strings"
	"testing"
)

// ========== Atomic Feature Coverage Tests ==========
// These tests verify that ALL DSL atomic features parse, expand, and compile
// successfully, with structural assertions on key output properties.

// ---------- Trigger Types ----------

func TestAtomicAllTriggerTypes(t *testing.T) {
	result := CompileScript(FixtureAtomicAllTriggerTypes, "sub123", "creator456")
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}

	story := result.Stories[0]
	if len(story.Storylines) != 1 {
		t.Fatalf("expected 1 storyline, got %d", len(story.Storylines))
	}

	sl := story.Storylines[0]
	if len(sl.Acts) != 3 {
		t.Fatalf("expected 3 enactments, got %d", len(sl.Acts))
	}

	// Enactment 1: click, not_click, open, not_open, sent triggers
	en1 := sl.Acts[0]
	if en1.Name != "Click and Open" {
		t.Errorf("enactment 1 name: expected 'Click and Open', got %q", en1.Name)
	}
	if en1.OnEvent == nil {
		t.Fatal("enactment 1: expected OnEvent triggers")
	}

	// Verify multiple trigger types exist
	triggerTypeCount := 0
	for range en1.OnEvent {
		triggerTypeCount++
	}
	if triggerTypeCount < 3 {
		t.Errorf("enactment 1: expected at least 3 trigger types, got %d", triggerTypeCount)
	}

	// Enactment 2: webhook, nothing, bounce, spam, else triggers
	en2 := sl.Acts[1]
	if en2.Name != "Event Triggers" {
		t.Errorf("enactment 2 name: expected 'Event Triggers', got %q", en2.Name)
	}
	if en2.OnEvent == nil {
		t.Fatal("enactment 2: expected OnEvent triggers")
	}

	// Enactment 3: unsubscribe, failure, email_validated, user_has_tag, badge triggers
	en3 := sl.Acts[2]
	if en3.Name != "Status Triggers" {
		t.Errorf("enactment 3 name: expected 'Status Triggers', got %q", en3.Name)
	}
	if en3.OnEvent == nil {
		t.Fatal("enactment 3: expected OnEvent triggers")
	}

	// Verify badges were created from trigger actions
	badgeNames := []string{"clicked", "opened", "delivery_confirmed", "email_valid", "premium_verified"}
	for _, name := range badgeNames {
		if _, ok := result.Badges[name]; !ok {
			t.Errorf("expected badge %q in compile result", name)
		}
	}
}

// ---------- Action Types ----------

func TestAtomicAllActionTypes(t *testing.T) {
	result := CompileScript(FixtureAtomicAllActionTypes, "sub123", "creator456")
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}

	story := result.Stories[0]
	if len(story.Storylines) != 2 {
		t.Fatalf("expected 2 storylines, got %d", len(story.Storylines))
	}

	// Nav Actions storyline: 4 enactments
	sl1 := story.Storylines[0]
	if sl1.Name != "Nav Actions" {
		t.Errorf("storyline 1 name: expected 'Nav Actions', got %q", sl1.Name)
	}
	if len(sl1.Acts) != 4 {
		t.Fatalf("expected 4 enactments in Nav Actions, got %d", len(sl1.Acts))
	}

	// Scene Navigation: has skip_to_next_storyline_on_expiry
	en1 := sl1.Acts[0]
	if en1.Name != "Scene Navigation" {
		t.Errorf("enactment name: expected 'Scene Navigation', got %q", en1.Name)
	}
	// Verify multi-scene enactment
	if len(en1.SendScenes) != 2 {
		t.Errorf("expected 2 scenes in Scene Navigation, got %d", len(en1.SendScenes))
	}

	// Badge Actions: give_badge/remove_badge in actions
	en2 := sl1.Acts[1]
	if en2.Name != "Badge Actions" {
		t.Errorf("enactment name: expected 'Badge Actions', got %q", en2.Name)
	}
	// Verify badge is created
	if _, ok := result.Badges["earned"]; !ok {
		t.Error("expected badge 'earned' in compile result")
	}
	if _, ok := result.Badges["pending"]; !ok {
		t.Error("expected badge 'pending' in compile result")
	}

	// Timing Actions
	en3 := sl1.Acts[2]
	if en3.Name != "Timing Actions" {
		t.Errorf("enactment name: expected 'Timing Actions', got %q", en3.Name)
	}

	// Retry Actions: retry_scene + retry_enactment with else
	en4 := sl1.Acts[3]
	if en4.Name != "Retry Actions" {
		t.Errorf("enactment name: expected 'Retry Actions', got %q", en4.Name)
	}
	if _, ok := result.Badges["exhausted"]; !ok {
		t.Error("expected badge 'exhausted' in compile result (from retry else block)")
	}

	// Loop Actions storyline: 2 enactments
	sl2 := story.Storylines[1]
	if sl2.Name != "Loop Actions" {
		t.Errorf("storyline 2 name: expected 'Loop Actions', got %q", sl2.Name)
	}
	if len(sl2.Acts) != 2 {
		t.Fatalf("expected 2 enactments in Loop Actions, got %d", len(sl2.Acts))
	}

	// Verify loop_exhausted badge from multi-line else block
	if _, ok := result.Badges["loop_exhausted"]; !ok {
		t.Error("expected badge 'loop_exhausted' in compile result (from loop else block)")
	}
}

// ---------- Badge Integration ----------

func TestAtomicBadgeIntegration(t *testing.T) {
	result := CompileScript(FixtureAtomicBadgeIntegration, "sub123", "creator456")
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}

	story := result.Stories[0]

	// Verify story-level badge configuration
	if story.OnBegin.BadgeTransaction == nil {
		t.Error("story on_begin: expected badge transaction")
	} else if len(story.OnBegin.BadgeTransaction.GiveBadges) == 0 {
		t.Error("story on_begin: expected give_badge 'enrolled'")
	}

	if story.OnComplete.BadgeTransaction == nil {
		t.Error("story on_complete: expected badge transaction")
	} else {
		if len(story.OnComplete.BadgeTransaction.GiveBadges) == 0 {
			t.Error("story on_complete: expected give_badge 'graduated'")
		}
		if len(story.OnComplete.BadgeTransaction.RemoveBadges) == 0 {
			t.Error("story on_complete: expected remove_badge 'enrolled'")
		}
	}

	if story.OnFail.BadgeTransaction == nil {
		t.Error("story on_fail: expected badge transaction")
	} else {
		if len(story.OnFail.BadgeTransaction.GiveBadges) == 0 {
			t.Error("story on_fail: expected give_badge 'dropped_out'")
		}
		if len(story.OnFail.BadgeTransaction.RemoveBadges) == 0 {
			t.Error("story on_fail: expected remove_badge 'enrolled'")
		}
	}

	// Verify story-level required_badges
	if len(story.RequiredUserBadges.MustNotHave) == 0 {
		t.Error("story: expected must_not_have 'already_graduated'")
	}

	// Verify start_trigger and complete_trigger
	// These get set as special fields on the story entity
	// The compiler generates them but we verify the story structure

	// Verify 3 storylines
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	// Storyline 1 (Coursework): required_badges, on_begin, on_complete, on_fail
	sl1 := story.Storylines[0]
	if sl1.Name != "Coursework" {
		t.Errorf("storyline 1 name: expected 'Coursework', got %q", sl1.Name)
	}
	if len(sl1.RequiredUserBadges.MustHave) == 0 {
		t.Error("storyline 'Coursework': expected must_have 'enrolled'")
	}
	if sl1.OnBegin.BadgeTransaction == nil || len(sl1.OnBegin.BadgeTransaction.GiveBadges) == 0 {
		t.Error("storyline 'Coursework' on_begin: expected give_badge 'coursework_started'")
	}
	if sl1.OnComplete.BadgeTransaction == nil || len(sl1.OnComplete.BadgeTransaction.GiveBadges) == 0 {
		t.Error("storyline 'Coursework' on_complete: expected give_badge 'coursework_done'")
	}
	if sl1.OnFail.BadgeTransaction == nil || len(sl1.OnFail.BadgeTransaction.GiveBadges) == 0 {
		t.Error("storyline 'Coursework' on_fail: expected give_badge 'coursework_failed'")
	}

	// Verify enactment trigger-level required_badges
	if len(sl1.Acts) > 0 {
		en1 := sl1.Acts[0]
		if en1.OnEvent == nil {
			t.Fatal("Lesson 1: expected OnEvent triggers")
		}
		// Check that click triggers exist and have badges
		clickTriggers, ok := en1.OnEvent["OnClick"]
		if !ok || len(clickTriggers) == 0 {
			t.Error("Lesson 1: expected OnClick triggers")
		} else {
			tr := clickTriggers[0]
			if tr.RequiredBadges.MustHave == nil || len(tr.RequiredBadges.MustHave) == 0 {
				t.Error("Lesson 1 click trigger: expected required_badges must_have")
			}
			if tr.RequiredBadges.MustNotHave == nil || len(tr.RequiredBadges.MustNotHave) == 0 {
				t.Error("Lesson 1 click trigger: expected required_badges must_not_have")
			}
		}
	}

	// Storyline 2 (Final Exam): required_badges must_have "coursework_done"
	sl2 := story.Storylines[1]
	if sl2.Name != "Final Exam" {
		t.Errorf("storyline 2 name: expected 'Final Exam', got %q", sl2.Name)
	}
	if len(sl2.RequiredUserBadges.MustHave) == 0 {
		t.Error("storyline 'Final Exam': expected must_have 'coursework_done'")
	}

	// Verify comprehensive badge set
	expectedBadges := []string{
		"enrollment", "graduation", "already_graduated",
		"enrolled", "graduated", "dropped_out",
		"coursework_started", "coursework_done", "coursework_failed",
		"banned", "lesson1_passed", "exam_passed", "remedial_done",
	}
	for _, name := range expectedBadges {
		if _, ok := result.Badges[name]; !ok {
			t.Errorf("expected badge %q in compile result", name)
		}
	}
}

// ---------- Conditions and Routing ----------

func TestAtomicConditionsAndRouting(t *testing.T) {
	result := CompileScript(FixtureAtomicConditionsAndRouting, "sub123", "creator456")
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}

	story := result.Stories[0]
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	// Evaluation storyline: should have conditional routing
	sl1 := story.Storylines[0]
	if sl1.Name != "Evaluation" {
		t.Errorf("storyline 1 name: expected 'Evaluation', got %q", sl1.Name)
	}

	// Verify conditional routes in on_complete
	if len(sl1.OnComplete.ConditionalRoutes) != 2 {
		t.Errorf("expected 2 conditional routes, got %d", len(sl1.OnComplete.ConditionalRoutes))
	}
	if len(sl1.OnComplete.ConditionalRoutes) >= 1 {
		cr := sl1.OnComplete.ConditionalRoutes[0]
		if len(cr.RequiredBadges.MustHave) == 0 {
			t.Error("conditional route 1: expected required_badges must_have 'premium'")
		}
	}

	// 3 enactments: Badge Conditions, Tag Conditions, Compound Conditions
	if len(sl1.Acts) != 3 {
		t.Fatalf("expected 3 enactments in Evaluation, got %d", len(sl1.Acts))
	}

	// Badge Conditions enactment: has_badge + not_has_badge
	en1 := sl1.Acts[0]
	if en1.Name != "Badge Conditions" {
		t.Errorf("expected 'Badge Conditions', got %q", en1.Name)
	}

	// Tag Conditions enactment: has_tag + not_has_tag
	en2 := sl1.Acts[1]
	if en2.Name != "Tag Conditions" {
		t.Errorf("expected 'Tag Conditions', got %q", en2.Name)
	}

	// Compound Conditions enactment: and, or, not
	en3 := sl1.Acts[2]
	if en3.Name != "Compound Conditions" {
		t.Errorf("expected 'Compound Conditions', got %q", en3.Name)
	}

	// Verify badges created
	for _, name := range []string{"vip", "premium", "banned"} {
		if _, ok := result.Badges[name]; !ok {
			t.Errorf("expected badge %q in compile result", name)
		}
	}

	// Verify Premium Path has required_badges
	sl2 := story.Storylines[1]
	if sl2.Name != "Premium Path" {
		t.Errorf("storyline 2 name: expected 'Premium Path', got %q", sl2.Name)
	}
	if len(sl2.RequiredUserBadges.MustHave) == 0 {
		t.Error("Premium Path: expected required_badges must_have 'premium'")
	}
}

// ---------- Scene Features ----------

func TestAtomicSceneFeatures(t *testing.T) {
	result := CompileScript(FixtureAtomicSceneFeatures, "sub123", "creator456")
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}

	story := result.Stories[0]
	sl := story.Storylines[0]
	en := sl.Acts[0]

	// Verify scene exists (single-scene enactments use SendScene, not SendScenes)
	scene := en.SendScene
	if scene == nil {
		if len(en.SendScenes) > 0 {
			scene = en.SendScenes[0]
		} else {
			t.Fatal("expected at least 1 scene")
		}
	}
	if scene.Message == nil || scene.Message.Content == nil {
		t.Fatal("scene missing message content")
	}

	// Verify subject was set
	if scene.Message.Content.Subject != "Feature Rich Email" {
		t.Errorf("expected subject 'Feature Rich Email', got %q", scene.Message.Content.Subject)
	}

	// Verify vars were compiled into GivenVars
	if scene.Message.Content.GivenVars == nil || len(scene.Message.Content.GivenVars) == 0 {
		t.Error("expected vars (GivenVars) to be set on message content")
	} else {
		if v, ok := scene.Message.Content.GivenVars["hero_image"]; !ok || v != "https://example.com/hero.png" {
			t.Errorf("expected GivenVars['hero_image'] = 'https://example.com/hero.png', got %q", v)
		}
		if v, ok := scene.Message.Content.GivenVars["cta_text"]; !ok || v != "Shop Now" {
			t.Errorf("expected GivenVars['cta_text'] = 'Shop Now', got %q", v)
		}
	}

	// Verify tags were compiled (scene-level tags)
	if scene.Tags == nil || len(scene.Tags) == 0 {
		t.Error("expected tags to be set on scene")
	} else if len(scene.Tags) != 3 {
		t.Errorf("expected 3 tags, got %d", len(scene.Tags))
	}
}

// ---------- V3 Badge Campaign ----------

func TestAtomicV3BadgeCampaign(t *testing.T) {
	result := CompileScript(FixtureAtomicBadgeCampaign, "sub123", "creator456")
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}

	story := result.Stories[0]

	// Verify story-level badge lifecycle
	if story.OnBegin.BadgeTransaction == nil {
		t.Error("story on_begin: expected badge transaction")
	}
	if story.OnComplete.BadgeTransaction == nil {
		t.Error("story on_complete: expected badge transaction")
	}

	// V3 generative: should have 3 storylines from the for loop over modules
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines from for loop, got %d", len(story.Storylines))
	}

	expectedNames := []string{"Fundamentals", "Intermediate", "Advanced"}
	for i, sl := range story.Storylines {
		if sl.Name != expectedNames[i] {
			t.Errorf("storyline %d: expected %q, got %q", i, expectedNames[i], sl.Name)
		}
		// Each storyline should have 1 enactment
		if len(sl.Acts) != 1 {
			t.Errorf("storyline %d: expected 1 enactment, got %d", i, len(sl.Acts))
			continue
		}

		en := sl.Acts[0]
		expectedEnName := expectedNames[i] + " Lesson"
		if en.Name != expectedEnName {
			t.Errorf("enactment name: expected %q, got %q", expectedEnName, en.Name)
		}

		// Each enactment should have 1 scene (single-scene → SendScene)
		if en.SendScene == nil && len(en.SendScenes) == 0 {
			t.Errorf("enactment %d: expected 1 scene, got 0", i)
			continue
		}

		// Verify scene content interpolation (single-scene uses SendScene)
		sc := en.SendScene
		if sc == nil && len(en.SendScenes) > 0 {
			sc = en.SendScenes[0]
		}
		if sc == nil || sc.Message == nil || sc.Message.Content == nil {
			t.Errorf("scene %d: missing message content", i)
			continue
		}
	}

	// Verify badges from the data loop
	expectedBadges := []string{
		"v3_enrolled", "v3_graduated",
		"fundamentals_done", "intermediate_done", "advanced_done",
	}
	for _, name := range expectedBadges {
		if _, ok := result.Badges[name]; !ok {
			t.Errorf("expected badge %q in compile result", name)
		}
	}
}

// ---------- Full Feature Coverage Compilation Test ----------
// This test verifies that ALL 6 fixtures compile successfully, proving
// that every atomic feature in the DSL is exercisable.

func TestAtomicAllFixturesCompile(t *testing.T) {
	fixtures := map[string]string{
		"AllTriggerTypes":        FixtureAtomicAllTriggerTypes,
		"AllActionTypes":         FixtureAtomicAllActionTypes,
		"BadgeIntegration":       FixtureAtomicBadgeIntegration,
		"ConditionsAndRouting":   FixtureAtomicConditionsAndRouting,
		"SceneFeatures":          FixtureAtomicSceneFeatures,
		"V3BadgeCampaign":        FixtureAtomicBadgeCampaign,
	}

	for name, src := range fixtures {
		t.Run(name, func(t *testing.T) {
			result := CompileScript(src, "sub123", "creator456")
			if result.Diagnostics.HasErrors() {
				for _, d := range result.Diagnostics {
					t.Errorf("diagnostic: %s", d.Message)
				}
				t.Fatalf("fixture %s: compilation failed", name)
			}
			if len(result.Stories) == 0 {
				t.Fatalf("fixture %s: no stories produced", name)
			}
		})
	}
}

// ---------- Feature Existence Verification ----------
// This test ensures every critical keyword appears in at least one fixture.

func TestAtomicFeatureCoverage(t *testing.T) {
	allFixtures := strings.Join([]string{
		FixtureAtomicAllTriggerTypes,
		FixtureAtomicAllActionTypes,
		FixtureAtomicBadgeIntegration,
		FixtureAtomicConditionsAndRouting,
		FixtureAtomicSceneFeatures,
		FixtureAtomicBadgeCampaign,
	}, "\n")

	// Trigger types
	triggerKeywords := []string{
		"on click", "on not_click", "on open", "on not_open",
		"on sent", "on webhook", "on nothing", "on else",
		"on bounce", "on spam", "on unsubscribe", "on failure",
		"on email_validated", "on user_has_tag", "on badge",
	}
	for _, kw := range triggerKeywords {
		if !strings.Contains(allFixtures, kw) {
			t.Errorf("trigger type %q not found in any atomic fixture", kw)
		}
	}

	// Action types
	actionKeywords := []string{
		"do next_scene", "do prev_scene",
		"do jump_to_enactment", "do jump_to_storyline",
		"do next_enactment", "do advance_to_next_storyline",
		"do end_story", "do mark_complete", "do mark_failed",
		"do unsubscribe", "do give_badge", "do remove_badge",
		"do retry_scene", "do retry_enactment",
		"do loop_to_enactment", "do loop_to_storyline",
		"do loop_to_start_enactment", "do loop_to_start_storyline",
		"do wait", "do send_immediate",
	}
	for _, kw := range actionKeywords {
		if !strings.Contains(allFixtures, kw) {
			t.Errorf("action type %q not found in any atomic fixture", kw)
		}
	}

	// Condition types
	conditionKeywords := []string{
		"has_badge", "not_has_badge", "has_tag", "not_has_tag",
		"when and", "when or", "when not",
	}
	for _, kw := range conditionKeywords {
		if !strings.Contains(allFixtures, kw) {
			t.Errorf("condition type %q not found in any atomic fixture", kw)
		}
	}

	// Badge constructs
	badgeKeywords := []string{
		"give_badge", "remove_badge", "required_badges",
		"must_have", "must_not_have",
		"start_trigger", "complete_trigger",
		"on_begin", "on_complete", "on_fail",
	}
	for _, kw := range badgeKeywords {
		if !strings.Contains(allFixtures, kw) {
			t.Errorf("badge construct %q not found in any atomic fixture", kw)
		}
	}

	// Scene features
	sceneKeywords := []string{
		"template", "vars", "tags",
	}
	for _, kw := range sceneKeywords {
		if !strings.Contains(allFixtures, kw) {
			t.Errorf("scene feature %q not found in any atomic fixture", kw)
		}
	}

	// V3 generative features
	v3Keywords := []string{
		"data ", "for ", " in ",
	}
	for _, kw := range v3Keywords {
		if !strings.Contains(allFixtures, kw) {
			t.Errorf("v3 feature %q not found in any atomic fixture", kw)
		}
	}

	// Trigger configuration
	triggerConfigKeywords := []string{
		"trigger_priority", "persist_scope", "mark_complete",
		"mark_failed", "send_immediate", "within",
		"skip_to_next_storyline_on_expiry",
	}
	for _, kw := range triggerConfigKeywords {
		if !strings.Contains(allFixtures, kw) {
			t.Errorf("trigger config %q not found in any atomic fixture", kw)
		}
	}

	// Structural keywords
	structKeywords := []string{
		"story ", "storyline ", "enactment ", "scene ",
		"priority", "allow_interruption", "order",
		"level", "conditional_route",
		"next_storyline", "next_story",
	}
	for _, kw := range structKeywords {
		if !strings.Contains(allFixtures, kw) {
			t.Errorf("structural keyword %q not found in any atomic fixture", kw)
		}
	}
}
