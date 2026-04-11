package scripting

// Fixture scripts for testing the SentanylScript engine.
// Each fixture covers a specific automation pattern.
// Authoring-sugar fixtures (default sender, patterns, policies, links,
// scene ranges, data blocks, for loops) are at the bottom of this file.

// FixtureSimpleOneStoryline — minimal single-storyline campaign.
const FixtureSimpleOneStoryline = `
story "Simple Welcome" {
	priority 1
	storyline "Main" {
		order 1
		enactment "Welcome" {
			level 1
			order 1
			scene "Welcome Email" {
				subject "Welcome to our platform!"
				body "<h1>Welcome!</h1><p>We are glad you joined.</p>"
				from_email "hello@example.com"
				from_name "The Team"
				reply_to "support@example.com"
			}
		}
	}
}
`

// FixtureMultiStoryline — campaign with 3 storylines in sequence.
const FixtureMultiStoryline = `
story "Multi Storyline Campaign" {
	priority 5

	storyline "Awareness" {
		order 1
		enactment "Intro" {
			level 1
			scene "Intro Email" {
				subject "Did you know?"
				body "<h1>Discover something new</h1>"
				from_email "team@example.com"
			}
		}
	}

	storyline "Engagement" {
		order 2
		enactment "Engage" {
			level 1
			scene "Engage Email" {
				subject "Take the next step"
				body "<h1>Engage with us</h1>"
				from_email "team@example.com"
			}
		}
	}

	storyline "Conversion" {
		order 3
		enactment "Convert" {
			level 1
			scene "Offer Email" {
				subject "Special Offer Just For You"
				body "<h1>50% Off Today Only</h1>"
				from_email "team@example.com"
			}
		}
	}
}
`

// FixtureMultiEnactment — storyline with 3 enactments.
const FixtureMultiEnactment = `
story "Multi Enactment" {
	storyline "Funnel" {
		order 1
		enactment "Top of Funnel" {
			level 1
			order 1
			scene "TOF Email" {
				subject "Discover"
				body "Top of funnel content"
			}
		}
		enactment "Middle of Funnel" {
			level 2
			order 2
			scene "MOF Email" {
				subject "Learn More"
				body "Middle of funnel content"
			}
		}
		enactment "Bottom of Funnel" {
			level 3
			order 3
			scene "BOF Email" {
				subject "Buy Now"
				body "Bottom of funnel content"
			}
		}
	}
}
`

// FixtureMultiScene — enactment with 3 scenes (multi-scene enactment).
const FixtureMultiScene = `
story "Multi Scene" {
	storyline "Main" {
		order 1
		enactment "Drip Sequence" {
			level 1
			scene "Day 1" {
				subject "Day 1 - Getting Started"
				body "<h1>Welcome to Day 1</h1>"
				from_email "drip@example.com"
			}
			scene "Day 3" {
				subject "Day 3 - Quick Tip"
				body "<h1>Here is a tip for Day 3</h1>"
				from_email "drip@example.com"
			}
			scene "Day 7" {
				subject "Day 7 - Bonus Content"
				body "<h1>Bonus content for Day 7</h1>"
				from_email "drip@example.com"
			}
		}
	}
}
`

// FixtureConditionalBadgeRouting — routing by badge status.
const FixtureConditionalBadgeRouting = `
story "Badge Routing" {
	priority 10

	storyline "Qualification" {
		order 1
		on_complete {
			conditional_route {
				required_badges { must_have "vip" }
				next_storyline "VIP Track"
				priority 1
			}
			conditional_route {
				required_badges { must_not_have "vip" }
				next_storyline "Standard Track"
				priority 2
			}
		}
		enactment "Qualify" {
			level 1
			scene "Qualify Email" {
				subject "Tell us about yourself"
				body "Take our quiz"
				from_email "team@example.com"
			}
			on click "quiz_link" {
				do give_badge "vip"
				do mark_complete
			}
			on not_click "quiz_link" {
				within 2d
				do mark_complete
			}
		}
	}

	storyline "VIP Track" {
		order 2
		required_badges { must_have "vip" }
		enactment "VIP Offer" {
			level 1
			scene "VIP Email" {
				subject "Exclusive VIP Offer"
				body "50% off for VIPs"
				from_email "vip@example.com"
			}
		}
	}

	storyline "Standard Track" {
		order 3
		enactment "Standard Offer" {
			level 1
			scene "Standard Email" {
				subject "Check out our deals"
				body "20% off for everyone"
				from_email "deals@example.com"
			}
		}
	}
}
`

// FixtureClickBranching — click/not-click branching.
const FixtureClickBranching = `
story "Click Branch" {
	storyline "Main" {
		order 1
		enactment "Hook" {
			level 1
			scene "Promo Email" {
				subject "Check this out"
				body "<a href='https://example.com/offer'>Click here</a>"
				from_email "promo@example.com"
			}
			on click "https://example.com/offer" {
				within 1d
				do jump_to_enactment "Offer"
				do give_badge "clicked_promo"
			}
			on not_click "https://example.com/offer" {
				within 1d
				do next_scene
			}
		}
		enactment "Offer" {
			level 2
			scene "Offer Email" {
				subject "Here is your offer"
				body "Buy now"
				from_email "promo@example.com"
			}
		}
	}
}
`

// FixtureOpenBranching — open/not-open branching.
const FixtureOpenBranching = `
story "Open Branch" {
	storyline "Main" {
		order 1
		enactment "Hook" {
			level 1
			scene "Initial Email" {
				subject "Important Update"
				body "Read this important update"
				from_email "updates@example.com"
			}
			on open {
				do give_badge "engaged"
				do next_scene
			}
			on not_open {
				within 1d
				do retry_scene up_to 2
					else do mark_failed
			}
		}
	}
}
`

// FixtureBoundedRetry — retry with max count and fallback.
const FixtureBoundedRetry = `
story "Bounded Retry" {
	storyline "Main" {
		order 1
		enactment "Hook" {
			level 1
			scene "Email" {
				subject "Action Required"
				body "Please complete this action"
				from_email "team@example.com"
			}
			on not_open {
				within 1d
				do retry_scene up_to 3 times
					else do jump_to_enactment "Last Chance"
			}
		}
		enactment "Last Chance" {
			level 2
			scene "Final Email" {
				subject "Last Chance"
				body "This is your final reminder"
				from_email "team@example.com"
			}
			on not_open {
				within 2d
				do mark_failed
			}
		}
	}
}
`

// FixtureLoopToPriorEnactment — loop back to an earlier enactment.
const FixtureLoopToPriorEnactment = `
story "Loop Back" {
	storyline "Main" {
		order 1
		enactment "Introduction" {
			level 1
			order 1
			scene "Intro Email" {
				subject "Welcome"
				body "Introduction content"
				from_email "team@example.com"
			}
			on click "next_link" {
				do jump_to_enactment "Offer"
			}
		}
		enactment "Offer" {
			level 2
			order 2
			scene "Offer Email" {
				subject "Special Offer"
				body "Buy now"
				from_email "team@example.com"
			}
			on click "buy_link" {
				do mark_complete
			}
			on not_click "buy_link" {
				within 2d
				do loop_to_enactment "Introduction" up_to 2
					else do mark_failed
			}
		}
	}
}
`

// FixtureFailureFallback — failure path with badge transaction.
const FixtureFailureFallback = `
story "Failure Fallback" {
	on_fail {
		give_badge "campaign_failed"
	}
	storyline "Main" {
		order 1
		on_fail {
			give_badge "storyline_failed"
			next_storyline "Recovery"
		}
		enactment "Hook" {
			level 1
			scene "Email" {
				subject "Take Action"
				body "Act now"
				from_email "team@example.com"
			}
			on not_open {
				within 3d
				do mark_failed
			}
		}
	}
	storyline "Recovery" {
		order 2
		enactment "Recovery" {
			level 1
			scene "Recovery Email" {
				subject "We noticed you haven't engaged"
				body "Here is another chance"
				from_email "team@example.com"
			}
		}
	}
}
`

// FixtureCompletionPath — completion with badges and next story reference.
const FixtureCompletionPath = `
story "Completion Path" {
	on_begin {
		give_badge "started_onboarding"
	}
	on_complete {
		give_badge "completed_onboarding"
		remove_badge "started_onboarding"
	}
	storyline "Onboarding" {
		order 1
		on_complete {
			give_badge "onboarding_done"
		}
		enactment "Step 1" {
			level 1
			scene "Welcome" {
				subject "Welcome to Onboarding"
				body "Let us get started"
				from_email "team@example.com"
			}
			on click "start_link" {
				do mark_complete
				do advance_to_next_storyline
			}
		}
	}
	storyline "Advanced" {
		order 2
		required_badges { must_have "onboarding_done" }
		enactment "Advanced Step" {
			level 1
			scene "Advanced Email" {
				subject "Advanced Topics"
				body "Deep dive content"
				from_email "team@example.com"
			}
			on click "complete_link" {
				do mark_complete
			}
		}
	}
}
`

// FixtureFullCampaign — comprehensive campaign exercising all features.
const FixtureFullCampaign = `
story "Q4 Product Launch" {
	priority 10
	allow_interruption true

	on_begin {
		give_badge "q4_launch_entered"
	}
	on_complete {
		give_badge "q4_launch_completed"
		remove_badge "q4_launch_entered"
	}
	on_fail {
		give_badge "q4_launch_failed"
	}

	required_badges {
		must_not_have "q4_launch_completed"
	}

	start_trigger "eligible_for_q4"
	complete_trigger "q4_purchase_made"

	storyline "Awareness" {
		order 1
		on_begin {
			give_badge "awareness_started"
		}
		on_complete {
			give_badge "awareness_done"
			conditional_route {
				required_badges { must_have "high_intent" }
				next_storyline "Hard Sell"
				priority 1
			}
			conditional_route {
				required_badges { must_not_have "high_intent" }
				next_storyline "Soft Sell"
				priority 2
			}
		}

		enactment "Teaser" {
			level 1
			order 1
			scene "Teaser Email" {
				subject "Something big is coming..."
				body "<h1>Get ready</h1><p>Something exciting is on the way.</p>"
				from_email "launch@example.com"
				from_name "Launch Team"
				reply_to "support@example.com"
			}
			on open {
				do give_badge "engaged_teaser"
			}
			on click "teaser_link" {
				within 1d
				do give_badge "high_intent"
				do jump_to_enactment "Reveal"
			}
			on not_open {
				within 2d
				do retry_scene up_to 2 times
					else do next_scene
			}
		}

		enactment "Reveal" {
			level 2
			order 2
			scene "Reveal Email" {
				subject "The wait is over!"
				body "<h1>Introducing our new product</h1>"
				from_email "launch@example.com"
				from_name "Launch Team"
			}
			on click "product_link" {
				do mark_complete
				do give_badge "saw_product"
			}
			on not_click "product_link" {
				within 2d
				do mark_complete
			}
		}
	}

	storyline "Hard Sell" {
		order 2
		required_badges {
			must_have "high_intent"
		}
		enactment "Urgency" {
			level 1
			order 1
			scene "Urgency Email" {
				subject "Limited time offer - 48 hours left"
				body "<h1>Act now</h1><p>Only 48 hours to claim your discount.</p>"
				from_email "launch@example.com"
				from_name "Launch Team"
			}
			on click "buy_now" {
				do mark_complete
				do give_badge "purchased"
				do advance_to_next_storyline
				send_immediate true
			}
			on not_click "buy_now" {
				within 1d
				do loop_to_enactment "Urgency" up_to 1
					else do mark_failed
			}
		}
	}

	storyline "Soft Sell" {
		order 3
		enactment "Education" {
			level 1
			order 1
			scene "Education Email 1" {
				subject "Why our product matters"
				body "<h1>Learn why</h1>"
				from_email "launch@example.com"
			}
			scene "Education Email 2" {
				subject "How others are using it"
				body "<h1>Case studies</h1>"
				from_email "launch@example.com"
			}
			on click "learn_more" {
				do give_badge "educated"
				do jump_to_enactment "Gentle Ask"
			}
			on not_open {
				within 3d
				do retry_scene up_to 1
					else do mark_failed
			}
		}
		enactment "Gentle Ask" {
			level 2
			order 2
			scene "CTA Email" {
				subject "Ready to try it?"
				body "<h1>Start your free trial</h1>"
				from_email "launch@example.com"
			}
			on click "trial_link" {
				do mark_complete
				do give_badge "trial_started"
			}
			on not_click "trial_link" {
				within 3d
				do mark_failed
			}
			on unsubscribe {
				do unsubscribe
				do end_story
			}
		}
	}
}
`

// ========== Authoring-sugar fixtures ==========
// These demonstrate default sender, patterns, policies, links, scene ranges,
// data blocks, for loops, and enactment_defaults — all first-class language
// features, not a separate "version".

// FixtureCompactCampaign demonstrates compact authoring:
// default sender, links block, policy/pattern definitions, and use statements.
// Expresses "4 phases × 3 scenes each, same structure" in ~40 lines instead
// of manually repeating 12 nearly identical blocks.
const FixtureCompactCampaign = `
default sender {
	from_email "coach@demo.com"
	from_name "Manifesting Coach"
	reply_to "coach@demo.com"
}

links {
	more_info_a = "https://example.com/more-info-a"
	more_info_b = "https://example.com/more-info-b"
	buy_now_soft = "https://example.com/buy-now-soft"
	buy_now_hard = "https://example.com/buy-now-hard"
}

policy click_completes(link) {
	on click link {
		trigger_priority 1
		mark_complete true
		within 1m
	}
}

pattern three_scene_phase(name, subject_prefix, body_prefix, link_name) {
	enactment "${name}" {
		level 1

		scenes 1..3 as n {
			scene "Scene ${n}" {
				subject "${subject_prefix} (Email ${n}/3)"
				body "<h1>${body_prefix}</h1><p>Email ${n} of 3</p>"
			}
		}

		use policy click_completes(link_name)
	}
}

story "Manifesting Workshops Compact" {
	use sender default

	storyline "Manifesting 101" {
		order 1
		use pattern three_scene_phase("Enactment A", "[Manifesting 101] Soft Intrigue", "Manifesting 101 - Enactment A", more_info_a)
		use pattern three_scene_phase("Enactment B", "[Manifesting 101] Hard Intrigue", "Manifesting 101 - Enactment B", more_info_b)
		use pattern three_scene_phase("Enactment C", "[Manifesting 101] Soft Sell", "Manifesting 101 - Enactment C", buy_now_soft)
		use pattern three_scene_phase("Enactment D", "[Manifesting 101] Hard Sell", "Manifesting 101 - Enactment D", buy_now_hard)
	}
}
`

// FixtureDefaultSender demonstrates basic sender default inheritance.
const FixtureDefaultSender = `
default sender {
	from_email "hello@example.com"
	from_name "The Team"
	reply_to "support@example.com"
}

story "Simple with Defaults" {
	use sender default

	storyline "Main" {
		order 1
		enactment "Welcome" {
			level 1
			scene "Welcome Email" {
				subject "Welcome!"
				body "<h1>Welcome</h1>"
			}
		}
	}
}
`

// FixtureLinksAndPolicies demonstrates links + policy definitions.
const FixtureLinksAndPolicies = `
links {
	signup = "https://example.com/signup"
	upgrade = "https://example.com/upgrade"
}

policy click_to_complete(link) {
	on click link {
		trigger_priority 1
		mark_complete true
		within 1d
	}
}

story "Links and Policies Demo" {
	storyline "Main" {
		order 1
		enactment "Signup CTA" {
			level 1
			scene "Signup Email" {
				subject "Sign up today!"
				body "<h1>Join us</h1><a href='https://example.com/signup'>Sign up</a>"
				from_email "team@example.com"
			}
			use policy click_to_complete(signup)
		}
		enactment "Upgrade CTA" {
			level 2
			scene "Upgrade Email" {
				subject "Upgrade your plan"
				body "<h1>Go premium</h1><a href='https://example.com/upgrade'>Upgrade</a>"
				from_email "team@example.com"
			}
			use policy click_to_complete(upgrade)
		}
	}
}
`

// FixtureScenesRange demonstrates scenes 1..N generation.
const FixtureScenesRange = `
default sender {
	from_email "drip@example.com"
	from_name "Drip Team"
	reply_to "drip@example.com"
}

story "Drip Campaign" {
	use sender default

	storyline "Onboarding" {
		order 1
		enactment "Week 1 Drip" {
			level 1

			scenes 1..5 as day {
				scene "Day ${day}" {
					subject "Onboarding Day ${day}"
					body "<h1>Day ${day} Content</h1><p>Welcome to day ${day} of onboarding.</p>"
				}
			}
		}
	}
}
`

// FixturePatternReuse demonstrates pattern reuse across storylines.
const FixturePatternReuse = `
default sender {
	from_email "coach@demo.com"
	from_name "Coach"
	reply_to "coach@demo.com"
}

links {
	main_link = "https://example.com/main"
}

policy click_completes(link) {
	on click link {
		trigger_priority 1
		mark_complete true
		within 1d
	}
}

pattern intro_phase(name, prefix) {
	enactment "${name}" {
		level 1
		scenes 1..2 as n {
			scene "${name} Scene ${n}" {
				subject "${prefix} (${n}/2)"
				body "<p>${prefix} email ${n}</p>"
			}
		}
		use policy click_completes(main_link)
	}
}

story "Pattern Reuse Demo" {
	use sender default

	storyline "Track A" {
		order 1
		use pattern intro_phase("Track A Intro", "[Track A] Welcome")
	}

	storyline "Track B" {
		order 2
		use pattern intro_phase("Track B Intro", "[Track B] Welcome")
	}
}
`

// FixtureDataBlockForLoop demonstrates a data block with a for loop
// that generates enactments from structured data.
const FixtureDataBlockForLoop = `
default sender {
	from_email "coach@demo.com"
	from_name "Manifesting Coach"
	reply_to "coach@demo.com"
}

links {
	more_info_a = "https://example.com/more-info-a"
	more_info_b = "https://example.com/more-info-b"
	buy_now_soft = "https://example.com/buy-now-soft"
	buy_now_hard = "https://example.com/buy-now-hard"
}

policy click_completes(link) {
	on click link {
		trigger_priority 1
		mark_complete true
		within 1m
	}
}

data phases = [
	{ name: "A", subject: "Soft Intrigue", link: more_info_a },
	{ name: "B", subject: "Hard Intrigue", link: more_info_b },
	{ name: "C", subject: "Soft Sell", link: buy_now_soft },
	{ name: "D", subject: "Hard Sell", link: buy_now_hard }
]

pattern three_scene_phase(name, subject_prefix, body_prefix, link_name) {
	enactment name {
		scenes 1..3 as n {
			scene "Scene ${n}" {
				subject "${subject_prefix} (Email ${n}/3)"
				body "<h1>${body_prefix}</h1><p>Email ${n} of 3</p>"
			}
		}
		use policy click_completes(link_name)
	}
}

story "Generative Campaign" {
	use sender default

	storyline "Manifesting 101" {
		order 1

		for phase in phases {
			use pattern three_scene_phase(
				"Enactment ${phase.name}",
				"[Manifesting 101] ${phase.subject}",
				"Manifesting 101 - Enactment ${phase.name}",
				phase.link
			)
		}
	}
}
`

// FixtureStorylineGeneration demonstrates generating entire storylines
// from a data block using a for loop.
const FixtureStorylineGeneration = `
default sender {
	from_email "coach@demo.com"
	from_name "Coach"
	reply_to "coach@demo.com"
}

links {
	more_info_a = "https://example.com/more-info-a"
	buy_now = "https://example.com/buy-now"
}

policy click_completes(link) {
	on click link {
		trigger_priority 1
		mark_complete true
		within 1m
	}
}

pattern intro_enactment(name, prefix, link_name) {
	enactment name {
		scenes 1..2 as n {
			scene "Email ${n}" {
				subject "${prefix} Email ${n}"
				body "<p>${prefix} content ${n}</p>"
			}
		}
		use policy click_completes(link_name)
	}
}

data tracks = [
	{ name: "Manifesting 101", prefix: "[M101]" },
	{ name: "Advanced Attraction", prefix: "[AA]" },
	{ name: "Quantum Wealth", prefix: "[QW]" }
]

story "Multi-Track Campaign" {
	use sender default

	for track in tracks {
		storyline "${track.name}" {
			use pattern intro_enactment(
				"Introduction",
				"${track.prefix} Welcome",
				more_info_a
			)
			use pattern intro_enactment(
				"Sell Phase",
				"${track.prefix} Offer",
				buy_now
			)
		}
	}
}
`

// FixtureInlineDataLoop demonstrates for loops with inline data (no separate data block).
const FixtureInlineDataLoop = `
default sender {
	from_email "team@example.com"
	from_name "Team"
	reply_to "team@example.com"
}

story "Inline Loop Demo" {
	use sender default

	storyline "Main" {
		order 1

		for phase in [
			{ name: "Hook", subject: "Attention Grabber" },
			{ name: "Value", subject: "Value Proposition" },
			{ name: "Close", subject: "Final Offer" }
		] {
			enactment "Enactment ${phase.name}" {
				scene "Email" {
					subject "${phase.subject}"
					body "<p>${phase.name} content</p>"
				}
			}
		}
	}
}
`

// FixtureEnactmentDefaults demonstrates enactment_defaults applied globally.
const FixtureEnactmentDefaults = `
default sender {
	from_email "team@example.com"
	from_name "Team"
	reply_to "team@example.com"
}

links {
	main_link = "https://example.com/main"
}

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

story "Defaults Demo" {
	use sender default

	storyline "Main" {
		order 1

		enactment "Phase 1" {
			level 1
			scene "Welcome" {
				subject "Welcome!"
				body "<p>Welcome aboard</p>"
			}
		}

		enactment "Phase 2" {
			level 2
			scene "Follow Up" {
				subject "Follow Up"
				body "<p>Following up</p>"
			}
		}
	}
}
`

// FixtureFullGenerativeCampaign demonstrates the complete generative system:
// data blocks, for loops generating storylines and enactments, patterns, policies, defaults.
const FixtureFullGenerativeCampaign = `
default sender {
	from_email "coach@demo.com"
	from_name "Manifesting Coach"
	reply_to "coach@demo.com"
}

links {
	more_info_a = "https://example.com/more-info-a"
	more_info_b = "https://example.com/more-info-b"
	buy_now_soft = "https://example.com/buy-now-soft"
	buy_now_hard = "https://example.com/buy-now-hard"
}

policy click_completes(link) {
	on click link {
		trigger_priority 1
		mark_complete true
		within 1m
	}
}

data phases = [
	{ name: "A", subject: "Soft Intrigue", link: more_info_a },
	{ name: "B", subject: "Hard Intrigue", link: more_info_b },
	{ name: "C", subject: "Soft Sell", link: buy_now_soft },
	{ name: "D", subject: "Hard Sell", link: buy_now_hard }
]

data tracks = [
	{ name: "Manifesting 101", prefix: "[M101]" },
	{ name: "Advanced Attraction", prefix: "[AA]" },
	{ name: "Quantum Wealth", prefix: "[QW]" }
]

pattern three_scene_phase(name, subject_prefix, body_prefix, link_name) {
	enactment name {
		scenes 1..3 as n {
			scene "Scene ${n}" {
				subject "${subject_prefix} (Email ${n}/3)"
				body "<h1>${body_prefix}</h1><p>Email ${n} of 3</p>"
			}
		}
		use policy click_completes(link_name)
	}
}

story "Manifesting Workshops Complete Bundle" {
	use sender default

	for track in tracks {
		storyline "${track.name}" {
			for phase in phases {
				use pattern three_scene_phase(
					"Enactment ${phase.name}",
					"${track.prefix} ${phase.subject}",
					"${track.name} - Enactment ${phase.name}",
					phase.link
				)
			}
		}
	}
}
`

// FixtureDotAccessTriggers demonstrates dot-access references (var.field) in
// trigger values inside for loops, integer data values, and dot-access order/level.
const FixtureDotAccessTriggers = `
# This fixture tests dot-access in triggers, # comments, integer data values, and dot-access order
default sender {
	from_email "coach@demo.com"
	from_name "Coach"
	reply_to "coach@demo.com"
}

links {
	w1_info = "https://example.com/wealth/info"
	w1_buy  = "https://example.com/wealth/buy"
	w2_info = "https://example.com/love/info"
	w2_buy  = "https://example.com/love/buy"
}

data workshops = [
	{ name: "Wealth", order: 1, info: w1_info, buy: w1_buy },
	{ name: "Love",   order: 2, info: w2_info, buy: w2_buy }
]

story "Workshop Campaign" {
	use sender default

	for ws in workshops {
		storyline "${ws.name} Track" {
			order ws.order
			on_complete { give_badge "${ws.name}_done" }

			enactment "Intrigue" {
				level 1
				scene "Email 1" {
					subject "Learn about ${ws.name}"
					body "<p>Get more info</p>"
				}
				# dot-access in click trigger
				on click ws.info {
					trigger_priority 1
					mark_complete true
					within 4m
					do jump_to_enactment "Sell"
				}
				on not_click ws.info {
					within 4m
					do mark_failed
				}
			}

			enactment "Sell" {
				level 2
				scene "Buy Email" {
					subject "Buy ${ws.name} now"
					body "<p>Buy now!</p>"
				}
				on click ws.buy {
					trigger_priority 1
					mark_complete true
					within 4m
					do advance_to_next_storyline
				}
				on not_click ws.buy {
					within 4m
					do advance_to_next_storyline
				}
			}
		}
	}
}
`
