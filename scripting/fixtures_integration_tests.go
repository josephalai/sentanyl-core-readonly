package scripting

import _ "embed"

// Individual Feature Integration Test Scripts

//go:embed fixtures/integration_tests/01_story_lifecycle.ss
var IntegrationTestStoryLifecycle string

//go:embed fixtures/integration_tests/02_storyline_badge_gating.ss
var IntegrationTestStorylineBadgeGating string

//go:embed fixtures/integration_tests/03_trigger_types.ss
var IntegrationTestTriggerTypes string

//go:embed fixtures/integration_tests/04_action_types.ss
var IntegrationTestActionTypes string

//go:embed fixtures/integration_tests/05_conditional_routing.ss
var IntegrationTestConditionalRouting string

//go:embed fixtures/integration_tests/06_click_branching.ss
var IntegrationTestClickBranching string

//go:embed fixtures/integration_tests/07_retry_and_loops.ss
var IntegrationTestRetryAndLoops string

//go:embed fixtures/integration_tests/08_story_interruption.ss
var IntegrationTestStoryInterruption string

//go:embed fixtures/integration_tests/09_deferred_transitions.ss
var IntegrationTestDeferredTransitions string

//go:embed fixtures/integration_tests/10_persistent_links.ss
var IntegrationTestPersistentLinks string

//go:embed fixtures/integration_tests/11_scene_template_vars.ss
var IntegrationTestSceneTemplateVars string

//go:embed fixtures/integration_tests/12_scene_defaults.ss
var IntegrationTestSceneDefaults string

//go:embed fixtures/integration_tests/13_enactment_defaults.ss
var IntegrationTestEnactmentDefaults string

//go:embed fixtures/integration_tests/14_default_sender.ss
var IntegrationTestDefaultSender string

//go:embed fixtures/integration_tests/15_links_and_policies.ss
var IntegrationTestLinksAndPolicies string

//go:embed fixtures/integration_tests/16_pattern_reuse.ss
var IntegrationTestPatternReuse string

//go:embed fixtures/integration_tests/17_data_blocks_for_loops.ss
var IntegrationTestDataBlocksForLoops string

//go:embed fixtures/integration_tests/18_scenes_range.ss
var IntegrationTestScenesRange string

//go:embed fixtures/integration_tests/19_multi_story.ss
var IntegrationTestMultiStory string

//go:embed fixtures/integration_tests/20_next_story_hopping.ss
var IntegrationTestNextStoryHopping string

//go:embed fixtures/integration_tests/21_storyline_on_fail_routes.ss
var IntegrationTestStorylineOnFailRoutes string

//go:embed fixtures/integration_tests/22_badge_integration.ss
var IntegrationTestBadgeIntegration string

//go:embed fixtures/integration_tests/23_condition_guards.ss
var IntegrationTestConditionGuards string

//go:embed fixtures/integration_tests/24_multi_scene_drip.ss
var IntegrationTestMultiSceneDrip string

//go:embed fixtures/integration_tests/25_skip_storyline_expiry.ss
var IntegrationTestSkipStorylineExpiry string

//go:embed fixtures/integration_tests/26_dot_access.ss
var IntegrationTestDotAccess string

//go:embed fixtures/integration_tests/27_compound_conditions.ss
var IntegrationTestCompoundConditions string

// Combo Integration Test Scripts

//go:embed fixtures/integration_tests/combo_ecommerce_funnel.ss
var IntegrationTestComboEcommerceFunnel string

//go:embed fixtures/integration_tests/combo_saas_onboarding.ss
var IntegrationTestComboSaaSOnboarding string

//go:embed fixtures/integration_tests/combo_newsletter_engagement.ss
var IntegrationTestComboNewsletterEngagement string

//go:embed fixtures/integration_tests/combo_event_promotion.ss
var IntegrationTestComboEventPromotion string

// Funnel Feature Integration Test Scripts

//go:embed fixtures/integration_tests/37_funnel_with_video.ss
var IntegrationTestFunnelWithVideo string

//go:embed fixtures/integration_tests/38_checkout_purchase.ss
var IntegrationTestCheckoutPurchase string

//go:embed fixtures/integration_tests/39_full_pipeline.ss
var IntegrationTestFullPipeline string

//go:embed fixtures/integration_tests/40_video_watch_operators.ss
var IntegrationTestVideoWatchOperators string

//go:embed fixtures/integration_tests/41_lead_magnet_funnel.ss
var IntegrationTestLeadMagnetFunnel string

//go:embed fixtures/integration_tests/42_webinar_funnel.ss
var IntegrationTestWebinarFunnel string

//go:embed fixtures/integration_tests/43_product_launch_funnel.ss
var IntegrationTestProductLaunchFunnel string

//go:embed fixtures/integration_tests/44_multi_route_membership.ss
var IntegrationTestMultiRouteMembership string

//go:embed fixtures/integration_tests/45_upsell_pipeline.ss
var IntegrationTestUpsellPipeline string

//go:embed fixtures/integration_tests/46_abandon_recovery.ss
var IntegrationTestAbandonRecovery string

// Website / Site Feature Integration Test Scripts

//go:embed fixtures/integration_tests/50_global_website.ss
var IntegrationTestGlobalWebsite string

// LMS Feature Integration Test Scripts

//go:embed fixtures/integration_tests/51_lms_modules.ss
var IntegrationTestLMSModules string

// Video Intelligence Feature Integration Test Scripts

//go:embed fixtures/integration_tests/52_video_intelligence.ss
var IntegrationTestVideoIntelligence string

// IntegrationTestScript holds metadata and source for an integration test script.
type IntegrationTestScript struct {
	ID          string
	Name        string
	Description string
	Category    string // "feature" or "combo"
	Source      string
}

// IntegrationTestScripts returns all integration test scripts with metadata.
func IntegrationTestScripts() []IntegrationTestScript {
	return []IntegrationTestScript{
		{
			ID:          "01_story_lifecycle",
			Name:        "Story Lifecycle",
			Description: "Tests full story lifecycle including creation, activation, scene transitions, and completion",
			Category:    "feature",
			Source:      IntegrationTestStoryLifecycle,
		},
		{
			ID:          "02_storyline_badge_gating",
			Name:        "Storyline Badge Gating",
			Description: "Tests gating storyline entry on badge ownership so only qualified contacts proceed",
			Category:    "feature",
			Source:      IntegrationTestStorylineBadgeGating,
		},
		{
			ID:          "03_trigger_types",
			Name:        "Trigger Types",
			Description: "Tests various trigger types including event, schedule, and webhook triggers",
			Category:    "feature",
			Source:      IntegrationTestTriggerTypes,
		},
		{
			ID:          "04_action_types",
			Name:        "Action Types",
			Description: "Tests different action types such as send email, send SMS, set badge, and webhook calls",
			Category:    "feature",
			Source:      IntegrationTestActionTypes,
		},
		{
			ID:          "05_conditional_routing",
			Name:        "Conditional Routing",
			Description: "Tests conditional scene routing based on contact attributes and event data",
			Category:    "feature",
			Source:      IntegrationTestConditionalRouting,
		},
		{
			ID:          "06_click_branching",
			Name:        "Click Branching",
			Description: "Tests branching logic based on link clicks within sent messages",
			Category:    "feature",
			Source:      IntegrationTestClickBranching,
		},
		{
			ID:          "07_retry_and_loops",
			Name:        "Retry and Loops",
			Description: "Tests retry mechanisms and looping scene transitions for repeated engagement",
			Category:    "feature",
			Source:      IntegrationTestRetryAndLoops,
		},
		{
			ID:          "08_story_interruption",
			Name:        "Story Interruption",
			Description: "Tests interrupting an active story mid-flow to redirect contacts to a different path",
			Category:    "feature",
			Source:      IntegrationTestStoryInterruption,
		},
		{
			ID:          "09_deferred_transitions",
			Name:        "Deferred Transitions",
			Description: "Tests delayed scene transitions using wait periods before advancing",
			Category:    "feature",
			Source:      IntegrationTestDeferredTransitions,
		},
		{
			ID:          "10_persistent_links",
			Name:        "Persistent Links",
			Description: "Tests persistent link tracking across scenes and story completions",
			Category:    "feature",
			Source:      IntegrationTestPersistentLinks,
		},
		{
			ID:          "11_scene_template_vars",
			Name:        "Scene Template Variables",
			Description: "Tests template variable interpolation within scene content",
			Category:    "feature",
			Source:      IntegrationTestSceneTemplateVars,
		},
		{
			ID:          "12_scene_defaults",
			Name:        "Scene Defaults",
			Description: "Tests default values applied to scenes when explicit values are not provided",
			Category:    "feature",
			Source:      IntegrationTestSceneDefaults,
		},
		{
			ID:          "13_enactment_defaults",
			Name:        "Enactment Defaults",
			Description: "Tests default enactment configuration inherited by scenes",
			Category:    "feature",
			Source:      IntegrationTestEnactmentDefaults,
		},
		{
			ID:          "14_default_sender",
			Name:        "Default Sender",
			Description: "Tests default sender identity applied to outbound messages",
			Category:    "feature",
			Source:      IntegrationTestDefaultSender,
		},
		{
			ID:          "15_links_and_policies",
			Name:        "Links and Policies",
			Description: "Tests link configuration and policy enforcement on story actions",
			Category:    "feature",
			Source:      IntegrationTestLinksAndPolicies,
		},
		{
			ID:          "16_pattern_reuse",
			Name:        "Pattern Reuse",
			Description: "Tests reusable scene patterns shared across multiple stories",
			Category:    "feature",
			Source:      IntegrationTestPatternReuse,
		},
		{
			ID:          "17_data_blocks_for_loops",
			Name:        "Data Blocks for Loops",
			Description: "Tests data block iteration for generating repeated scene content from data sets",
			Category:    "feature",
			Source:      IntegrationTestDataBlocksForLoops,
		},
		{
			ID:          "18_scenes_range",
			Name:        "Scenes Range",
			Description: "Tests range-based scene generation for dynamically creating scene sequences",
			Category:    "feature",
			Source:      IntegrationTestScenesRange,
		},
		{
			ID:          "19_multi_story",
			Name:        "Multi Story",
			Description: "Tests defining and running multiple stories within a single script",
			Category:    "feature",
			Source:      IntegrationTestMultiStory,
		},
		{
			ID:          "20_next_story_hopping",
			Name:        "Next Story Hopping",
			Description: "Tests chaining stories so completion of one triggers the next",
			Category:    "feature",
			Source:      IntegrationTestNextStoryHopping,
		},
		{
			ID:          "21_storyline_on_fail_routes",
			Name:        "Storyline On-Fail Routes",
			Description: "Tests fallback routing when a storyline fails or a scene errors out",
			Category:    "feature",
			Source:      IntegrationTestStorylineOnFailRoutes,
		},
		{
			ID:          "22_badge_integration",
			Name:        "Badge Integration",
			Description: "Tests badge granting, revoking, and checking within story flows",
			Category:    "feature",
			Source:      IntegrationTestBadgeIntegration,
		},
		{
			ID:          "23_condition_guards",
			Name:        "Condition Guards",
			Description: "Tests guard conditions that gate scene entry based on runtime state",
			Category:    "feature",
			Source:      IntegrationTestConditionGuards,
		},
		{
			ID:          "24_multi_scene_drip",
			Name:        "Multi-Scene Drip",
			Description: "Tests drip-style delivery across multiple scenes with timed intervals",
			Category:    "feature",
			Source:      IntegrationTestMultiSceneDrip,
		},
		{
			ID:          "25_skip_storyline_expiry",
			Name:        "Skip Storyline Expiry",
			Description: "Tests skipping storylines that have expired based on time constraints",
			Category:    "feature",
			Source:      IntegrationTestSkipStorylineExpiry,
		},
		{
			ID:          "26_dot_access",
			Name:        "Dot Access",
			Description: "Tests dot-notation access for nested contact and event data fields",
			Category:    "feature",
			Source:      IntegrationTestDotAccess,
		},
		{
			ID:          "27_compound_conditions",
			Name:        "Compound Conditions",
			Description: "Tests compound conditional expressions combining multiple predicates with AND/OR logic",
			Category:    "feature",
			Source:      IntegrationTestCompoundConditions,
		},
		{
			ID:          "combo_ecommerce_funnel",
			Name:        "Combo: E-commerce Funnel",
			Description: "End-to-end e-commerce funnel combining triggers, conditions, drip sequences, and badges",
			Category:    "combo",
			Source:      IntegrationTestComboEcommerceFunnel,
		},
		{
			ID:          "combo_saas_onboarding",
			Name:        "Combo: SaaS Onboarding",
			Description: "End-to-end SaaS onboarding flow with multi-story chaining, badge gating, and conditional routing",
			Category:    "combo",
			Source:      IntegrationTestComboSaaSOnboarding,
		},
		{
			ID:          "combo_newsletter_engagement",
			Name:        "Combo: Newsletter Engagement",
			Description: "End-to-end newsletter engagement campaign with click branching, retries, and engagement scoring",
			Category:    "combo",
			Source:      IntegrationTestComboNewsletterEngagement,
		},
		{
			ID:          "combo_event_promotion",
			Name:        "Combo: Event Promotion",
			Description: "End-to-end event promotion workflow with deferred transitions, reminders, and follow-up sequences",
			Category:    "combo",
			Source:      IntegrationTestComboEventPromotion,
		},
		{
			ID:          "37_funnel_with_video",
			Name:        "Funnel with Video Tracking",
			Description: "Tests video block type, source_url, autoplay, and on watch triggers with threshold operators",
			Category:    "feature",
			Source:      IntegrationTestFunnelWithVideo,
		},
		{
			ID:          "38_checkout_purchase",
			Name:        "Checkout and Purchase Flow",
			Description: "Tests checkout form type, product_id, on purchase trigger, and post-purchase story",
			Category:    "feature",
			Source:      IntegrationTestCheckoutPurchase,
		},
		{
			ID:          "39_full_pipeline",
			Name:        "Full Pipeline",
			Description: "End-to-end test: funnel with video, lead capture, checkout, email stories, badge-gated routes",
			Category:    "combo",
			Source:      IntegrationTestFullPipeline,
		},
		{
			ID:          "40_video_watch_operators",
			Name:        "Video Watch Operators",
			Description: "Tests all four watch trigger operators: >, <, >=, <= with different thresholds",
			Category:    "feature",
			Source:      IntegrationTestVideoWatchOperators,
		},
		{
			ID:          "41_lead_magnet_funnel",
			Name:        "Lead Magnet Funnel",
			Description: "Classic lead magnet: opt-in page → download delivery → email nurture drip. Tests funnel + story combo with jump_to_stage and start_story.",
			Category:    "combo",
			Source:      IntegrationTestLeadMagnetFunnel,
		},
		{
			ID:          "42_webinar_funnel",
			Name:        "Webinar Registration Funnel",
			Description: "Webinar funnel: registration → confirmation → replay with video tracking. Tests badge-gated replay route and checkout form.",
			Category:    "combo",
			Source:      IntegrationTestWebinarFunnel,
		},
		{
			ID:          "43_product_launch_funnel",
			Name:        "Product Launch Funnel (PLF-Style)",
			Description: "Product Launch Formula: 3 video series → cart open → checkout. Tests multiple video blocks, badge accumulation, sequential stages.",
			Category:    "combo",
			Source:      IntegrationTestProductLaunchFunnel,
		},
		{
			ID:          "44_multi_route_membership",
			Name:        "Multi-Route Membership Site",
			Description: "Membership with badge-gated routes: public → free → paid. Tests must_have_badge/must_not_have_badge routing with progressive upgrade.",
			Category:    "combo",
			Source:      IntegrationTestMultiRouteMembership,
		},
		{
			ID:          "45_upsell_pipeline",
			Name:        "Upsell Pipeline",
			Description: "Multi-stage upsell: main offer → order bump → OTO. Tests sequential checkout stages, video in offers, and badge-gated routing.",
			Category:    "combo",
			Source:      IntegrationTestUpsellPipeline,
		},
		{
			ID:          "46_abandon_recovery",
			Name:        "Cart Abandon Recovery",
			Description: "Abandon recovery: sales page with video → on abandon triggers recovery email drip. Tests abandon trigger, badge removal on purchase.",
			Category:    "combo",
			Source:      IntegrationTestAbandonRecovery,
		},
		{
			ID:          "50_global_website",
			Name:        "Global Website with SEO",
			Description: "Tests Site declaration with domain, theme, SEO metadata, navigation header/footer, and page blocks.",
			Category:    "feature",
			Source:      IntegrationTestGlobalWebsite,
		},
		{
			ID:          "51_lms_modules",
			Name:        "LMS Modules and Lessons",
			Description: "Tests Product expansion with nested Module/Lesson structure, video URLs, content HTML, and draft status.",
			Category:    "feature",
			Source:      IntegrationTestLMSModules,
		},
		{
			ID:          "52_video_intelligence",
			Name:        "Video Intelligence Full Pipeline",
			Description: "Tests first-class media entities, player presets, channels, badge rules, video triggers, and media-funnel integration.",
			Category:    "combo",
			Source:      IntegrationTestVideoIntelligence,
		},
	}
}
