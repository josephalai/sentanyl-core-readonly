package scripting

import (
	"testing"
)

// ---------- Basic Parser Tests ----------

func TestParserEmptyStory(t *testing.T) {
	src := `story "Test Campaign" {}`
	result := ParseScript(src)
	if result.Diagnostics.HasErrors() {
		// Empty story has a warning about no storylines, but check for syntax errors
		for _, d := range result.Diagnostics {
			if d.Level == DiagError {
				t.Errorf("unexpected error: %s", d)
			}
		}
	}
	if len(result.AST.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.AST.Stories))
	}
	if result.AST.Stories[0].Name != "Test Campaign" {
		t.Errorf("expected story name 'Test Campaign', got %q", result.AST.Stories[0].Name)
	}
}

func TestParserStoryWithPriority(t *testing.T) {
	src := `story "My Campaign" {
		priority 10
		allow_interruption true
	}`
	result := ParseScript(src)
	story := result.AST.Stories[0]
	if story.Priority == nil || *story.Priority != 10 {
		t.Errorf("expected priority 10, got %v", story.Priority)
	}
	if story.AllowInterruption == nil || !*story.AllowInterruption {
		t.Errorf("expected allow_interruption true")
	}
}

func TestParserMinimalCampaign(t *testing.T) {
	src := `story "Welcome" {
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
				on click "link_1" {
					do jump_to_enactment "Offer"
				}
			}
			enactment "Offer" {
				level 2
				order 2
				scene "Offer Email" {
					subject "Special Offer"
					body "Buy now"
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	if len(result.AST.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.AST.Stories))
	}
	story := result.AST.Stories[0]
	if len(story.Storylines) != 1 {
		t.Fatalf("expected 1 storyline, got %d", len(story.Storylines))
	}
	sl := story.Storylines[0]
	if len(sl.Enactments) != 2 {
		t.Fatalf("expected 2 enactments, got %d", len(sl.Enactments))
	}
	// Check trigger
	en := sl.Enactments[0]
	if len(en.Triggers) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(en.Triggers))
	}
	tr := en.Triggers[0]
	if tr.TriggerType != "click" {
		t.Errorf("expected trigger type 'click', got %q", tr.TriggerType)
	}
	if tr.UserActionValue != "link_1" {
		t.Errorf("expected user action value 'link_1', got %q", tr.UserActionValue)
	}
	if len(tr.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(tr.Actions))
	}
	if tr.Actions[0].ActionType != "jump_to_enactment" {
		t.Errorf("expected action type 'jump_to_enactment', got %q", tr.Actions[0].ActionType)
	}
	if tr.Actions[0].Target != "Offer" {
		t.Errorf("expected target 'Offer', got %q", tr.Actions[0].Target)
	}
}

func TestParserMultiSceneEnactment(t *testing.T) {
	src := `story "Multi Scene" {
		storyline "Main" {
			enactment "Triple" {
				scene "Scene 1" {
					subject "First"
					body "Body 1"
				}
				scene "Scene 2" {
					subject "Second"
					body "Body 2"
				}
				scene "Scene 3" {
					subject "Third"
					body "Body 3"
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	en := result.AST.Stories[0].Storylines[0].Enactments[0]
	if len(en.Scenes) != 3 {
		t.Errorf("expected 3 scenes, got %d", len(en.Scenes))
	}
}

func TestParserTriggerWithConditions(t *testing.T) {
	src := `story "Conditional" {
		storyline "Main" {
			enactment "Hook" {
				scene "Email" {
					subject "Test"
					body "Body"
				}
				on click "link" {
					when has_badge "vip"
					do mark_complete
					do give_badge "clicked_hook"
					send_immediate true
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	tr := result.AST.Stories[0].Storylines[0].Enactments[0].Triggers[0]
	if len(tr.Conditions) != 1 {
		t.Fatalf("expected 1 condition, got %d", len(tr.Conditions))
	}
	if tr.Conditions[0].ConditionType != "has_badge" {
		t.Errorf("expected condition type 'has_badge', got %q", tr.Conditions[0].ConditionType)
	}
	if tr.Conditions[0].Value != "vip" {
		t.Errorf("expected condition value 'vip', got %q", tr.Conditions[0].Value)
	}
	if len(tr.Actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(tr.Actions))
	}
}

func TestParserTriggerWithElse(t *testing.T) {
	src := `story "Else Test" {
		storyline "Main" {
			enactment "Hook" {
				scene "Email" {
					subject "Test"
					body "Body"
				}
				on not_click "link" {
					within 1d
					do next_scene
					else {
						do jump_to_enactment "Rescue"
					}
				}
			}
			enactment "Rescue" {
				scene "Rescue Email" {
					subject "Rescue"
					body "Body"
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	tr := result.AST.Stories[0].Storylines[0].Enactments[0].Triggers[0]
	if tr.TriggerType != "not_click" {
		t.Errorf("expected 'not_click', got %q", tr.TriggerType)
	}
	if tr.Within == nil {
		t.Error("expected within duration")
	} else if tr.Within.Amount != 1 || tr.Within.Unit != "d" {
		t.Errorf("expected 1d, got %d%s", tr.Within.Amount, tr.Within.Unit)
	}
	if len(tr.Actions) != 1 {
		t.Errorf("expected 1 main action, got %d", len(tr.Actions))
	}
	if len(tr.ElseActions) != 1 {
		t.Errorf("expected 1 else action, got %d", len(tr.ElseActions))
	}
}

func TestParserRetryWithFallback(t *testing.T) {
	src := `story "Retry Test" {
		storyline "Main" {
			enactment "Hook" {
				scene "Email" {
					subject "Test"
					body "Body"
				}
				on not_open {
					within 1d
					do retry_scene up_to 2 times
						else do jump_to_enactment "Rescue"
				}
			}
			enactment "Rescue" {
				scene "Rescue" {
					subject "Rescue"
					body "Body"
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	tr := result.AST.Stories[0].Storylines[0].Enactments[0].Triggers[0]
	if len(tr.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(tr.Actions))
	}
	action := tr.Actions[0]
	if action.ActionType != "retry_scene" {
		t.Errorf("expected 'retry_scene', got %q", action.ActionType)
	}
	if action.RetryMaxCount == nil || *action.RetryMaxCount != 2 {
		t.Errorf("expected retry max 2, got %v", action.RetryMaxCount)
	}
	if len(action.RetryFallback) != 1 {
		t.Errorf("expected 1 fallback action, got %d", len(action.RetryFallback))
	}
}

func TestParserOnBeginOnComplete(t *testing.T) {
	src := `story "Lifecycle" {
		on_begin {
			give_badge "entered_campaign"
		}
		on_complete {
			give_badge "finished_campaign"
			remove_badge "entered_campaign"
			next_story "Follow Up"
		}
		on_fail {
			give_badge "failed_campaign"
		}
		storyline "Main" {
			enactment "Hook" {
				scene "Email" {
					subject "Test"
					body "Body"
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	story := result.AST.Stories[0]
	if story.OnBegin == nil {
		t.Fatal("expected on_begin")
	}
	if len(story.OnBegin.BadgeTransaction.GiveBadges) != 1 {
		t.Errorf("expected 1 give badge in on_begin")
	}
	if story.OnComplete == nil {
		t.Fatal("expected on_complete")
	}
	if len(story.OnComplete.BadgeTransaction.GiveBadges) != 1 {
		t.Errorf("expected 1 give badge in on_complete")
	}
	if len(story.OnComplete.BadgeTransaction.RemoveBadges) != 1 {
		t.Errorf("expected 1 remove badge in on_complete")
	}
	if story.OnComplete.NextStory != "Follow Up" {
		t.Errorf("expected next_story 'Follow Up', got %q", story.OnComplete.NextStory)
	}
	if story.OnFail == nil {
		t.Fatal("expected on_fail")
	}
}

func TestParserRequiredBadges(t *testing.T) {
	src := `story "Badge Guard" {
		required_badges {
			must_have "vip"
			must_not_have "banned"
		}
		storyline "Main" {
			required_badges {
				must_have ["warm", "engaged"]
			}
			enactment "Hook" {
				scene "Email" {
					subject "Test"
					body "Body"
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	story := result.AST.Stories[0]
	if story.RequiredBadges == nil {
		t.Fatal("expected required badges on story")
	}
	if len(story.RequiredBadges.MustHave) != 1 || story.RequiredBadges.MustHave[0] != "vip" {
		t.Errorf("expected must_have [vip], got %v", story.RequiredBadges.MustHave)
	}
	if len(story.RequiredBadges.MustNotHave) != 1 || story.RequiredBadges.MustNotHave[0] != "banned" {
		t.Errorf("expected must_not_have [banned], got %v", story.RequiredBadges.MustNotHave)
	}
	sl := story.Storylines[0]
	if sl.RequiredBadges == nil {
		t.Fatal("expected required badges on storyline")
	}
	if len(sl.RequiredBadges.MustHave) != 2 {
		t.Errorf("expected 2 must_have badges, got %d", len(sl.RequiredBadges.MustHave))
	}
}

func TestParserConditionalRoutes(t *testing.T) {
	src := `story "Routes" {
		storyline "Main" {
			on_complete {
				conditional_route {
					required_badges {
						must_have "vip"
					}
					next_storyline "VIP Path"
					priority 1
				}
				conditional_route {
					required_badges {
						must_not_have "vip"
					}
					next_storyline "Standard"
					priority 2
				}
			}
			enactment "Hook" {
				scene "Email" {
					subject "Test"
					body "Body"
				}
			}
		}
		storyline "VIP Path" {
			order 2
			enactment "VIP Hook" {
				scene "VIP Email" {
					subject "VIP"
					body "Body"
				}
			}
		}
		storyline "Standard" {
			order 3
			enactment "Std Hook" {
				scene "Std Email" {
					subject "Standard"
					body "Body"
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	sl := result.AST.Stories[0].Storylines[0]
	if sl.OnComplete == nil {
		t.Fatal("expected on_complete on storyline")
	}
	if len(sl.OnComplete.ConditionalRoutes) != 2 {
		t.Fatalf("expected 2 conditional routes, got %d", len(sl.OnComplete.ConditionalRoutes))
	}
	cr1 := sl.OnComplete.ConditionalRoutes[0]
	if cr1.NextStoryline != "VIP Path" {
		t.Errorf("expected first route to 'VIP Path', got %q", cr1.NextStoryline)
	}
}

func TestParserSceneWithTagsAndVars(t *testing.T) {
	src := `story "Tags" {
		storyline "Main" {
			enactment "Hook" {
				scene "Email" {
					subject "Test"
					body "<h1>Hello</h1>"
					template "welcome-template"
					tags ["promo", "welcome"]
					vars {
						firstName: "John"
						lastName: "Doe"
					}
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	sc := result.AST.Stories[0].Storylines[0].Enactments[0].Scenes[0]
	if sc.TemplateName != "welcome-template" {
		t.Errorf("expected template 'welcome-template', got %q", sc.TemplateName)
	}
	if len(sc.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(sc.Tags))
	}
	if len(sc.Vars) != 2 {
		t.Errorf("expected 2 vars, got %d", len(sc.Vars))
	}
	if sc.Vars["firstName"] != "John" {
		t.Errorf("expected var firstName='John', got %q", sc.Vars["firstName"])
	}
}

func TestParserMultipleTriggerTypes(t *testing.T) {
	src := `story "Triggers" {
		storyline "Main" {
			enactment "Hook" {
				scene "Email" { subject "Test" body "Body" }
				on open { do give_badge "opened" }
				on not_open { within 1d do next_scene }
				on bounce { do mark_failed }
				on spam { do unsubscribe }
				on sent { do give_badge "sent_ok" }
				on webhook "my-hook" { do advance_to_next_storyline }
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	en := result.AST.Stories[0].Storylines[0].Enactments[0]
	if len(en.Triggers) != 6 {
		t.Errorf("expected 6 triggers, got %d", len(en.Triggers))
	}
	types := []string{"open", "not_open", "bounce", "spam", "sent", "webhook"}
	for i, exp := range types {
		if en.Triggers[i].TriggerType != exp {
			t.Errorf("trigger %d: expected %q, got %q", i, exp, en.Triggers[i].TriggerType)
		}
	}
}

func TestParserTriggerPersistScope(t *testing.T) {
	src := `story "Scoped" {
		storyline "Main" {
			enactment "Hook" {
				scene "Email" { subject "Test" body "Body" }
				on click "link" {
					persist_scope "storyline"
					trigger_priority 3
					do mark_complete
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	tr := result.AST.Stories[0].Storylines[0].Enactments[0].Triggers[0]
	if tr.PersistScope != "storyline" {
		t.Errorf("expected persist_scope 'storyline', got %q", tr.PersistScope)
	}
	if tr.Priority == nil || *tr.Priority != 3 {
		t.Errorf("expected priority 3")
	}
}

func TestParserStartCompleteTrigger(t *testing.T) {
	src := `story "Triggers" {
		start_trigger "enrolled"
		complete_trigger "graduated"
		storyline "Main" {
			enactment "Hook" {
				scene "Email" { subject "Test" body "Body" }
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	story := result.AST.Stories[0]
	if story.StartTrigger == nil || *story.StartTrigger != "enrolled" {
		t.Errorf("expected start_trigger 'enrolled'")
	}
	if story.CompleteTrigger == nil || *story.CompleteTrigger != "graduated" {
		t.Errorf("expected complete_trigger 'graduated'")
	}
}

func TestParserSkipToNextStorylineOnExpiry(t *testing.T) {
	src := `story "Skip" {
		storyline "Main" {
			enactment "Hook" {
				skip_to_next_storyline_on_expiry true
				scene "Email" { subject "Test" body "Body" }
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	en := result.AST.Stories[0].Storylines[0].Enactments[0]
	if en.SkipToNextStorylineOnExpiry == nil || !*en.SkipToNextStorylineOnExpiry {
		t.Errorf("expected skip_to_next_storyline_on_expiry true")
	}
}

// ---------- Negative Parser Tests ----------

func TestParserMissingStoryName(t *testing.T) {
	src := `story {}`
	result := ParseScript(src)
	if !result.Diagnostics.HasErrors() {
		t.Error("expected parse errors for missing story name")
	}
}

func TestParserMissingBrace(t *testing.T) {
	src := `story "Test" { storyline "Main" { enactment "Hook" { scene "Email" { subject "Test" body "Body" } } }`
	result := ParseScript(src)
	if !result.Diagnostics.HasErrors() {
		t.Error("expected parse errors for missing closing brace")
	}
}

func TestParserUnexpectedTokenInStory(t *testing.T) {
	src := `story "Test" { foobar "unknown" }`
	result := ParseScript(src)
	if !result.Diagnostics.HasErrors() {
		t.Error("expected parse errors for unexpected token")
	}
}

func TestParserCampaignBlock(t *testing.T) {
	src := `campaign "Summer Launch" {
		from_email "hello@acme.com"
		from_name "Acme"
		reply_to "support@acme.com"
		context_pack "brand-tone-v1"
		subject_gen "Tease the v2 launch"
		body_gen "Lead with one outcome, list 3 highlights, single CTA"
		audience {
			must_have ["paid_subscriber"]
			must_not_have ["churned"]
		}
		on_click "https://acme.com/v2" {
			give_badge "v2_announce_click"
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Errorf("parse error: %s", d)
		}
	}
	if len(result.AST.Campaigns) != 1 {
		t.Fatalf("expected 1 campaign, got %d", len(result.AST.Campaigns))
	}
	c := result.AST.Campaigns[0]
	if c.Name != "Summer Launch" {
		t.Errorf("expected campaign name 'Summer Launch', got %q", c.Name)
	}
	if c.FromEmail != "hello@acme.com" {
		t.Errorf("expected from_email, got %q", c.FromEmail)
	}
	if c.SubjectGen == "" || c.BodyGen == "" {
		t.Errorf("expected subject_gen + body_gen to be set")
	}
	if len(c.ContextPackRefs) != 1 || c.ContextPackRefs[0] != "brand-tone-v1" {
		t.Errorf("expected context_pack 'brand-tone-v1', got %v", c.ContextPackRefs)
	}
	if c.Audience == nil {
		t.Fatal("expected audience block")
	}
	if len(c.Audience.MustHave) != 1 || c.Audience.MustHave[0] != "paid_subscriber" {
		t.Errorf("expected must_have ['paid_subscriber'], got %v", c.Audience.MustHave)
	}
	if len(c.Audience.MustNotHave) != 1 || c.Audience.MustNotHave[0] != "churned" {
		t.Errorf("expected must_not_have ['churned'], got %v", c.Audience.MustNotHave)
	}
	if len(c.OnClick) != 1 {
		t.Fatalf("expected 1 on_click rule, got %d", len(c.OnClick))
	}
	if c.OnClick[0].URLPattern != "https://acme.com/v2" {
		t.Errorf("expected url pattern, got %q", c.OnClick[0].URLPattern)
	}
	if c.OnClick[0].AwardBadge != "v2_announce_click" {
		t.Errorf("expected award badge, got %q", c.OnClick[0].AwardBadge)
	}
}
