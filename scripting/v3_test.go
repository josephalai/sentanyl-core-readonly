package scripting

import (
	"strings"
	"testing"
)

// ========== DSL v3 Parser Tests ==========

func TestParserDataBlock(t *testing.T) {
	src := `
data phases = [
	{ name: "A", subject: "Soft Intrigue", link: more_info_a },
	{ name: "B", subject: "Hard Intrigue" }
]

story "Test" {
	storyline "Main" {
		order 1
		enactment "Hook" {
			level 1
			scene "Email" {
				subject "Hello"
				body "Body"
			}
		}
	}
}
`
	result := ParseScript(src)
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("parse failed")
	}

	if len(result.AST.DataBlocks) != 1 {
		t.Fatalf("expected 1 data block, got %d", len(result.AST.DataBlocks))
	}

	db := result.AST.DataBlocks[0]
	if db.Name != "phases" {
		t.Errorf("expected data block name 'phases', got %q", db.Name)
	}
	if len(db.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(db.Items))
	}

	// Check first item
	item0 := db.Items[0]
	if item0.Fields["name"] != "A" {
		t.Errorf("expected item[0].name = 'A', got %q", item0.Fields["name"])
	}
	if item0.Fields["subject"] != "Soft Intrigue" {
		t.Errorf("expected item[0].subject = 'Soft Intrigue', got %q", item0.Fields["subject"])
	}
	if item0.Fields["link"] != "more_info_a" {
		t.Errorf("expected item[0].link = 'more_info_a', got %q", item0.Fields["link"])
	}

	// Check second item
	item1 := db.Items[1]
	if item1.Fields["name"] != "B" {
		t.Errorf("expected item[1].name = 'B', got %q", item1.Fields["name"])
	}
}

func TestParserForLoopInStoryline(t *testing.T) {
	src := `
data phases = [
	{ name: "A" }
]

story "Test" {
	storyline "Main" {
		order 1
		for phase in phases {
			enactment "Enactment ${phase.name}" {
				scene "Email" {
					subject "Subject ${phase.name}"
					body "Body"
				}
			}
		}
	}
}
`
	result := ParseScript(src)
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("parse failed")
	}

	story := result.AST.Stories[0]
	sl := story.Storylines[0]
	if len(sl.ForLoops) != 1 {
		t.Fatalf("expected 1 for loop, got %d", len(sl.ForLoops))
	}

	fl := sl.ForLoops[0]
	if fl.Variable != "phase" {
		t.Errorf("expected variable 'phase', got %q", fl.Variable)
	}
	if fl.DataRef != "phases" {
		t.Errorf("expected data ref 'phases', got %q", fl.DataRef)
	}
	if len(fl.BodyEnactments) != 1 {
		t.Errorf("expected 1 body enactment, got %d", len(fl.BodyEnactments))
	}
}

func TestParserForLoopInStory(t *testing.T) {
	src := `
data tracks = [
	{ name: "Track 1" }
]

story "Test" {
	for track in tracks {
		storyline "${track.name}" {
			order 1
			enactment "Hook" {
				level 1
				scene "Email" { subject "Hello" body "Body" }
			}
		}
	}
}
`
	result := ParseScript(src)
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("parse failed")
	}

	story := result.AST.Stories[0]
	if len(story.ForLoops) != 1 {
		t.Fatalf("expected 1 for loop, got %d", len(story.ForLoops))
	}

	fl := story.ForLoops[0]
	if fl.Variable != "track" {
		t.Errorf("expected variable 'track', got %q", fl.Variable)
	}
	if fl.DataRef != "tracks" {
		t.Errorf("expected data ref 'tracks', got %q", fl.DataRef)
	}
	if len(fl.Body) != 1 {
		t.Errorf("expected 1 storyline body, got %d", len(fl.Body))
	}
}

func TestParserInlineForLoop(t *testing.T) {
	src := `
story "Test" {
	storyline "Main" {
		order 1
		for phase in [
			{ name: "Hook", subject: "Attention" },
			{ name: "Close", subject: "Offer" }
		] {
			enactment "Enactment ${phase.name}" {
				scene "Email" {
					subject "${phase.subject}"
					body "Body"
				}
			}
		}
	}
}
`
	result := ParseScript(src)
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("parse failed")
	}

	sl := result.AST.Stories[0].Storylines[0]
	if len(sl.ForLoops) != 1 {
		t.Fatalf("expected 1 for loop, got %d", len(sl.ForLoops))
	}

	fl := sl.ForLoops[0]
	if len(fl.ObjectItems) != 2 {
		t.Fatalf("expected 2 inline items, got %d", len(fl.ObjectItems))
	}
	if fl.ObjectItems[0].Fields["name"] != "Hook" {
		t.Errorf("expected first item name 'Hook', got %q", fl.ObjectItems[0].Fields["name"])
	}
}

func TestParserForLoopWithUsePattern(t *testing.T) {
	src := `
data phases = [
	{ name: "A", subject: "Intro" }
]

pattern simple(name, subject) {
	enactment name {
		scene "Email" {
			subject subject
			body "Body"
		}
	}
}

story "Test" {
	storyline "Main" {
		order 1
		for phase in phases {
			use pattern simple("${phase.name}", "${phase.subject}")
		}
	}
}
`
	result := ParseScript(src)
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("parse failed")
	}

	sl := result.AST.Stories[0].Storylines[0]
	if len(sl.ForLoops) != 1 {
		t.Fatalf("expected 1 for loop, got %d", len(sl.ForLoops))
	}

	fl := sl.ForLoops[0]
	if len(fl.BodyUseStatements) != 1 {
		t.Fatalf("expected 1 use statement in for body, got %d", len(fl.BodyUseStatements))
	}
	if fl.BodyUseStatements[0].Target != "simple" {
		t.Errorf("expected target 'simple', got %q", fl.BodyUseStatements[0].Target)
	}
}

func TestParserSceneDefaults(t *testing.T) {
	src := `
scene_defaults {
	on not_click {
		within 1d
		do next_scene
	}
}

story "Test" {
	storyline "Main" {
		order 1
		enactment "Hook" {
			level 1
			scene "Email" { subject "Hello" body "Body" }
		}
	}
}
`
	result := ParseScript(src)
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("parse failed")
	}

	if result.AST.SceneDefaults == nil {
		t.Fatal("expected scene_defaults")
	}
	if len(result.AST.SceneDefaults.Triggers) != 1 {
		t.Errorf("expected 1 trigger in scene_defaults, got %d", len(result.AST.SceneDefaults.Triggers))
	}
}

func TestParserEnactmentDefaults(t *testing.T) {
	src := `
policy click_completes(link) {
	on click link {
		trigger_priority 1
		mark_complete true
		within 1m
	}
}

enactment_defaults {
	use policy click_completes(main_link)
}

story "Test" {
	storyline "Main" {
		order 1
		enactment "Hook" {
			level 1
			scene "Email" { subject "Hello" body "Body" }
		}
	}
}
`
	result := ParseScript(src)
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("parse failed")
	}

	if result.AST.EnactmentDefaults == nil {
		t.Fatal("expected enactment_defaults")
	}
	if len(result.AST.EnactmentDefaults.UseStatements) != 1 {
		t.Errorf("expected 1 use statement in enactment_defaults, got %d", len(result.AST.EnactmentDefaults.UseStatements))
	}
}

// ========== DSL v3 Expander Tests ==========

func TestExpanderForLoopEnactments(t *testing.T) {
	result := CompileScript(FixtureDataBlockForLoop, "sub123", "creator456")
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
	// Should have 4 enactments (A, B, C, D) from the for loop over phases
	if len(sl.Acts) != 4 {
		t.Fatalf("expected 4 enactments from for loop, got %d", len(sl.Acts))
	}

	// Verify enactment names
	expectedNames := []string{"Enactment A", "Enactment B", "Enactment C", "Enactment D"}
	for i, en := range sl.Acts {
		if en.Name != expectedNames[i] {
			t.Errorf("enactment %d: expected name %q, got %q", i, expectedNames[i], en.Name)
		}
	}

	// Verify each enactment has 3 scenes (from scenes 1..3)
	for i, en := range sl.Acts {
		if len(en.SendScenes) != 3 {
			t.Errorf("enactment %d (%s): expected 3 scenes, got %d", i, en.Name, len(en.SendScenes))
		}
	}

	// Verify first enactment's first scene has correct subject interpolation
	if len(sl.Acts[0].SendScenes) >= 1 {
		sc := sl.Acts[0].SendScenes[0]
		if sc.Message == nil || sc.Message.Content == nil {
			t.Fatal("first scene missing message content")
		}
		expected := "[Manifesting 101] Soft Intrigue (Email 1/3)"
		if sc.Message.Content.Subject != expected {
			t.Errorf("first scene subject: expected %q, got %q", expected, sc.Message.Content.Subject)
		}
	}

	// Verify triggers have resolved link URLs
	for i, en := range sl.Acts {
		if en.OnEvent == nil {
			t.Errorf("enactment %d: expected OnEvent triggers from policy", i)
			continue
		}
		clickTriggers, ok := en.OnEvent["OnClick"]
		if !ok || len(clickTriggers) == 0 {
			t.Errorf("enactment %d: no OnClick triggers found", i)
			continue
		}
		for _, tr := range clickTriggers {
			if !strings.HasPrefix(tr.UserActionValue, "https://") {
				t.Errorf("enactment %d: expected resolved URL in click trigger, got %q", i, tr.UserActionValue)
			}
		}
	}
}

func TestExpanderStorylineGeneration(t *testing.T) {
	result := CompileScript(FixtureStorylineGeneration, "sub123", "creator456")
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
	// Should have 3 storylines from the for loop over tracks
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	expectedNames := []string{"Manifesting 101", "Advanced Attraction", "Quantum Wealth"}
	for i, sl := range story.Storylines {
		if sl.Name != expectedNames[i] {
			t.Errorf("storyline %d: expected name %q, got %q", i, expectedNames[i], sl.Name)
		}
		// Each storyline should have 2 enactments (intro + sell)
		if len(sl.Acts) != 2 {
			t.Errorf("storyline %d (%s): expected 2 enactments, got %d", i, sl.Name, len(sl.Acts))
		}
	}

	// Verify first storyline's first enactment has correct scene count
	if len(story.Storylines[0].Acts) >= 1 {
		en := story.Storylines[0].Acts[0]
		if len(en.SendScenes) != 2 {
			t.Errorf("first enactment: expected 2 scenes from scenes 1..2, got %d", len(en.SendScenes))
		}
		// Check subject has correct prefix interpolation
		if len(en.SendScenes) >= 1 {
			sc := en.SendScenes[0]
			if sc.Message == nil || sc.Message.Content == nil {
				t.Error("first scene missing message content")
			} else {
				expected := "[M101] Welcome Email 1"
				if sc.Message.Content.Subject != expected {
					t.Errorf("first scene subject: expected %q, got %q", expected, sc.Message.Content.Subject)
				}
			}
		}
	}
}

func TestExpanderInlineDataLoop(t *testing.T) {
	result := CompileScript(FixtureInlineDataLoop, "sub123", "creator456")
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("compilation failed")
	}

	story := result.Stories[0]
	sl := story.Storylines[0]

	// Should have 3 enactments from inline loop
	if len(sl.Acts) != 3 {
		t.Fatalf("expected 3 enactments, got %d", len(sl.Acts))
	}

	expectedNames := []string{"Enactment Hook", "Enactment Value", "Enactment Close"}
	for i, en := range sl.Acts {
		if en.Name != expectedNames[i] {
			t.Errorf("enactment %d: expected %q, got %q", i, expectedNames[i], en.Name)
		}
	}

	// Verify subject interpolation
	if len(sl.Acts[0].SendScenes) >= 1 {
		sc := sl.Acts[0].SendScenes[0]
		if sc.Message != nil && sc.Message.Content != nil {
			if sc.Message.Content.Subject != "Attention Grabber" {
				t.Errorf("expected subject 'Attention Grabber', got %q", sc.Message.Content.Subject)
			}
		}
	}
}

func TestExpanderEnactmentDefaults(t *testing.T) {
	result := CompileScript(FixtureEnactmentDefaults, "sub123", "creator456")
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s", d.Message)
		}
		t.Fatal("compilation failed")
	}

	story := result.Stories[0]
	sl := story.Storylines[0]

	if len(sl.Acts) != 2 {
		t.Fatalf("expected 2 enactments, got %d", len(sl.Acts))
	}

	// Both enactments should have the click_completes policy trigger applied
	for i, en := range sl.Acts {
		if en.OnEvent == nil {
			t.Errorf("enactment %d: expected OnEvent triggers from enactment_defaults", i)
			continue
		}
		clickTriggers, ok := en.OnEvent["OnClick"]
		if !ok || len(clickTriggers) == 0 {
			t.Errorf("enactment %d: expected OnClick triggers from enactment_defaults", i)
			continue
		}
		for _, tr := range clickTriggers {
			if !strings.HasPrefix(tr.UserActionValue, "https://") {
				t.Errorf("enactment %d: expected resolved URL, got %q", i, tr.UserActionValue)
			}
		}
	}
}

func TestExpanderFullGenerativeCampaign(t *testing.T) {
	result := CompileScript(FixtureFullGenerativeCampaign, "sub123", "creator456")
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

	// Should have 3 storylines (from 3 tracks)
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	expectedStorylines := []string{"Manifesting 101", "Advanced Attraction", "Quantum Wealth"}
	for i, sl := range story.Storylines {
		if sl.Name != expectedStorylines[i] {
			t.Errorf("storyline %d: expected %q, got %q", i, expectedStorylines[i], sl.Name)
		}

		// Each storyline should have 4 enactments (A, B, C, D from phases)
		if len(sl.Acts) != 4 {
			t.Errorf("storyline %d (%s): expected 4 enactments, got %d", i, sl.Name, len(sl.Acts))
			continue
		}

		// Each enactment should have 3 scenes
		for j, en := range sl.Acts {
			if len(en.SendScenes) != 3 {
				t.Errorf("storyline %d enactment %d (%s): expected 3 scenes, got %d",
					i, j, en.Name, len(en.SendScenes))
			}
		}
	}

	// Total: 3 storylines × 4 enactments × 3 scenes = 36 scenes
	// Total: 3 storylines × 4 enactments = 12 enactments
	totalEnactments := 0
	totalScenes := 0
	for _, sl := range story.Storylines {
		totalEnactments += len(sl.Acts)
		for _, en := range sl.Acts {
			totalScenes += len(en.SendScenes)
		}
	}
	if totalEnactments != 12 {
		t.Errorf("expected 12 total enactments, got %d", totalEnactments)
	}
	if totalScenes != 36 {
		t.Errorf("expected 36 total scenes, got %d", totalScenes)
	}

	// Verify first storyline, first enactment, first scene has correct interpolation
	firstScene := story.Storylines[0].Acts[0].SendScenes[0]
	if firstScene.Message == nil || firstScene.Message.Content == nil {
		t.Fatal("first scene missing message content")
	}
	expectedSubject := "[M101] Soft Intrigue (Email 1/3)"
	if firstScene.Message.Content.Subject != expectedSubject {
		t.Errorf("first scene subject: expected %q, got %q", expectedSubject, firstScene.Message.Content.Subject)
	}

	// Verify third storyline, last enactment, last scene
	lastSL := story.Storylines[2]
	lastEN := lastSL.Acts[3]
	lastScene := lastEN.SendScenes[2]
	if lastScene.Message == nil || lastScene.Message.Content == nil {
		t.Fatal("last scene missing message content")
	}
	expectedSubject = "[QW] Hard Sell (Email 3/3)"
	if lastScene.Message.Content.Subject != expectedSubject {
		t.Errorf("last scene subject: expected %q, got %q", expectedSubject, lastScene.Message.Content.Subject)
	}

	// Verify sender defaults were applied
	if firstScene.Message.Content.FromEmail != "coach@demo.com" {
		t.Errorf("expected from_email 'coach@demo.com', got %q", firstScene.Message.Content.FromEmail)
	}
}

func TestExpanderDotAccessTriggers(t *testing.T) {
	result := CompileScript(FixtureDotAccessTriggers, "sub123", "creator456")
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s at %d:%d", d.Message, d.Pos.Line, d.Pos.Col)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}

	story := result.Stories[0]
	// 2 workshops → 2 storylines
	if len(story.Storylines) != 2 {
		t.Fatalf("expected 2 storylines, got %d", len(story.Storylines))
	}

	// Verify storyline names
	expectedSLNames := []string{"Wealth Track", "Love Track"}
	for i, sl := range story.Storylines {
		if sl.Name != expectedSLNames[i] {
			t.Errorf("storyline %d: expected name %q, got %q", i, expectedSLNames[i], sl.Name)
		}
	}

	// Each storyline should have 2 enactments (Intrigue + Sell)
	for i, sl := range story.Storylines {
		if len(sl.Acts) != 2 {
			t.Errorf("storyline %d (%s): expected 2 enactments, got %d", i, sl.Name, len(sl.Acts))
			continue
		}
	}

	// Verify triggers have resolved URLs from dot-access (ws.info, ws.buy)
	// First storyline (Wealth): ws.info → w1_info → "https://example.com/wealth/info"
	sl0 := story.Storylines[0]
	en0 := sl0.Acts[0] // Intrigue enactment
	if en0.OnEvent == nil {
		t.Fatal("Wealth Intrigue enactment: expected OnEvent triggers")
	}
	clickTriggers, ok := en0.OnEvent["OnClick"]
	if !ok || len(clickTriggers) == 0 {
		t.Fatal("Wealth Intrigue: expected OnClick triggers")
	}
	expectedURL := "https://example.com/wealth/info"
	if clickTriggers[0].UserActionValue != expectedURL {
		t.Errorf("Wealth Intrigue OnClick URL: expected %q, got %q", expectedURL, clickTriggers[0].UserActionValue)
	}

	// Second storyline (Love): sell enactment click should be ws.buy → w2_buy
	sl1 := story.Storylines[1]
	en1Sell := sl1.Acts[1] // Sell enactment
	if en1Sell.OnEvent == nil {
		t.Fatal("Love Sell enactment: expected OnEvent triggers")
	}
	clickSell, ok := en1Sell.OnEvent["OnClick"]
	if !ok || len(clickSell) == 0 {
		t.Fatal("Love Sell: expected OnClick triggers")
	}
	expectedBuyURL := "https://example.com/love/buy"
	if clickSell[0].UserActionValue != expectedBuyURL {
		t.Errorf("Love Sell OnClick URL: expected %q, got %q", expectedBuyURL, clickSell[0].UserActionValue)
	}
}

func TestHashComments(t *testing.T) {
	src := `
# This is a hash comment
story "Test" {
	# Another comment
	storyline "Main" {
		order 1
		enactment "Welcome" {
			level 1
			scene "Email" {
				subject "Hello" # inline comment
				body "Body"
			}
		}
	}
}
`
	result := ParseScript(src)
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			if d.Level == DiagError {
				t.Errorf("parse error: %s at %d:%d", d.Message, d.Pos.Line, d.Pos.Col)
			}
		}
		t.Fatal("parsing failed with hash comments")
	}

	// Verify the script structure parsed correctly
	if len(result.AST.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.AST.Stories))
	}
}

func TestExpanderUnknownDataBlock(t *testing.T) {
	src := `
story "Test" {
	storyline "Main" {
		order 1
		for phase in unknown_data {
			enactment "Test" {
				scene "Email" { subject "Hello" body "Body" }
			}
		}
	}
}
`
	result := ValidateScript(src)
	hasError := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && strings.Contains(d.Message, "unknown data block") {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected 'unknown data block' error")
	}
}

func TestSubstituteParamsWithDotAccess(t *testing.T) {
	tests := []struct {
		input    string
		params   map[string]string
		expected string
	}{
		{
			"Hello ${phase.name}",
			map[string]string{"phase.name": "World"},
			"Hello World",
		},
		{
			"${track.prefix} ${phase.subject}",
			map[string]string{"track.prefix": "[M101]", "phase.subject": "Intro"},
			"[M101] Intro",
		},
		{
			"phase.link",
			map[string]string{"phase.link": "https://example.com"},
			"https://example.com",
		},
		{
			"No vars",
			map[string]string{"phase.name": "World"},
			"No vars",
		},
	}

	for _, tt := range tests {
		got := substituteParamsWithDotAccess(tt.input, tt.params)
		if got != tt.expected {
			t.Errorf("substituteParamsWithDotAccess(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDataBlockIntegerValues(t *testing.T) {
	src := `
data items = [
	{ name: "Alpha", order: 1 },
	{ name: "Beta",  order: 2 }
]

story "Test" {
	for item in items {
		storyline "${item.name} Track" {
			order item.order
			enactment "Hook" {
				level 1
				scene "Email" {
					subject "Hello ${item.name}"
					body "Body"
				}
			}
		}
	}
}
`
	result := CompileScript(src, "sub123", "creator456")
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s at %d:%d", d.Message, d.Pos.Line, d.Pos.Col)
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

	// Verify order was resolved from data block integer values
	sl0 := story.Storylines[0]
	sl1 := story.Storylines[1]
	if sl0.Name != "Alpha Track" {
		t.Errorf("expected storyline 0 name 'Alpha Track', got %q", sl0.Name)
	}
	if sl1.Name != "Beta Track" {
		t.Errorf("expected storyline 1 name 'Beta Track', got %q", sl1.Name)
	}
	if sl0.NaturalOrder != 1 {
		t.Errorf("expected storyline 0 order 1, got %d", sl0.NaturalOrder)
	}
	if sl1.NaturalOrder != 2 {
		t.Errorf("expected storyline 1 order 2, got %d", sl1.NaturalOrder)
	}
}
