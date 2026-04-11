package scripting

import (
	"testing"

	"gopkg.in/mgo.v2/bson"
)

// TestE2ELeadMagnetFunnel validates the lead magnet funnel fixture (41)
// compiles correctly with expected entity counts.
func TestE2ELeadMagnetFunnel(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(IntegrationTestLeadMagnetFunnel, "sub_e2e", bson.NewObjectId())

	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}
	funnel := result.Funnels[0]
	if funnel.Name != "Free Guide Funnel" {
		t.Errorf("expected funnel name 'Free Guide Funnel', got %q", funnel.Name)
	}
	if funnel.Domain != "guides.example.com" {
		t.Errorf("expected domain 'guides.example.com', got %q", funnel.Domain)
	}
	if len(funnel.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(funnel.Routes))
	}
	if len(funnel.Routes[0].Stages) != 2 {
		t.Errorf("expected 2 stages (Opt-In, Download), got %d", len(funnel.Routes[0].Stages))
	}

	// Must have at least 1 companion story
	if len(result.Stories) < 1 {
		t.Fatalf("expected at least 1 story, got %d", len(result.Stories))
	}
	t.Logf("funnel: %q, routes: %d, stages: %d, stories: %d",
		funnel.Name, len(funnel.Routes), len(funnel.Routes[0].Stages), len(result.Stories))
}

// TestE2EWebinarFunnel validates the webinar funnel fixture (42)
// compiles with video, checkout, and badge-gated routes.
func TestE2EWebinarFunnel(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(IntegrationTestWebinarFunnel, "sub_e2e", bson.NewObjectId())

	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}
	funnel := result.Funnels[0]
	if len(funnel.Routes) != 2 {
		t.Fatalf("expected 2 routes (Registration, Replay), got %d", len(funnel.Routes))
	}
	// Replay route should have badge gating
	replayRoute := funnel.Routes[1]
	if len(replayRoute.RequiredUserBadges.MustHave) == 0 {
		t.Error("expected replay route to have must_have badge gating")
	}

	// Should have 2 companion stories
	if len(result.Stories) < 2 {
		t.Errorf("expected at least 2 stories, got %d", len(result.Stories))
	}
	t.Logf("funnel: %q, routes: %d, stories: %d", funnel.Name, len(funnel.Routes), len(result.Stories))
}

// TestE2EProductLaunchFunnel validates the PLF-style product launch fixture (43)
// compiles with multiple video stages and sequential progression.
func TestE2EProductLaunchFunnel(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(IntegrationTestProductLaunchFunnel, "sub_e2e", bson.NewObjectId())

	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}
	funnel := result.Funnels[0]
	if len(funnel.Routes[0].Stages) != 5 {
		t.Errorf("expected 5 stages (Video 1-3, Cart Open, Welcome), got %d", len(funnel.Routes[0].Stages))
	}

	// Should have 2 companion stories
	if len(result.Stories) < 2 {
		t.Errorf("expected at least 2 stories, got %d", len(result.Stories))
	}
	t.Logf("funnel: %q, stages: %d, stories: %d",
		funnel.Name, len(funnel.Routes[0].Stages), len(result.Stories))
}

// TestE2EMultiRouteMembership validates the multi-route membership fixture (44)
// compiles with 3 badge-gated routes and progressive badges.
func TestE2EMultiRouteMembership(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(IntegrationTestMultiRouteMembership, "sub_e2e", bson.NewObjectId())

	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}
	funnel := result.Funnels[0]
	if len(funnel.Routes) != 3 {
		t.Fatalf("expected 3 routes (Public, Free, Paid), got %d", len(funnel.Routes))
	}

	// Public route should have must_not_have badge
	publicRoute := funnel.Routes[0]
	if len(publicRoute.RequiredUserBadges.MustNotHave) == 0 {
		t.Error("expected public route to have must_not_have badge gating")
	}
	// Paid route should have must_have badge
	paidRoute := funnel.Routes[2]
	if len(paidRoute.RequiredUserBadges.MustHave) == 0 {
		t.Error("expected paid route to have must_have badge gating")
	}

	if len(result.Stories) < 2 {
		t.Errorf("expected at least 2 stories, got %d", len(result.Stories))
	}
	t.Logf("funnel: %q, routes: %d, stories: %d", funnel.Name, len(funnel.Routes), len(result.Stories))
}

// TestE2EUpsellPipeline validates the upsell pipeline fixture (45)
// compiles with sequential checkout stages and video.
func TestE2EUpsellPipeline(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(IntegrationTestUpsellPipeline, "sub_e2e", bson.NewObjectId())

	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}
	funnel := result.Funnels[0]
	if len(funnel.Routes[0].Stages) != 4 {
		t.Errorf("expected 4 stages (Main, Bump, OTO, Thank You), got %d", len(funnel.Routes[0].Stages))
	}

	// Should have at least 2 stories
	if len(result.Stories) < 2 {
		t.Errorf("expected at least 2 stories, got %d", len(result.Stories))
	}
	t.Logf("funnel: %q, stages: %d, stories: %d",
		funnel.Name, len(funnel.Routes[0].Stages), len(result.Stories))
}

// TestE2EAbandonRecovery validates the abandon recovery fixture (46)
// compiles with abandon trigger and badge-gated recovery story.
func TestE2EAbandonRecovery(t *testing.T) {
	ResetIDCounter()
	result := CompileScript(IntegrationTestAbandonRecovery, "sub_e2e", bson.NewObjectId())

	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error: %s at %s", d.Message, d.Pos)
		}
	}

	if len(result.Funnels) != 1 {
		t.Fatalf("expected 1 funnel, got %d", len(result.Funnels))
	}
	funnel := result.Funnels[0]
	if len(funnel.Routes[0].Stages) != 2 {
		t.Errorf("expected 2 stages (Sales Page, Thank You), got %d", len(funnel.Routes[0].Stages))
	}

	// Should have 2 stories (Cart Recovery, Customer Welcome)
	if len(result.Stories) < 2 {
		t.Errorf("expected at least 2 stories, got %d", len(result.Stories))
	}

	// Cart Recovery story should have required_badges
	for _, story := range result.Stories {
		if story.Name == "Cart Recovery" {
			if len(story.RequiredUserBadges.MustHave) == 0 {
				t.Error("Cart Recovery story should require 'cart_abandoner' badge")
			}
			if len(story.RequiredUserBadges.MustNotHave) == 0 {
				t.Error("Cart Recovery story should exclude 'customer' badge")
			}
		}
	}
	t.Logf("funnel: %q, stages: %d, stories: %d",
		funnel.Name, len(funnel.Routes[0].Stages), len(result.Stories))
}
