package scripting

import (
	"testing"

	"gopkg.in/mgo.v2/bson"
)

// TestParseFunnelBasic verifies basic funnel parsing.
func TestParseFunnelBasic(t *testing.T) {
	src := `funnel "Test Funnel" {
		domain "test.example.com"
		route "Main" {
			order 1
			stage "Landing" {
				path "/landing"
				page "Landing Page" {
					template "minimal_v1"
					block "hero" {
						length short
						prompt "Test headline"
					}
					form "SignUp" {
						field email required
						field first_name
					}
				}
				on submit "SignUp" {
					do give_badge "lead"
				}
			}
		}
	}`

	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("parse error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.AST.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.AST.Funnels))
	}

	f := result.AST.Funnels[0]
	if f.Name != "Test Funnel" {
		t.Errorf("expected funnel name 'Test Funnel', got %q", f.Name)
	}
	if f.Domain != "test.example.com" {
		t.Errorf("expected domain 'test.example.com', got %q", f.Domain)
	}
	if len(f.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(f.Routes))
	}
	if len(f.Routes[0].Stages) != 1 {
		t.Fatalf("expected 1 stage, got %d", len(f.Routes[0].Stages))
	}

	stage := f.Routes[0].Stages[0]
	if stage.Path != "/landing" {
		t.Errorf("expected path '/landing', got %q", stage.Path)
	}
	if len(stage.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(stage.Pages))
	}
	if len(stage.Pages[0].Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(stage.Pages[0].Blocks))
	}
	if len(stage.Pages[0].Forms) != 1 {
		t.Errorf("expected 1 form, got %d", len(stage.Pages[0].Forms))
	}

	form := stage.Pages[0].Forms[0]
	if form.Name != "SignUp" {
		t.Errorf("expected form name 'SignUp', got %q", form.Name)
	}
	if len(form.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(form.Fields))
	}

	// Verify submit trigger
	if len(stage.Triggers) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(stage.Triggers))
	}
	if stage.Triggers[0].TriggerType != "submit" {
		t.Errorf("expected trigger type 'submit', got %q", stage.Triggers[0].TriggerType)
	}
}

// TestCompileFunnelBasic verifies basic funnel compilation.
func TestCompileFunnelBasic(t *testing.T) {
	src := `funnel "Test Funnel" {
		domain "test.example.com"
		route "Main" {
			order 1
			stage "Landing" {
				path "/landing"
				page "Landing Page" {
					template "minimal_v1"
					form "SignUp" {
						field email required
					}
				}
				on submit "SignUp" {
					do give_badge "lead"
				}
			}
		}
	}`

	ResetIDCounter()
	result := CompileScript(src, "sub_test", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}

	funnel := result.Funnels[0]
	if funnel.Name != "Test Funnel" {
		t.Errorf("expected funnel name 'Test Funnel', got %q", funnel.Name)
	}
	if funnel.Domain != "test.example.com" {
		t.Errorf("expected domain 'test.example.com', got %q", funnel.Domain)
	}
	if len(funnel.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(funnel.Routes))
	}
}

// TestCompileFunnelWithCheckout verifies checkout form compilation.
func TestCompileFunnelWithCheckout(t *testing.T) {
	src := `funnel "Checkout Test" {
		domain "shop.example.com"
		route "Buyers" {
			order 1
			stage "Product" {
				path "/product"
				page "Product Page" {
					template "tripwire_v1"
					form "Checkout" {
						type checkout
						product_id "prod-001"
						field email required
						field card required
					}
				}
				on purchase "Checkout" {
					do give_badge "buyer"
					do jump_to_stage "Confirmation"
				}
			}
			stage "Confirmation" {
				path "/confirmation"
				page "Confirmed" {
					template "minimal_v1"
				}
			}
		}
	}`

	ResetIDCounter()
	result := CompileScript(src, "sub_test", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}

	funnel := result.Funnels[0]
	route := funnel.Routes[0]
	if len(route.Stages) != 2 {
		t.Fatalf("expected 2 stages, got %d", len(route.Stages))
	}

	stage := route.Stages[0]
	if len(stage.Pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(stage.Pages))
	}
	page := stage.Pages[0]
	if len(page.Forms) != 1 {
		t.Fatalf("expected 1 form, got %d", len(page.Forms))
	}
	form := page.Forms[0]
	if form.FormType != "checkout" {
		t.Errorf("expected form type 'checkout', got %q", form.FormType)
	}
	if form.ProductId != "prod-001" {
		t.Errorf("expected product_id 'prod-001', got %q", form.ProductId)
	}
}

// TestCompileFunnelWithRoutes verifies multi-route funnel with badge gates.
func TestCompileFunnelWithRoutes(t *testing.T) {
	src := `funnel "Multi Route" {
		domain "test.example.com"
		route "Cold" {
			order 1
			must_not_have_badge "lead"
			stage "OptIn" {
				path "/optin"
				page "Opt In" {
					form "Lead" {
						field email required
					}
				}
				on submit "Lead" {
					do give_badge "lead"
				}
			}
		}
		route "Warm" {
			order 2
			must_have_badge "lead"
			stage "Offer" {
				path "/offer"
				page "Offer Page" {
					block "content" {
						length medium
						prompt "Offer details"
					}
				}
			}
		}
	}`

	ResetIDCounter()
	result := CompileScript(src, "sub_test", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}
	if len(result.Funnels[0].Routes) != 2 {
		t.Fatalf("expected 2 routes, got %d", len(result.Funnels[0].Routes))
	}
}

// TestCompileCombinedFunnelAndStory verifies a script with both funnel and story.
func TestCompileCombinedFunnelAndStory(t *testing.T) {
	src := `funnel "Launch Funnel" {
		domain "launch.example.com"
		route "Main" {
			order 1
			stage "OptIn" {
				path "/join"
				page "Join" {
					form "JoinForm" {
						field email required
					}
				}
				on submit "JoinForm" {
					do give_badge "member"
					do start_story "Welcome"
				}
			}
		}
	}

	story "Welcome" {
		storyline "Onboarding" {
			enactment "Welcome Email" {
				scene "Welcome" {
					subject "Welcome!"
					body "<p>Thanks for joining!</p>"
					from_email "hello@test.com"
					from_name "Test"
				}
				on sent {
					within 1d
					do next_scene
				}
			}
		}
	}`

	ResetIDCounter()
	result := CompileScript(src, "sub_test", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}
	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
}

// TestParseFunnelAIContext verifies AI context parsing.
func TestParseFunnelAIContext(t *testing.T) {
	src := `funnel "AI Test" {
		domain "ai.example.com"
		ai context global "https://example.com/data.txt" "source_data"
		route "Main" {
			order 1
			stage "Landing" {
				path "/"
				page "AI Page" {
					ai context extend "source_data"
					block "hero" {
						length short
						ai context extend "source_data"
						prompt "Test headline"
					}
					form "Subscribe" {
						field email required
					}
				}
			}
		}
	}`

	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("parse error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.AST.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.AST.Funnels))
	}

	f := result.AST.Funnels[0]
	if f.AIContext == nil {
		t.Fatal("expected AI context on funnel")
	}
	if f.AIContext.Mode != "global" {
		t.Errorf("expected context mode 'global', got %q", f.AIContext.Mode)
	}
}

func TestParseVideoBlock(t *testing.T) {
	src := `funnel "Video Test" {
		domain "test.com"
		route "Main" {
			order 1
			stage "Watch" {
				path "/watch"
				page "Video Page" {
					block "main_video" {
						type video
						source_url "https://cdn.example.com/intro.mp4"
						autoplay false
					}
					form "Lead" { field email required }
				}
				on submit "Lead" { do give_badge "lead" }
			}
		}
	}
	story "Companion" {
		storyline "Main" {
			enactment "E1" {
				scene "S1" {
					subject "Hi"
					body "<p>Hi</p>"
					from_email "a@b.com"
					from_name "A"
				}
			}
		}
	}`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("parse error: %s at %s", d.Message, d.Pos)
		}
	}
	if len(result.AST.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.AST.Funnels))
	}
	f := result.AST.Funnels[0]
	page := f.Routes[0].Stages[0].Pages[0]
	if len(page.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(page.Blocks))
	}
	block := page.Blocks[0]
	if block.BlockType != "video" {
		t.Errorf("expected block type 'video', got %q", block.BlockType)
	}
	if block.SourceURL != "https://cdn.example.com/intro.mp4" {
		t.Errorf("expected source URL, got %q", block.SourceURL)
	}
	if block.Autoplay {
		t.Error("expected autoplay false")
	}
}

func TestParseWatchTrigger(t *testing.T) {
	src := `funnel "Watch Test" {
		domain "test.com"
		route "Main" {
			order 1
			stage "Video" {
				path "/v"
				page "P" {
					block "vid" { type video  source_url "https://x.com/v.mp4" }
					form "F" { field email required }
				}
				on watch "vid" > 50 {
					do give_badge "engaged"
				}
				on watch "vid" >= 90 {
					do give_badge "completed"
				}
				on submit "F" { do give_badge "lead" }
			}
		}
	}
	story "C" { storyline "M" { enactment "E" { scene "S" { subject "x" body "<p>x</p>" from_email "a@b.com" from_name "A" } } } }`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("parse error: %s at %s", d.Message, d.Pos)
		}
	}
	triggers := result.AST.Funnels[0].Routes[0].Stages[0].Triggers
	if len(triggers) != 3 {
		t.Fatalf("expected 3 triggers, got %d", len(triggers))
	}
	w1 := triggers[0]
	if w1.TriggerType != "watch" {
		t.Errorf("expected trigger type 'watch', got %q", w1.TriggerType)
	}
	if w1.WatchBlockID != "vid" {
		t.Errorf("expected watch block 'vid', got %q", w1.WatchBlockID)
	}
	if w1.WatchOperator != ">" {
		t.Errorf("expected operator '>', got %q", w1.WatchOperator)
	}
	if w1.WatchPercent != 50 {
		t.Errorf("expected 50, got %d", w1.WatchPercent)
	}
	w2 := triggers[1]
	if w2.WatchOperator != ">=" {
		t.Errorf("expected operator '>=', got %q", w2.WatchOperator)
	}
	if w2.WatchPercent != 90 {
		t.Errorf("expected 90, got %d", w2.WatchPercent)
	}
}

func TestCompileVideoBlock(t *testing.T) {
	ResetIDCounter()
	src := `funnel "Video Compile" {
		domain "test.com"
		route "Main" {
			order 1
			stage "Watch" {
				path "/watch"
				page "Video Page" {
					block "main_video" {
						type video
						source_url "https://cdn.example.com/intro.mp4"
						autoplay true
					}
					form "Lead" { field email required }
				}
				on watch "main_video" > 75 {
					do give_badge "engaged"
				}
				on submit "Lead" { do give_badge "lead" }
			}
		}
	}
	story "Companion" {
		storyline "Main" { enactment "E" { scene "S" { subject "Hi" body "<p>x</p>" from_email "a@b.com" from_name "A" } } }
	}`
	result := CompileScript(src, "sub_test", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}
	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}
	funnel := result.Funnels[0]
	page := funnel.Routes[0].Stages[0].Pages[0]
	if len(page.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(page.Blocks))
	}
	block := page.Blocks[0]
	if block.BlockType != "video" {
		t.Errorf("expected block type 'video', got %q", block.BlockType)
	}
	if block.SourceURL != "https://cdn.example.com/intro.mp4" {
		t.Errorf("expected source URL, got %q", block.SourceURL)
	}
	if !block.Autoplay {
		t.Error("expected autoplay true")
	}
}

func TestParseCheckoutForm(t *testing.T) {
	src := `funnel "Checkout Test" {
		domain "shop.com"
		route "Main" {
			order 1
			stage "Buy" {
				path "/buy"
				page "Purchase" {
					form "PaymentForm" {
						type checkout
						product_id "course-v1"
						field email required
						field card required
					}
				}
				on purchase "PaymentForm" {
					do give_badge "buyer"
					do jump_to_stage "Thanks"
				}
			}
			stage "Thanks" {
				path "/thanks"
				page "Thanks" {
					block "confirm" { length short  prompt "Confirmed" }
				}
			}
		}
	}
	story "C" { storyline "M" { enactment "E" { scene "S" { subject "x" body "<p>x</p>" from_email "a@b.com" from_name "A" } } } }`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("parse error: %s at %s", d.Message, d.Pos)
		}
	}
	form := result.AST.Funnels[0].Routes[0].Stages[0].Pages[0].Forms[0]
	if form.FormType != "checkout" {
		t.Errorf("expected form type 'checkout', got %q", form.FormType)
	}
	if form.ProductID != "course-v1" {
		t.Errorf("expected product_id 'course-v1', got %q", form.ProductID)
	}
	if len(form.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(form.Fields))
	}
	triggers := result.AST.Funnels[0].Routes[0].Stages[0].Triggers
	if len(triggers) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(triggers))
	}
	if triggers[0].TriggerType != "purchase" {
		t.Errorf("expected 'purchase', got %q", triggers[0].TriggerType)
	}
}

func TestParseWatchWithPercent(t *testing.T) {
	src := `funnel "Pct Test" {
		domain "test.com"
		route "Main" {
			order 1
			stage "V" {
				path "/v"
				page "P" {
					block "vid" { type video  source_url "https://x.com/v.mp4" }
					form "F" { field email required }
				}
				on watch "vid" >= 90% {
					do give_badge "done"
				}
				on submit "F" { do give_badge "lead" }
			}
		}
	}
	story "C" { storyline "M" { enactment "E" { scene "S" { subject "x" body "<p>x</p>" from_email "a@b.com" from_name "A" } } } }`
	result := ParseScript(src)
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("parse error: %s at %s", d.Message, d.Pos)
		}
	}
	trigger := result.AST.Funnels[0].Routes[0].Stages[0].Triggers[0]
	if trigger.WatchOperator != ">=" {
		t.Errorf("expected '>=', got %q", trigger.WatchOperator)
	}
	if trigger.WatchPercent != 90 {
		t.Errorf("expected 90, got %d", trigger.WatchPercent)
	}
}
