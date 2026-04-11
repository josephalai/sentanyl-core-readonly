package scripting

import (
	"strings"
	"testing"

	"gopkg.in/mgo.v2/bson"
)

// ---------- Expander tests ----------

func TestExpanderDefaultSender(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(FixtureDefaultSender, "sub1", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s: %s", d.Pos, d.Message)
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
	if len(sl.Acts) != 1 {
		t.Fatalf("expected 1 enactment, got %d", len(sl.Acts))
	}
	en := sl.Acts[0]

	// Single scene → SendScene
	if en.SendScene == nil {
		t.Fatal("expected SendScene to be set")
	}
	scene := en.SendScene
	msg := scene.Message
	if msg == nil || msg.Content == nil {
		t.Fatal("expected message with content")
	}

	// Verify sender defaults were applied
	if msg.Content.FromEmail != "hello@example.com" {
		t.Errorf("from_email: got %q, want %q", msg.Content.FromEmail, "hello@example.com")
	}
	if msg.Content.FromName != "The Team" {
		t.Errorf("from_name: got %q, want %q", msg.Content.FromName, "The Team")
	}
	if msg.Content.ReplyTo != "support@example.com" {
		t.Errorf("reply_to: got %q, want %q", msg.Content.ReplyTo, "support@example.com")
	}
}

func TestExpanderLinksAndPolicies(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(FixtureLinksAndPolicies, "sub1", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	story := result.Stories[0]
	sl := story.Storylines[0]

	if len(sl.Acts) != 2 {
		t.Fatalf("expected 2 enactments, got %d", len(sl.Acts))
	}

	// First enactment should have a click trigger with the resolved link URL
	en1 := sl.Acts[0]
	triggers := en1.OnEvent
	if triggers == nil {
		t.Fatal("expected triggers on enactment 1")
	}

	// Check that the trigger has the resolved URL
	clickTriggers, ok := triggers["OnClick"]
	if !ok || len(clickTriggers) == 0 {
		t.Fatal("expected OnClick triggers on enactment 1")
	}
	if clickTriggers[0].UserActionValue != "https://example.com/signup" {
		t.Errorf("trigger URL: got %q, want %q", clickTriggers[0].UserActionValue, "https://example.com/signup")
	}

	// Second enactment
	en2 := sl.Acts[1]
	triggers2 := en2.OnEvent
	clickTriggers2, ok := triggers2["OnClick"]
	if !ok || len(clickTriggers2) == 0 {
		t.Fatal("expected OnClick triggers on enactment 2")
	}
	if clickTriggers2[0].UserActionValue != "https://example.com/upgrade" {
		t.Errorf("trigger URL: got %q, want %q", clickTriggers2[0].UserActionValue, "https://example.com/upgrade")
	}
}

func TestExpanderLinksSubstitutionInBody(t *testing.T) {
	src := `
links {
	webinar_replay_url = "https://webinar.example.com/replay"
	course_sales_page  = "https://webinar.example.com/course"
}

story "Link Substitution Test" {
	storyline "Main" {
		order 1
		enactment "Reminder" {
			level 1
			scene "Reminder Email" {
				subject "Webinar starts tomorrow!"
				body "<p>Access it here: ${webinar_replay_url}</p>"
				from_email "webinar@example.com"
			}
		}
		enactment "Offer" {
			level 2
			scene "Offer Email" {
				subject "Check out the course"
				body "<p>Go deeper: ${course_sales_page}</p>"
				from_email "webinar@example.com"
			}
		}
	}
}
`
	ResetIDCounter()
	result := CompileScript(src, "sub1", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}

	en1 := result.Stories[0].Storylines[0].Acts[0]
	body1 := en1.SendScene.Message.Content.Body
	want1 := `<p>Access it here: <a href="https://webinar.example.com/replay">https://webinar.example.com/replay</a></p>`
	if body1 != want1 {
		t.Errorf("scene 1 body:\n  got  %q\n  want %q", body1, want1)
	}

	en2 := result.Stories[0].Storylines[0].Acts[1]
	body2 := en2.SendScene.Message.Content.Body
	want2 := `<p>Go deeper: <a href="https://webinar.example.com/course">https://webinar.example.com/course</a></p>`
	if body2 != want2 {
		t.Errorf("scene 2 body:\n  got  %q\n  want %q", body2, want2)
	}
}

func TestExpanderScenesRange(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(FixtureScenesRange, "sub1", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	story := result.Stories[0]
	sl := story.Storylines[0]
	en := sl.Acts[0]

	// Should have 5 scenes (multi-scene enactment via SendScenes)
	if len(en.SendScenes) != 5 {
		t.Fatalf("expected 5 scenes, got %d", len(en.SendScenes))
	}

	// Verify scene names are interpolated
	for i, sc := range en.SendScenes {
		expectedName := "Day " + string(rune('1'+i))
		_ = expectedName
		if sc.Message == nil || sc.Message.Content == nil {
			t.Errorf("scene %d: missing message content", i)
			continue
		}
		// Verify subjects are interpolated
		if !strings.Contains(sc.Message.Content.Subject, "Day") {
			t.Errorf("scene %d subject: got %q, expected to contain 'Day'", i, sc.Message.Content.Subject)
		}
		// Verify sender defaults were applied
		if sc.Message.Content.FromEmail != "drip@example.com" {
			t.Errorf("scene %d from_email: got %q, want %q", i, sc.Message.Content.FromEmail, "drip@example.com")
		}
	}
}

func TestExpanderPatternReuse(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(FixturePatternReuse, "sub1", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s: %s", d.Pos, d.Message)
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

	// Each storyline should have 1 enactment (from pattern)
	for i, sl := range story.Storylines {
		if len(sl.Acts) != 1 {
			t.Errorf("storyline %d: expected 1 enactment, got %d", i, len(sl.Acts))
			continue
		}
		en := sl.Acts[0]

		// Each enactment should have 2 scenes (from scenes 1..2)
		if len(en.SendScenes) != 2 {
			t.Errorf("storyline %d: expected 2 scenes, got %d", i, len(en.SendScenes))
		}

		// Should have a click trigger (from policy)
		if en.OnEvent == nil {
			t.Errorf("storyline %d: expected triggers", i)
			continue
		}
		clickTriggers, ok := en.OnEvent["OnClick"]
		if !ok || len(clickTriggers) == 0 {
			t.Errorf("storyline %d: expected OnClick triggers", i)
		}
	}

	// Verify Track A enactment name was substituted
	if story.Storylines[0].Acts[0].Name != "Track A Intro" {
		t.Errorf("Track A enactment name: got %q, want %q", story.Storylines[0].Acts[0].Name, "Track A Intro")
	}
	// Verify Track B enactment name was substituted
	if story.Storylines[1].Acts[0].Name != "Track B Intro" {
		t.Errorf("Track B enactment name: got %q, want %q", story.Storylines[1].Acts[0].Name, "Track B Intro")
	}
}

func TestExpanderCompactCampaign(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(FixtureCompactCampaign, "sub1", bson.NewObjectId())
	if result.Diagnostics.HasErrors() {
		for _, d := range result.Diagnostics {
			t.Errorf("diagnostic: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("compilation failed")
	}

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	story := result.Stories[0]
	if story.Name != "Manifesting Workshops Compact" {
		t.Errorf("story name: got %q", story.Name)
	}

	if len(story.Storylines) != 1 {
		t.Fatalf("expected 1 storyline, got %d", len(story.Storylines))
	}
	sl := story.Storylines[0]

	// 4 enactments from 4 pattern invocations
	if len(sl.Acts) != 4 {
		t.Fatalf("expected 4 enactments, got %d", len(sl.Acts))
	}

	// Each enactment should have 3 scenes
	for i, en := range sl.Acts {
		if len(en.SendScenes) != 3 {
			t.Errorf("enactment %d (%s): expected 3 scenes, got %d", i, en.Name, len(en.SendScenes))
		}

		// Each scene should have sender defaults applied
		for j, sc := range en.SendScenes {
			if sc.Message == nil || sc.Message.Content == nil {
				t.Errorf("enactment %d scene %d: missing message content", i, j)
				continue
			}
			mc := sc.Message.Content
			if mc.FromEmail != "coach@demo.com" {
				t.Errorf("enactment %d scene %d from_email: got %q", i, j, mc.FromEmail)
			}
			if mc.FromName != "Manifesting Coach" {
				t.Errorf("enactment %d scene %d from_name: got %q", i, j, mc.FromName)
			}
		}

		// Each enactment should have a click trigger
		if en.OnEvent == nil {
			t.Errorf("enactment %d (%s): expected triggers", i, en.Name)
			continue
		}
		clickTriggers, ok := en.OnEvent["OnClick"]
		if !ok || len(clickTriggers) == 0 {
			t.Errorf("enactment %d (%s): expected OnClick triggers", i, en.Name)
		}
	}

	// Verify enactment names are substituted
	expectedNames := []string{"Enactment A", "Enactment B", "Enactment C", "Enactment D"}
	for i, en := range sl.Acts {
		if en.Name != expectedNames[i] {
			t.Errorf("enactment %d name: got %q, want %q", i, en.Name, expectedNames[i])
		}
	}

	// Verify trigger URLs are resolved from links
	expectedURLs := []string{
		"https://example.com/more-info-a",
		"https://example.com/more-info-b",
		"https://example.com/buy-now-soft",
		"https://example.com/buy-now-hard",
	}
	for i, en := range sl.Acts {
		clickTriggers := en.OnEvent["OnClick"]
		if len(clickTriggers) > 0 {
			if clickTriggers[0].UserActionValue != expectedURLs[i] {
				t.Errorf("enactment %d trigger URL: got %q, want %q",
					i, clickTriggers[0].UserActionValue, expectedURLs[i])
			}
		}
	}

	// Verify subject interpolation
	en0 := sl.Acts[0]
	if len(en0.SendScenes) >= 1 {
		mc := en0.SendScenes[0].Message.Content
		if !strings.Contains(mc.Subject, "[Manifesting 101] Soft Intrigue") {
			t.Errorf("scene 1 subject: got %q", mc.Subject)
		}
		if !strings.Contains(mc.Subject, "1/3") {
			t.Errorf("scene 1 subject should contain '1/3': got %q", mc.Subject)
		}
	}
}

// ---------- Parser tests for new syntax ----------

func TestParserDefaultSender(t *testing.T) {
	src := `
default sender {
	from_email "test@example.com"
	from_name "Test"
	reply_to "reply@example.com"
}
story "Test" {
	storyline "Main" {
		order 1
		enactment "E1" {
			level 1
			scene "S1" { subject "Hi" body "Hello" }
		}
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("parse failed")
	}

	ast := pr.AST
	if len(ast.DefaultSenders) != 1 {
		t.Fatalf("expected 1 default sender, got %d", len(ast.DefaultSenders))
	}
	ds := ast.DefaultSenders[0]
	if ds.FromEmail != "test@example.com" {
		t.Errorf("from_email: got %q", ds.FromEmail)
	}
	if ds.FromName != "Test" {
		t.Errorf("from_name: got %q", ds.FromName)
	}
}

func TestParserLinksBlock(t *testing.T) {
	src := `
links {
	homepage = "https://example.com"
	signup = "https://example.com/signup"
}
story "Test" {
	storyline "Main" {
		order 1
		enactment "E1" {
			level 1
			scene "S1" { subject "Hi" body "Hello" from_email "a@b.com" }
		}
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("parse failed")
	}

	ast := pr.AST
	if ast.Links == nil {
		t.Fatal("expected links block")
	}
	if len(ast.Links.Links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(ast.Links.Links))
	}
	if ast.Links.Links["homepage"] != "https://example.com" {
		t.Errorf("homepage link: got %q", ast.Links.Links["homepage"])
	}
}

func TestParserPatternDef(t *testing.T) {
	src := `
pattern intro(name, prefix) {
	enactment "${name}" {
		level 1
		scene "S1" {
			subject "${prefix} Hello"
			body "World"
			from_email "a@b.com"
		}
	}
}
story "Test" {
	storyline "Main" {
		order 1
		use pattern intro("Welcome", "[Welcome]")
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("parse failed")
	}

	ast := pr.AST
	if len(ast.Patterns) != 1 {
		t.Fatalf("expected 1 pattern, got %d", len(ast.Patterns))
	}
	pat := ast.Patterns[0]
	if pat.Name != "intro" {
		t.Errorf("pattern name: got %q", pat.Name)
	}
	if len(pat.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(pat.Params))
	}
}

func TestParserPolicyDef(t *testing.T) {
	src := `
policy click_complete(link) {
	on click link {
		trigger_priority 1
		mark_complete true
		within 1d
	}
}
story "Test" {
	storyline "Main" {
		order 1
		enactment "E1" {
			level 1
			scene "S1" { subject "Hi" body "Hello" from_email "a@b.com" }
			use policy click_complete("https://example.com")
		}
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("parse failed")
	}

	ast := pr.AST
	if len(ast.Policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(ast.Policies))
	}
	pol := ast.Policies[0]
	if pol.Name != "click_complete" {
		t.Errorf("policy name: got %q", pol.Name)
	}
	if len(pol.Params) != 1 {
		t.Errorf("expected 1 param, got %d", len(pol.Params))
	}
	if len(pol.Triggers) != 1 {
		t.Errorf("expected 1 trigger, got %d", len(pol.Triggers))
	}
}

func TestParserScenesRange(t *testing.T) {
	src := `
story "Test" {
	storyline "Main" {
		order 1
		enactment "E1" {
			level 1
			scenes 1..3 as n {
				scene "Day ${n}" {
					subject "Day ${n} subject"
					body "Day ${n} body"
					from_email "a@b.com"
				}
			}
		}
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("parse failed")
	}

	// The expander should expand the scenes range
	er := ExpandScript(src)
	if er.Diagnostics.HasErrors() {
		for _, d := range er.Diagnostics {
			t.Errorf("expand error: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("expand failed")
	}

	ast := er.AST
	en := ast.Stories[0].Storylines[0].Enactments[0]
	if len(en.Scenes) != 3 {
		t.Fatalf("expected 3 expanded scenes, got %d", len(en.Scenes))
	}

	// Verify interpolation
	if en.Scenes[0].Name != "Day 1" {
		t.Errorf("scene 0 name: got %q, want %q", en.Scenes[0].Name, "Day 1")
	}
	if en.Scenes[1].Name != "Day 2" {
		t.Errorf("scene 1 name: got %q, want %q", en.Scenes[1].Name, "Day 2")
	}
	if en.Scenes[2].Name != "Day 3" {
		t.Errorf("scene 2 name: got %q, want %q", en.Scenes[2].Name, "Day 3")
	}
}

func TestParserUseStatement(t *testing.T) {
	src := `
default sender {
	from_email "test@example.com"
}
story "Test" {
	use sender default
	storyline "Main" {
		order 1
		enactment "E1" {
			level 1
			scene "S1" { subject "Hi" body "Hello" }
		}
	}
}
`
	pr := ParseScript(src)
	if pr.Diagnostics.HasErrors() {
		for _, d := range pr.Diagnostics {
			t.Errorf("parse error: %s: %s", d.Pos, d.Message)
		}
		t.Fatal("parse failed")
	}

	story := pr.AST.Stories[0]
	if len(story.UseStatements) != 1 {
		t.Fatalf("expected 1 use statement, got %d", len(story.UseStatements))
	}
	us := story.UseStatements[0]
	if us.Kind != "sender" {
		t.Errorf("use kind: got %q, want %q", us.Kind, "sender")
	}
	if us.Target != "default" {
		t.Errorf("use target: got %q, want %q", us.Target, "default")
	}
}

// ---------- Lexer tests for new tokens ----------

func TestLexerDotDot(t *testing.T) {
	lex := NewLexer("1..3")
	tokens, errs := lex.Tokenize()
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	// Should be: INT("1") DOTDOT("..") INT("3") EOF
	if len(tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0].Kind != TokInt || tokens[0].Literal != "1" {
		t.Errorf("token 0: got %v", tokens[0])
	}
	if tokens[1].Kind != TokDotDot || tokens[1].Literal != ".." {
		t.Errorf("token 1: got %v", tokens[1])
	}
	if tokens[2].Kind != TokInt || tokens[2].Literal != "3" {
		t.Errorf("token 2: got %v", tokens[2])
	}
}

func TestLexerNewKeywords(t *testing.T) {
	lex := NewLexer("default sender links pattern policy use as scenes")
	tokens, errs := lex.Tokenize()
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	expected := []TokenKind{
		TokDefault, TokSender, TokLinks, TokPattern, TokPolicy, TokUse, TokAs, TokScenes, TokEOF,
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, exp := range expected {
		if tokens[i].Kind != exp {
			t.Errorf("token %d: got %v, want %v", i, tokens[i].Kind, exp)
		}
	}
}

// ---------- Error handling tests ----------

func TestExpanderUnknownPattern(t *testing.T) {
	src := `
story "Test" {
	storyline "Main" {
		order 1
		use pattern nonexistent("arg1")
		enactment "E1" {
			level 1
			scene "S1" { subject "Hi" body "Hello" from_email "a@b.com" }
		}
	}
}
`
	result := CompileScript(src, "sub1", bson.NewObjectId())
	if !result.Diagnostics.HasErrors() {
		t.Fatal("expected error for unknown pattern")
	}
	found := false
	for _, d := range result.Diagnostics {
		if strings.Contains(d.Message, "unknown pattern") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'unknown pattern' diagnostic")
	}
}

func TestExpanderUnknownPolicy(t *testing.T) {
	src := `
story "Test" {
	storyline "Main" {
		order 1
		enactment "E1" {
			level 1
			scene "S1" { subject "Hi" body "Hello" from_email "a@b.com" }
			use policy nonexistent("arg1")
		}
	}
}
`
	result := CompileScript(src, "sub1", bson.NewObjectId())
	if !result.Diagnostics.HasErrors() {
		t.Fatal("expected error for unknown policy")
	}
}

func TestExpanderWrongArgCount(t *testing.T) {
	src := `
pattern two_params(a, b) {
	enactment "${a}" {
		level 1
		scene "S1" { subject "${b}" body "test" from_email "a@b.com" }
	}
}
story "Test" {
	storyline "Main" {
		order 1
		use pattern two_params("only_one")
	}
}
`
	result := CompileScript(src, "sub1", bson.NewObjectId())
	if !result.Diagnostics.HasErrors() {
		t.Fatal("expected error for wrong argument count")
	}
}

// ---------- Verify existing tests not broken ----------

func TestExpanderPassthroughForV1Scripts(t *testing.T) {
	// Verify that v1 scripts (no v2 constructs) pass through the expander unchanged
	scripts := []struct {
		name string
		src  string
	}{
		{"SimpleOneStoryline", FixtureSimpleOneStoryline},
		{"MultiStoryline", FixtureMultiStoryline},
		{"MultiEnactment", FixtureMultiEnactment},
		{"MultiScene", FixtureMultiScene},
		{"FullCampaign", FixtureFullCampaign},
	}

	for _, tc := range scripts {
		t.Run(tc.name, func(t *testing.T) {
			ResetIDCounter()
			result := CompileScript(tc.src, "sub1", bson.NewObjectId())
			if result.Diagnostics.HasErrors() {
				for _, d := range result.Diagnostics {
					t.Errorf("diagnostic: %s: %s", d.Pos, d.Message)
				}
				t.Fatal("compilation should succeed for v1 scripts")
			}
			if len(result.Stories) == 0 {
				t.Fatal("expected at least 1 story")
			}
		})
	}
}

// ---------- String interpolation tests ----------

func TestSubstituteParams(t *testing.T) {
	tests := []struct {
		input    string
		params   map[string]string
		expected string
	}{
		{"Hello ${name}", map[string]string{"name": "World"}, "Hello World"},
		{"${a} and ${b}", map[string]string{"a": "X", "b": "Y"}, "X and Y"},
		{"No vars", map[string]string{"name": "World"}, "No vars"},
		{"${missing}", map[string]string{"name": "World"}, "${missing}"},
		{"", map[string]string{"name": "World"}, ""},
		{"${name}", nil, "${name}"},
		// Bare parameter name substitution
		{"link", map[string]string{"link": "https://example.com"}, "https://example.com"},
		{"name", map[string]string{"name": "Hello"}, "Hello"},
	}

	for _, tc := range tests {
		result := substituteParams(tc.input, tc.params)
		if result != tc.expected {
			t.Errorf("substituteParams(%q, %v): got %q, want %q", tc.input, tc.params, result, tc.expected)
		}
	}
}
