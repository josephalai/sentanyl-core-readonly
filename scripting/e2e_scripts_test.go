package scripting

import (
	"strings"
	"testing"

	pkgmodels "github.com/josephalai/sentanyl/pkg/models"

	"gopkg.in/mgo.v2/bson"
)

// Helper: compile a fixture and assert no errors
func compileFixtureE2E(t *testing.T, src string) *CompileResult {
	t.Helper()
	ResetIDCounter()
	result := CompileScript(src, "sub_e2e", bson.NewObjectId())
	for _, d := range result.Diagnostics {
		if d.Level == DiagError {
			t.Fatalf("compile error at %s: %s", d.Pos, d.Message)
		}
	}
	return result
}

// ---------------------------------------------------------------------------
// 1. e2e-compound-trigger-conditions.sh
// ---------------------------------------------------------------------------

func TestE2ECompoundTriggerConditions(t *testing.T) {
	result := compileFixtureE2E(t, FixtureCompoundTriggerConditions)

	if len(result.Stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(result.Stories))
	}
	story := result.Stories[0]
	if story.Name != "Elite Member Campaign" {
		t.Errorf("expected story name 'Elite Member Campaign', got %q", story.Name)
	}
	if story.Priority != 1 {
		t.Errorf("expected priority 1, got %d", story.Priority)
	}
	if story.AllowInterruption != false {
		t.Errorf("expected allow_interruption false")
	}
	if story.StartTrigger == nil {
		t.Fatal("expected start_trigger")
	}
	if story.StartTrigger.Name != "story-start" {
		t.Errorf("expected start_trigger 'story-start', got %q", story.StartTrigger.Name)
	}

	// 1 storyline
	if len(story.Storylines) != 1 {
		t.Fatalf("expected 1 storyline, got %d", len(story.Storylines))
	}
	sl := story.Storylines[0]

	// 4 enactments
	if len(sl.Acts) != 4 {
		t.Fatalf("expected 4 enactments, got %d", len(sl.Acts))
	}

	// First enactment should have 3 triggers (3 click triggers)
	campaignEn := sl.Acts[0]
	totalTriggers := 0
	for _, triggers := range campaignEn.OnEvent {
		totalTriggers += len(triggers)
	}
	if totalTriggers != 3 {
		t.Errorf("expected 3 triggers on campaign enactment, got %d", totalTriggers)
	}

	// Verify compound badge requirement on first trigger (Elite)
	clickTriggers := campaignEn.OnEvent[pkgmodels.OnClick]
	if len(clickTriggers) == 0 {
		t.Fatal("expected OnClick triggers")
	}
	// Find the elite trigger (priority 3)
	var eliteTrigger *pkgmodels.Trigger
	for _, tr := range clickTriggers {
		if int(tr.Priority) == 3 {
			eliteTrigger = tr
			break
		}
	}
	if eliteTrigger == nil {
		t.Fatal("expected trigger with priority 3 (elite)")
	}
	if len(eliteTrigger.RequiredBadges.MustHave) != 2 {
		t.Errorf("expected 2 must_have badges on elite trigger, got %d", len(eliteTrigger.RequiredBadges.MustHave))
	}

	// Verify badges
	if _, ok := result.Badges["vip"]; !ok {
		t.Error("expected 'vip' badge")
	}
	if _, ok := result.Badges["verified"]; !ok {
		t.Error("expected 'verified' badge")
	}
	if _, ok := result.Badges["story-start"]; !ok {
		t.Error("expected 'story-start' badge")
	}
}

// ---------------------------------------------------------------------------
// 2. e2e-conditional-routing.sh
// ---------------------------------------------------------------------------

func TestE2EConditionalRoutingScript(t *testing.T) {
	result := compileFixtureE2E(t, FixtureConditionalRouting)

	story := result.Stories[0]
	if story.Name != "Premium Learning Path" {
		t.Errorf("expected story name 'Premium Learning Path', got %q", story.Name)
	}

	// 3 storylines
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	sl1 := story.Storylines[0]
	if sl1.Name != "SL1 — Introduction" {
		t.Errorf("expected storyline name 'SL1 — Introduction', got %q", sl1.Name)
	}

	// SL1 should have 2 conditional routes
	if len(sl1.OnComplete.ConditionalRoutes) != 2 {
		t.Fatalf("expected 2 conditional routes, got %d", len(sl1.OnComplete.ConditionalRoutes))
	}

	// First route (priority 10) should require premium-member
	route1 := sl1.OnComplete.ConditionalRoutes[0]
	if route1.Priority != 10 {
		t.Errorf("expected route 1 priority 10, got %d", route1.Priority)
	}
	if len(route1.RequiredBadges.MustHave) != 1 {
		t.Errorf("expected 1 must_have on route 1, got %d", len(route1.RequiredBadges.MustHave))
	}
	if route1.NextStoryline == nil {
		t.Error("expected route 1 to have next_storyline wired")
	}

	// Second route (priority 1) — fallback, no badge requirement
	route2 := sl1.OnComplete.ConditionalRoutes[1]
	if route2.Priority != 1 {
		t.Errorf("expected route 2 priority 1, got %d", route2.Priority)
	}

	// Click trigger on SL1 enactment
	en := sl1.Acts[0]
	if len(en.OnEvent[pkgmodels.OnClick]) != 1 {
		t.Errorf("expected 1 OnClick trigger, got %d", len(en.OnEvent[pkgmodels.OnClick]))
	}
	tr := en.OnEvent[pkgmodels.OnClick][0]
	if tr.DoAction == nil || !tr.DoAction.AdvanceToNextStoryline {
		t.Error("expected advance_to_next_storyline on click trigger")
	}
}

// ---------------------------------------------------------------------------
// 3. e2e-conditional-trigger.sh
// ---------------------------------------------------------------------------

func TestE2EConditionalTriggerScript(t *testing.T) {
	result := compileFixtureE2E(t, FixtureConditionalTrigger)

	story := result.Stories[0]
	if story.Name != "VIP Membership Campaign" {
		t.Errorf("unexpected story name %q", story.Name)
	}

	// 3 storylines
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	// SL1 campaign enactment has 3 click triggers
	sl1 := story.Storylines[0]
	campaignEn := sl1.Acts[0]
	clickTriggers := campaignEn.OnEvent[pkgmodels.OnClick]
	if len(clickTriggers) != 3 {
		t.Fatalf("expected 3 click triggers on SL1 campaign enactment, got %d", len(clickTriggers))
	}

	// SL-VIP storyline has required_badges must_have vip-member
	slVIP := story.Storylines[1]
	if len(slVIP.RequiredUserBadges.MustHave) != 1 {
		t.Errorf("expected 1 must_have badge on VIP storyline, got %d", len(slVIP.RequiredUserBadges.MustHave))
	}
	if slVIP.RequiredUserBadges.MustHave[0].Name != "vip-member" {
		t.Errorf("expected must_have 'vip-member', got %q", slVIP.RequiredUserBadges.MustHave[0].Name)
	}

	// SL-STD storyline has required_badges must_not_have vip-member
	slSTD := story.Storylines[2]
	if len(slSTD.RequiredUserBadges.MustNotHave) != 1 {
		t.Errorf("expected 1 must_not_have badge on Standard storyline, got %d", len(slSTD.RequiredUserBadges.MustNotHave))
	}
	if slSTD.RequiredUserBadges.MustNotHave[0].Name != "vip-member" {
		t.Errorf("expected must_not_have 'vip-member', got %q", slSTD.RequiredUserBadges.MustNotHave[0].Name)
	}

	// VIP confirmation trigger has mark_complete
	vipEn := slVIP.Acts[0]
	vipClickTriggers := vipEn.OnEvent[pkgmodels.OnClick]
	if len(vipClickTriggers) != 1 {
		t.Fatalf("expected 1 click trigger on VIP enactment, got %d", len(vipClickTriggers))
	}
	if !vipClickTriggers[0].MarkComplete {
		t.Error("expected mark_complete on VIP confirmation trigger")
	}
}

// ---------------------------------------------------------------------------
// 4. e2e-storyline-badge-gating.sh
// ---------------------------------------------------------------------------

func TestE2EStorylineBadgeGatingScript(t *testing.T) {
	result := compileFixtureE2E(t, FixtureStorylineBadgeGating)

	story := result.Stories[0]
	if story.Name != "Adaptive Learning Path" {
		t.Errorf("unexpected story name %q", story.Name)
	}

	// 3 storylines
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	// SL1 — no required badges
	sl1 := story.Storylines[0]
	if len(sl1.RequiredUserBadges.MustHave) != 0 {
		t.Error("SL1 should have no required badges")
	}

	// SL2 — must_have advanced-learner
	sl2 := story.Storylines[1]
	if len(sl2.RequiredUserBadges.MustHave) != 1 {
		t.Fatalf("expected 1 must_have on SL2, got %d", len(sl2.RequiredUserBadges.MustHave))
	}
	if sl2.RequiredUserBadges.MustHave[0].Name != "advanced-learner" {
		t.Errorf("expected 'advanced-learner', got %q", sl2.RequiredUserBadges.MustHave[0].Name)
	}

	// SL3 — no required badges
	sl3 := story.Storylines[2]
	if len(sl3.RequiredUserBadges.MustHave) != 0 {
		t.Error("SL3 should have no required badges")
	}

	// Click trigger on SL1 enactment advances to next storyline
	en := sl1.Acts[0]
	tr := en.OnEvent[pkgmodels.OnClick]
	if len(tr) != 1 {
		t.Fatalf("expected 1 click trigger on SL1, got %d", len(tr))
	}
	if !tr[0].DoAction.AdvanceToNextStoryline {
		t.Error("expected advance_to_next_storyline on SL1 click")
	}
}

// ---------------------------------------------------------------------------
// 5. e2e-story-interruption.sh
// ---------------------------------------------------------------------------

func TestE2EStoryInterruptionScript(t *testing.T) {
	result := compileFixtureE2E(t, FixtureStoryInterruption)

	// 2 stories
	if len(result.Stories) != 2 {
		t.Fatalf("expected 2 stories, got %d", len(result.Stories))
	}

	var newsletter, cart *pkgmodels.Story
	for _, s := range result.Stories {
		switch s.Name {
		case "Monthly Newsletter":
			newsletter = s
		case "Cart Abandonment Recovery":
			cart = s
		}
	}

	if newsletter == nil {
		t.Fatal("missing 'Monthly Newsletter' story")
	}
	if cart == nil {
		t.Fatal("missing 'Cart Abandonment Recovery' story")
	}

	// Newsletter: priority 1, allow_interruption true
	if newsletter.Priority != 1 {
		t.Errorf("newsletter priority expected 1, got %d", newsletter.Priority)
	}
	if !newsletter.AllowInterruption {
		t.Error("newsletter should have allow_interruption true")
	}
	if newsletter.StartTrigger == nil || newsletter.StartTrigger.Name != "newsletter-subscriber" {
		t.Error("newsletter should have start_trigger 'newsletter-subscriber'")
	}

	// Newsletter has 3 enactments
	if len(newsletter.Storylines) != 1 {
		t.Fatalf("expected 1 storyline in newsletter, got %d", len(newsletter.Storylines))
	}
	if len(newsletter.Storylines[0].Acts) != 3 {
		t.Fatalf("expected 3 enactments in newsletter, got %d", len(newsletter.Storylines[0].Acts))
	}

	// Cart: priority 10, allow_interruption false
	if cart.Priority != 10 {
		t.Errorf("cart priority expected 10, got %d", cart.Priority)
	}
	if cart.AllowInterruption {
		t.Error("cart should have allow_interruption false")
	}
	if cart.StartTrigger == nil || cart.StartTrigger.Name != "cart-abandoned" {
		t.Error("cart should have start_trigger 'cart-abandoned'")
	}

	// Cart has 1 enactment with click trigger → advance_to_next_storyline
	cartEn := cart.Storylines[0].Acts[0]
	cartClick := cartEn.OnEvent[pkgmodels.OnClick]
	if len(cartClick) != 1 {
		t.Fatalf("expected 1 click trigger on cart, got %d", len(cartClick))
	}
	if !cartClick[0].DoAction.AdvanceToNextStoryline {
		t.Error("expected advance_to_next_storyline on cart click")
	}
}

// ---------------------------------------------------------------------------
// 6. e2e-outbound-webhooks.sh
// ---------------------------------------------------------------------------

func TestE2EOutboundWebhooksScript(t *testing.T) {
	result := compileFixtureE2E(t, FixtureOutboundWebhooks)

	story := result.Stories[0]
	if story.Name != "Webhook Demo Story" {
		t.Errorf("unexpected story name %q", story.Name)
	}
	if story.Priority != 1 {
		t.Errorf("expected priority 1, got %d", story.Priority)
	}

	// 1 storyline, 1 enactment, 1 click trigger
	sl := story.Storylines[0]
	en := sl.Acts[0]
	if en.SendScene.Message.Content.Subject != "🪝 Webhook Demo — Click to Fire Events" {
		t.Errorf("unexpected subject %q", en.SendScene.Message.Content.Subject)
	}
	tr := en.OnEvent[pkgmodels.OnClick]
	if len(tr) != 1 {
		t.Fatalf("expected 1 click trigger, got %d", len(tr))
	}
	if !tr[0].DoAction.AdvanceToNextStoryline {
		t.Error("expected advance_to_next_storyline")
	}
}

// ---------------------------------------------------------------------------
// 7. e2e-persistent-links.sh
// ---------------------------------------------------------------------------

func TestE2EPersistentLinksScript(t *testing.T) {
	result := compileFixtureE2E(t, FixturePersistentLinks)

	story := result.Stories[0]
	if story.Name != "Online Course — Persistent Links Demo" {
		t.Errorf("unexpected story name %q", story.Name)
	}

	sl := story.Storylines[0]
	// 12 enactments
	if len(sl.Acts) != 12 {
		t.Fatalf("expected 12 enactments, got %d", len(sl.Acts))
	}

	// Verify persist_scope on EA/EB enactments (first 6)
	for i := 0; i < 6; i++ {
		en := sl.Acts[i]
		clickTriggers := en.OnEvent[pkgmodels.OnClick]
		if len(clickTriggers) == 0 {
			t.Fatalf("expected click trigger on enactment %d", i)
		}
		if clickTriggers[0].PersistScope != "enactment" {
			t.Errorf("enactment %d (%s): expected persist_scope 'enactment', got %q",
				i, en.Name, clickTriggers[0].PersistScope)
		}
	}

	// Verify EC/ED enactments have NO persist_scope (default)
	for i := 6; i < 12; i++ {
		en := sl.Acts[i]
		clickTriggers := en.OnEvent[pkgmodels.OnClick]
		if len(clickTriggers) == 0 {
			t.Fatalf("expected click trigger on enactment %d", i)
		}
		if clickTriggers[0].PersistScope != "" {
			t.Errorf("enactment %d (%s): expected empty persist_scope, got %q",
				i, en.Name, clickTriggers[0].PersistScope)
		}
	}

	// Verify A/B triggers jump to EC-Sc1 (forward reference stored in ActionName)
	ea1Click := sl.Acts[0].OnEvent[pkgmodels.OnClick][0]
	if ea1Click.DoAction == nil {
		t.Fatal("expected action on EA-Sc1 trigger")
	}
	// Forward reference resolved via NextEnactmentId (not NextEnactment, to avoid
	// duplicate entity inserts during ReadyMongoStore).
	if ea1Click.DoAction.NextEnactment == nil && ea1Click.DoAction.NextEnactmentId == nil {
		t.Error("expected EA-Sc1 trigger to reference EC-Sc1")
	}

	// Verify C/D triggers advance to next storyline
	ec1Click := sl.Acts[6].OnEvent[pkgmodels.OnClick][0]
	if ec1Click.DoAction == nil || !ec1Click.DoAction.AdvanceToNextStoryline {
		t.Error("expected EC-Sc1 trigger to advance_to_next_storyline")
	}
}

// ---------------------------------------------------------------------------
// 8. e2e-deferred-transitions.sh
// ---------------------------------------------------------------------------

func TestE2EDeferredTransitionsScript(t *testing.T) {
	result := compileFixtureE2E(t, FixtureDeferredTransitions)

	story := result.Stories[0]
	if story.Name != "Buy All Three Manifesting Workshops" {
		t.Errorf("unexpected story name %q", story.Name)
	}

	// 3 storylines
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	// Each storyline has 12 enactments
	for i, sl := range story.Storylines {
		if len(sl.Acts) != 12 {
			t.Errorf("storyline %d: expected 12 enactments, got %d", i+1, len(sl.Acts))
		}
	}

	sl1 := story.Storylines[0]

	// Verify on_complete badge transaction (sl1-purchased)
	if sl1.OnComplete.BadgeTransaction == nil {
		t.Fatal("expected on_complete badge transaction on SL1")
	}
	if len(sl1.OnComplete.BadgeTransaction.GiveBadges) != 1 {
		t.Fatal("expected 1 give_badge on SL1 on_complete")
	}
	if sl1.OnComplete.BadgeTransaction.GiveBadges[0].Name != "sl1-purchased" {
		t.Errorf("expected 'sl1-purchased', got %q", sl1.OnComplete.BadgeTransaction.GiveBadges[0].Name)
	}

	// Verify on_fail badge transaction (sl1-not-purchased)
	if sl1.OnFail.BadgeTransaction == nil {
		t.Fatal("expected on_fail badge transaction on SL1")
	}
	if len(sl1.OnFail.BadgeTransaction.GiveBadges) != 1 {
		t.Fatal("expected 1 give_badge on SL1 on_fail")
	}
	if sl1.OnFail.BadgeTransaction.GiveBadges[0].Name != "sl1-not-purchased" {
		t.Errorf("expected 'sl1-not-purchased', got %q", sl1.OnFail.BadgeTransaction.GiveBadges[0].Name)
	}

	// Verify send_immediate false on A/B triggers
	ea1 := sl1.Acts[0]
	ea1Click := ea1.OnEvent[pkgmodels.OnClick]
	if len(ea1Click) == 0 {
		t.Fatal("expected click trigger on EA-Sc1")
	}
	action := ea1Click[0].DoAction
	if action == nil {
		t.Fatal("expected action on EA-Sc1 trigger")
	}
	if action.SendImmediate == nil || *action.SendImmediate != false {
		t.Error("expected send_immediate false on deferred transition trigger")
	}

	// Verify B-Sc3 has skip_to_next_storyline_on_expiry
	ebSc3 := sl1.Acts[5] // order 6 = index 5
	if !ebSc3.SkipToNextStorylineOnExpiry {
		t.Errorf("expected skip_to_next_storyline_on_expiry on SL1-EB-Sc3")
	}

	// Verify C/D triggers have advance_to_next_storyline with send_immediate false
	ec1 := sl1.Acts[6] // order 7 = index 6
	ec1Click := ec1.OnEvent[pkgmodels.OnClick]
	if len(ec1Click) == 0 {
		t.Fatal("expected click trigger on EC-Sc1")
	}
	cAction := ec1Click[0].DoAction
	if !cAction.AdvanceToNextStoryline {
		t.Error("expected advance_to_next_storyline on EC trigger")
	}
	if cAction.SendImmediate == nil || *cAction.SendImmediate != false {
		t.Error("expected send_immediate false on EC trigger")
	}

	// Verify all badges exist
	expectedBadges := []string{"start_story_a", "sl1-purchased", "sl1-not-purchased",
		"sl2-purchased", "sl2-not-purchased", "sl3-purchased", "sl3-not-purchased"}
	for _, name := range expectedBadges {
		if _, ok := result.Badges[name]; !ok {
			t.Errorf("missing badge %q", name)
		}
	}
}

// ---------------------------------------------------------------------------
// 9. e2e-mailhog-full-sequence.sh (instant transitions)
// ---------------------------------------------------------------------------

func TestE2EMailhogFullSequenceScript(t *testing.T) {
	result := compileFixtureE2E(t, FixtureMailhogFullSequence)

	story := result.Stories[0]
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	sl1 := story.Storylines[0]
	if len(sl1.Acts) != 12 {
		t.Fatalf("expected 12 enactments in SL1, got %d", len(sl1.Acts))
	}

	// A triggers: jump to EC-Sc1 (NO send_immediate false = instant)
	ea1 := sl1.Acts[0]
	ea1Click := ea1.OnEvent[pkgmodels.OnClick]
	if len(ea1Click) == 0 {
		t.Fatal("expected click trigger on EA-Sc1")
	}
	action := ea1Click[0].DoAction
	// Forward reference resolved via NextEnactmentId.
	if action.NextEnactment == nil && action.NextEnactmentId == nil {
		t.Error("expected next_enactment or forward reference to SL1-EC-Sc1 on EA-Sc1 trigger")
	}
	// send_immediate should be nil (not explicitly set, defaults to true in engine)
	if action.SendImmediate != nil {
		t.Errorf("expected send_immediate nil (instant/default), got %v", *action.SendImmediate)
	}

	// C triggers: advance_to_next_storyline (instant)
	ec1 := sl1.Acts[6]
	ec1Click := ec1.OnEvent[pkgmodels.OnClick]
	if len(ec1Click) == 0 {
		t.Fatal("expected click trigger on EC-Sc1")
	}
	if !ec1Click[0].DoAction.AdvanceToNextStoryline {
		t.Error("expected advance_to_next_storyline on EC trigger")
	}

	// B-Sc3 skip_to_next_storyline_on_expiry
	ebSc3 := sl1.Acts[5]
	if !ebSc3.SkipToNextStorylineOnExpiry {
		t.Error("expected skip_to_next_storyline_on_expiry on EB-Sc3")
	}
}

// ---------------------------------------------------------------------------
// 10. e2e-hybrid-transitions.sh
// ---------------------------------------------------------------------------

func TestE2EHybridTransitionsScript(t *testing.T) {
	result := compileFixtureE2E(t, FixtureHybridTransitions)

	story := result.Stories[0]
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	sl1 := story.Storylines[0]
	// 13 enactments (12 + thank you)
	if len(sl1.Acts) != 13 {
		t.Fatalf("expected 13 enactments in SL1, got %d", len(sl1.Acts))
	}

	// A/B triggers: deferred (send_immediate false) → jump to EC-Sc1
	ea1 := sl1.Acts[0]
	ea1Click := ea1.OnEvent[pkgmodels.OnClick]
	if len(ea1Click) == 0 {
		t.Fatal("expected click trigger on EA-Sc1")
	}
	aAction := ea1Click[0].DoAction
	if aAction.SendImmediate == nil || *aAction.SendImmediate != false {
		t.Error("expected send_immediate false on A enactment trigger")
	}

	// C/D triggers: instant → jump to EE-Sc1 (thank you)
	ec1 := sl1.Acts[6]
	ec1Click := ec1.OnEvent[pkgmodels.OnClick]
	if len(ec1Click) == 0 {
		t.Fatal("expected click trigger on EC-Sc1")
	}
	cAction := ec1Click[0].DoAction
	// send_immediate NOT set → instant
	if cAction.SendImmediate != nil {
		t.Errorf("expected send_immediate nil (instant) on C trigger, got %v", *cAction.SendImmediate)
	}
	// C/D should jump to EE-Sc1, not advance_to_next_storyline
	// Forward reference resolved via NextEnactmentId.
	if cAction.NextEnactment == nil && cAction.NextEnactmentId == nil {
		t.Error("expected next_enactment or forward reference (EE-Sc1) on C trigger")
	}

	// EE-Sc1 (thank you) should have no triggers
	eeSc1 := sl1.Acts[12]
	totalTriggers := 0
	for _, triggers := range eeSc1.OnEvent {
		totalTriggers += len(triggers)
	}
	if totalTriggers != 0 {
		t.Errorf("expected 0 triggers on thank-you enactment, got %d", totalTriggers)
	}
	// Verify it has the right subject
	if !strings.Contains(eeSc1.SendScene.Message.Content.Subject, "Thank you") {
		t.Errorf("expected thank-you subject, got %q", eeSc1.SendScene.Message.Content.Subject)
	}
}

// ---------------------------------------------------------------------------
// 11. multiple-storyline-enactment-scene.sh
// ---------------------------------------------------------------------------

func TestE2EMultiStorylineEnactmentSceneScript(t *testing.T) {
	result := compileFixtureE2E(t, FixtureMultiStorylineEnactmentScene)

	story := result.Stories[0]
	if story.Name != "Manifesting Workshops Complete Bundle" {
		t.Errorf("unexpected story name %q", story.Name)
	}

	// 3 storylines
	if len(story.Storylines) != 3 {
		t.Fatalf("expected 3 storylines, got %d", len(story.Storylines))
	}

	// Each storyline has 12 enactments (4 types × 3 scenes)
	for i, sl := range story.Storylines {
		if len(sl.Acts) != 12 {
			t.Errorf("storyline %d: expected 12 enactments, got %d", i+1, len(sl.Acts))
		}
	}

	// Verify scene subjects use correct product names
	sl1 := story.Storylines[0]
	if !strings.Contains(sl1.Acts[0].SendScene.Message.Content.Subject, "Manifesting 101") {
		t.Errorf("expected 'Manifesting 101' in SL1 subject, got %q",
			sl1.Acts[0].SendScene.Message.Content.Subject)
	}

	sl2 := story.Storylines[1]
	if !strings.Contains(sl2.Acts[0].SendScene.Message.Content.Subject, "Advanced Attraction") {
		t.Errorf("expected 'Advanced Attraction' in SL2 subject, got %q",
			sl2.Acts[0].SendScene.Message.Content.Subject)
	}

	sl3 := story.Storylines[2]
	if !strings.Contains(sl3.Acts[0].SendScene.Message.Content.Subject, "Quantum Wealth") {
		t.Errorf("expected 'Quantum Wealth' in SL3 subject, got %q",
			sl3.Acts[0].SendScene.Message.Content.Subject)
	}

	// Verify all triggers have mark_complete true
	for _, sl := range story.Storylines {
		for _, en := range sl.Acts {
			for _, triggers := range en.OnEvent {
				for _, tr := range triggers {
					if !tr.MarkComplete {
						t.Errorf("expected mark_complete on trigger in enactment %q", en.Name)
					}
				}
			}
		}
	}

	// Total scenes: 3 × 12 = 36
	totalScenes := 0
	for _, sl := range story.Storylines {
		for _, en := range sl.Acts {
			if en.SendScene != nil {
				totalScenes++
			}
			totalScenes += len(en.SendScenes)
		}
	}
	// Each single-scene enactment counts once via SendScene
	if totalScenes < 36 {
		t.Errorf("expected at least 36 scenes, got %d", totalScenes)
	}
}

// ---------------------------------------------------------------------------
// Cross-cutting: Verify all fixtures parse + validate + compile cleanly
// ---------------------------------------------------------------------------

func TestE2EAllScriptFixturesCompile(t *testing.T) {
	fixtures := map[string]string{
		"CompoundTriggerConditions":      FixtureCompoundTriggerConditions,
		"ConditionalRouting":             FixtureConditionalRouting,
		"ConditionalTrigger":             FixtureConditionalTrigger,
		"StorylineBadgeGating":           FixtureStorylineBadgeGating,
		"StoryInterruption":              FixtureStoryInterruption,
		"OutboundWebhooks":               FixtureOutboundWebhooks,
		"PersistentLinks":                FixturePersistentLinks,
		"DeferredTransitions":            FixtureDeferredTransitions,
		"MailhogFullSequence":            FixtureMailhogFullSequence,
		"HybridTransitions":              FixtureHybridTransitions,
		"MultiStorylineEnactmentScene":   FixtureMultiStorylineEnactmentScene,
	}

	for name, src := range fixtures {
		t.Run(name, func(t *testing.T) {
			ResetIDCounter()
			result := CompileScript(src, "sub_e2e", bson.NewObjectId())
			for _, d := range result.Diagnostics {
				if d.Level == DiagError {
					t.Errorf("compile error: %s at %s", d.Message, d.Pos)
				}
			}
			if len(result.Stories) == 0 {
				t.Error("expected at least 1 story")
			}
			// Verify all story entities have valid IDs
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
					}
				}
			}
		})
	}
}
