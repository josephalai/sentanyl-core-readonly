package scripting

import (
	"strings"
	"testing"
)

// ---------- Validation Tests ----------

func TestValidatorDuplicateStoryName(t *testing.T) {
	src := `
story "Campaign A" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
story "Campaign A" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "duplicate story name") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'duplicate story name' error")
	}
}

func TestValidatorDuplicateStorylineName(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "duplicate storyline name") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'duplicate storyline name' error")
	}
}

func TestValidatorDuplicateEnactmentName(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "duplicate enactment name") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'duplicate enactment name' error")
	}
}

func TestValidatorUnknownEnactmentTarget(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on click "link" {
				do jump_to_enactment "NonExistent"
			}
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "unknown enactment") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'unknown enactment' error")
	}
}

func TestValidatorUnknownStorylineTarget(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on click "link" {
				do jump_to_storyline "NonExistent"
			}
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "unknown storyline") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'unknown storyline' error")
	}
}

func TestValidatorEmptyStoryline(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "must have at least one enactment") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'must have at least one enactment' error")
	}
}

func TestValidatorEmptyEnactment(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		enactment "Hook" {
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "must have at least one scene") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'must have at least one scene' error")
	}
}

func TestValidatorDuplicateLevel(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		enactment "Hook" {
			level 1
			scene "Email" { subject "Test" body "Body" }
		}
		enactment "Offer" {
			level 1
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "duplicate level") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'duplicate level' error")
	}
}

func TestValidatorDuplicateOrder(t *testing.T) {
	src := `
story "Campaign" {
	storyline "SL1" {
		order 1
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
	storyline "SL2" {
		order 1
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "duplicate order") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'duplicate order' error")
	}
}

func TestValidatorInvalidPersistScope(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on click "link" {
				persist_scope "invalid_scope"
				do mark_complete
			}
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "invalid persist_scope") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'invalid persist_scope' error")
	}
}

func TestValidatorMarkCompleteAndFailed(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on click "link" {
				mark_complete true
				mark_failed true
				do next_scene
			}
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "cannot both mark_complete and mark_failed") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'cannot both mark_complete and mark_failed' error")
	}
}

func TestValidatorNegativeTriggerNoWithin(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on not_open {
				do next_scene
			}
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagWarning && containsStr(d.Message, "no 'within' duration") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about missing 'within' for negative trigger")
	}
}

func TestValidatorUnboundedRetry(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
			on not_open {
				within 1d
				do retry_scene
			}
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagWarning && containsStr(d.Message, "no up_to bound") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about unbounded retry")
	}
}

func TestValidatorConditionalRouteUnknownStoryline(t *testing.T) {
	src := `
story "Campaign" {
	storyline "Main" {
		on_complete {
			conditional_route {
				required_badges {
					must_have "vip"
				}
				next_storyline "NonExistent"
			}
		}
		enactment "Hook" {
			scene "Email" { subject "Test" body "Body" }
		}
	}
}
`
	result := ValidateScript(src)
	found := false
	for _, d := range result.Diagnostics {
		if d.Level == DiagError && containsStr(d.Message, "unknown storyline") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'unknown storyline' error for conditional route")
	}
}

func TestValidatorValidFullScript(t *testing.T) {
	src := `
story "Welcome Campaign" {
	priority 10
	allow_interruption true
	on_begin { give_badge "entered_welcome" }
	on_complete { give_badge "completed_welcome" }

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
				reply_to "reply@example.com"
			}
			on click "link_1" {
				within 1d
				when has_badge "warm"
				do jump_to_enactment "Offer"
				do give_badge "clicked_hook"
			}
			on not_click "link_1" {
				within 1d
				do next_scene
			}
			on not_open {
				within 1d
				do retry_scene up_to 2 times
					else do jump_to_enactment "Rescue"
			}
		}
		enactment "Offer" {
			level 2
			order 2
			scene "Offer Email" {
				subject "Special Offer"
				body "Buy now"
			}
			on click "buy_link" {
				do mark_complete
				do advance_to_next_storyline
			}
			on not_click "buy_link" {
				within 2d
				do loop_to_enactment "Hook" up_to 1
					else do mark_failed
			}
		}
		enactment "Rescue" {
			level 3
			order 3
			scene "Rescue Email" {
				subject "One more chance"
				body "Last chance"
			}
		}
	}

	storyline "Recovery" {
		order 2
		required_badges {
			must_have "clicked_hook"
		}
		enactment "Recovery Hook" {
			level 1
			scene "Recovery Email" {
				subject "We miss you"
				body "Come back"
			}
		}
	}
}
`
	result := ValidateScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("unexpected validation error: %s", d)
		}
	}
	if result.Symbols == nil {
		t.Error("expected non-nil symbol table")
	}
	if len(result.Symbols.BadgeNames) < 3 {
		t.Errorf("expected at least 3 badges, got %d", len(result.Symbols.BadgeNames))
	}
}

// helper
func containsStr(s, substr string) bool {
	return strings.Contains(s, substr)
}
