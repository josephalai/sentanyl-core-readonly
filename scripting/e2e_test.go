package scripting

import (
	"testing"

	"gopkg.in/mgo.v2/bson"
)

// End-to-end tests that verify complete script-to-entity-graph compilation.

func TestE2ESimpleOneStoryline(t *testing.T) {
	ResetIDCounter()
	src := FixtureSimpleOneStoryline
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	story := result.Stories[0]
	if story.Name != "Simple Welcome" {
		t.Errorf("expected name 'Simple Welcome', got %q", story.Name)
	}
	if len(story.Storylines) != 1 {
		t.Fatalf("expected 1 storyline, got %d", len(story.Storylines))
	}
	if len(story.Storylines[0].Acts) != 1 {
		t.Fatalf("expected 1 enactment, got %d", len(story.Storylines[0].Acts))
	}
}

func TestE2EMultiStoryline(t *testing.T) {
	ResetIDCounter()
	src := FixtureMultiStoryline
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	story := result.Stories[0]
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}
}

func TestE2EMultiEnactment(t *testing.T) {
	ResetIDCounter()
	src := FixtureMultiEnactment
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	sl := result.Stories[0].Storylines[0]
	if len(sl.Acts) != 3 {
		t.Fatalf("expected 3 enactments, got %d", len(sl.Acts))
	}
}

func TestE2EMultiScene(t *testing.T) {
	ResetIDCounter()
	src := FixtureMultiScene
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	if len(en.SendScenes) != 3 {
		t.Fatalf("expected 3 scenes, got %d", len(en.SendScenes))
	}
}

func TestE2EConditionalBadgeRouting(t *testing.T) {
	ResetIDCounter()
	src := FixtureConditionalBadgeRouting
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	story := result.Stories[0]
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}
	// First storyline should have conditional routes
	sl := story.Storylines[0]
	if len(sl.OnComplete.ConditionalRoutes) != 2 {
		t.Error("expected 2 conditional routes on first storyline on_complete")
	}
}

func TestE2EClickBranching(t *testing.T) {
	ResetIDCounter()
	src := FixtureClickBranching
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	if len(en.OnEvent) < 2 {
		t.Errorf("expected at least 2 trigger types, got %d", len(en.OnEvent))
	}
}

func TestE2EOpenBranching(t *testing.T) {
	ResetIDCounter()
	src := FixtureOpenBranching
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	en := result.Stories[0].Storylines[0].Acts[0]
	if _, ok := en.OnEvent["OnOpen"]; !ok {
		t.Error("expected OnOpen triggers")
	}
	if _, ok := en.OnEvent["NotOpen"]; !ok {
		t.Error("expected NotOpen triggers")
	}
}

func TestE2EBoundedRetry(t *testing.T) {
	ResetIDCounter()
	src := FixtureBoundedRetry
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	// Should compile without errors and contain retry actions
	en := result.Stories[0].Storylines[0].Acts[0]
	if len(en.OnEvent) == 0 {
		t.Error("expected triggers with retry logic")
	}
}

func TestE2ELoopToPriorEnactment(t *testing.T) {
	ResetIDCounter()
	src := FixtureLoopToPriorEnactment
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	// Should compile with 2 enactments and a loop trigger
	sl := result.Stories[0].Storylines[0]
	if len(sl.Acts) != 2 {
		t.Fatalf("expected 2 enactments, got %d", len(sl.Acts))
	}
}

func TestE2EFailureFallback(t *testing.T) {
	ResetIDCounter()
	src := FixtureFailureFallback
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	story := result.Stories[0]
	if story.OnFail.BadgeTransaction == nil {
		t.Error("expected on_fail badge transaction")
	}
}

func TestE2ECompletionPath(t *testing.T) {
	ResetIDCounter()
	src := FixtureCompletionPath
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	story := result.Stories[0]
	if story.OnComplete.BadgeTransaction == nil {
		t.Error("expected on_complete badge transaction")
	}
}

func TestE2EFullCampaign(t *testing.T) {
	ResetIDCounter()
	src := FixtureFullCampaign
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}
	story := result.Stories[0]
	if len(story.Storylines) < 2 {
		t.Errorf("expected at least 2 storylines, got %d", len(story.Storylines))
	}
	// Verify all entity IDs are valid
	for _, sl := range story.Storylines {
		if !sl.Id.Valid() {
			t.Error("invalid storyline ID")
		}
		for _, en := range sl.Acts {
			if !en.Id.Valid() {
				t.Error("invalid enactment ID")
			}
			if en.SendScene != nil && !en.SendScene.Id.Valid() {
				t.Error("invalid scene ID")
			}
		}
	}
	// Verify badges were created
	if len(result.Badges) == 0 {
		t.Error("expected badges to be created")
	}
}

func TestE2EMultiStorySequence(t *testing.T) {
	ResetIDCounter()
	src := FixtureMultiStorySequence
	result := CompileScript(src, "sub_e2e_mss", bson.NewObjectId())

	// No compile errors
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s", d)
		}
	}

	// 2 stories: "Buy Manifesting Workshops" + "Buy Coaching Products"
	if len(result.Stories) != 2 {
		t.Fatalf("expected 2 stories, got %d", len(result.Stories))
	}

	// Story A: Buy Manifesting Workshops
	storyA := result.Stories[0]
	if storyA.Name != "Buy Manifesting Workshops" {
		t.Errorf("expected story name 'Buy Manifesting Workshops', got %q", storyA.Name)
	}
	if storyA.Priority != 1 {
		t.Errorf("expected story A priority 1, got %d", storyA.Priority)
	}
	// 3 storylines (Wealth, Love, Health)
	if len(storyA.Storylines) != 3 {
		t.Fatalf("expected 3 storylines in story A, got %d", len(storyA.Storylines))
	}
	// Each storyline should have 4 enactments
	for i, sl := range storyA.Storylines {
		if len(sl.Acts) != 4 {
			t.Errorf("storyline %d: expected 4 enactments, got %d", i, len(sl.Acts))
		}
		// Each of the 4 enactments should have 3 scenes
		for j, en := range sl.Acts {
			if len(en.SendScenes) != 3 {
				t.Errorf("storyline %d, enactment %d: expected 3 scenes, got %d", i, j, len(en.SendScenes))
			}
			// Each enactment should have click + not_click triggers
			if len(en.OnEvent) < 2 {
				t.Errorf("storyline %d, enactment %d: expected at least 2 trigger types, got %d", i, j, len(en.OnEvent))
			}
		}
	}

	// Story B: Buy Coaching Products
	storyB := result.Stories[1]
	if storyB.Name != "Buy Coaching Products" {
		t.Errorf("expected story name 'Buy Coaching Products', got %q", storyB.Name)
	}
	if storyB.Priority != 2 {
		t.Errorf("expected story B priority 2, got %d", storyB.Priority)
	}
	// 2 storylines (Executive, Leadership)
	if len(storyB.Storylines) != 2 {
		t.Fatalf("expected 2 storylines in story B, got %d", len(storyB.Storylines))
	}
	// Each coaching storyline has 1 enactment with 2 scenes
	for i, sl := range storyB.Storylines {
		if len(sl.Acts) != 1 {
			t.Errorf("coaching storyline %d: expected 1 enactment, got %d", i, len(sl.Acts))
		}
		en := sl.Acts[0]
		if len(en.SendScenes) != 2 {
			t.Errorf("coaching storyline %d: expected 2 scenes, got %d", i, len(en.SendScenes))
		}
	}

	// Verify badges were created (5 workshop purchase badges + 2 coaching purchase badges)
	if len(result.Badges) == 0 {
		t.Error("expected badges to be created")
	}

	// Verify all entity IDs are valid
	for _, story := range result.Stories {
		if !story.Id.Valid() {
			t.Error("invalid story ID")
		}
		for _, sl := range story.Storylines {
			if !sl.Id.Valid() {
				t.Error("invalid storyline ID")
			}
			for _, en := range sl.Acts {
				if !en.Id.Valid() {
					t.Error("invalid enactment ID")
				}
				for _, sc := range en.SendScenes {
					if !sc.Id.Valid() {
						t.Error("invalid scene ID")
					}
				}
			}
		}
	}
}
