package scripting

// Fixture scripts converted from the e2e shell scripts in scripts/*.sh.
// Each fixture can be compiled with CompileScript to produce the same entity graph
// that the shell script would create via sequential API calls.

// FixtureCompoundTriggerConditions — from e2e-compound-trigger-conditions.sh
// Demonstrates multi-badge AND gating on triggers: a single trigger with
// required_badges { must_have "vip" must_have "verified" } requires ALL badges.
// Three triggers with priorities 3/2/1 route to different enactment tiers.
const FixtureCompoundTriggerConditions = `
story "Elite Member Campaign" {
	priority 1
	allow_interruption false
	start_trigger "story-start"

	storyline "Elite Campaign SL1" {
		order 1

		enactment "Elite Campaign Email" {
			level 1
			order 1

			scene "Elite Campaign Scene" {
				subject "🏆 Your Elite Campaign Invitation"
				body "<html><body><h2>🏆 Elite Campaign</h2><p>Choose your tier:</p><a href='https://example.com/elite-action'>Activate Elite</a> <a href='https://example.com/verified-action'>Activate Verified</a> <a href='https://example.com/standard-action'>Activate Standard</a></body></html>"
				from_email "creator@example.com"
				from_name "Elite Campaign"
				reply_to "creator@example.com"
			}

			on click "https://example.com/elite-action" {
				trigger_priority 3
				required_badges {
					must_have "vip"
					must_have "verified"
				}
				do jump_to_enactment "Elite Tier Result"
			}

			on click "https://example.com/verified-action" {
				trigger_priority 2
				required_badges {
					must_have "verified"
				}
				do jump_to_enactment "Verified Tier Result"
			}

			on click "https://example.com/standard-action" {
				trigger_priority 1
				do jump_to_enactment "Standard Tier Result"
			}
		}

		enactment "Elite Tier Result" {
			level 2
			order 2

			scene "Elite Result Scene" {
				subject "💎 [ELITE TIER] Compound Condition Passed — Welcome!"
				body "<html><body><h2>💎 Elite Tier</h2><p>Compound AND condition passed: both vip AND verified badges present.</p></body></html>"
				from_email "creator@example.com"
				from_name "Elite Campaign"
				reply_to "creator@example.com"
			}
		}

		enactment "Verified Tier Result" {
			level 3
			order 3

			scene "Verified Result Scene" {
				subject "✅ [VERIFIED TIER] Verified Condition Passed — Welcome!"
				body "<html><body><h2>✅ Verified Tier</h2><p>Verified badge check passed but vip was missing.</p></body></html>"
				from_email "creator@example.com"
				from_name "Elite Campaign"
				reply_to "creator@example.com"
			}
		}

		enactment "Standard Tier Result" {
			level 4
			order 4

			scene "Standard Result Scene" {
				subject "📌 [STANDARD TIER] Standard Path Confirmed — Welcome!"
				body "<html><body><h2>📌 Standard Tier</h2><p>No badge requirements — standard fallback path.</p></body></html>"
				from_email "creator@example.com"
				from_name "Elite Campaign"
				reply_to "creator@example.com"
			}
		}
	}
}
`

// FixtureConditionalRouting — from e2e-conditional-routing.sh
// Demonstrates on_complete conditional routes on a storyline: after SL1 completes,
// the engine evaluates conditional_routes by priority. If user has "premium-member"
// badge, route to SL-PREMIUM (priority 10). Otherwise, fall through to SL-STANDARD (priority 1).
const FixtureConditionalRouting = `
story "Premium Learning Path" {
	priority 1
	allow_interruption false
	start_trigger "story-start"

	storyline "SL1 — Introduction" {
		order 1

		on_complete {
			conditional_route {
				priority 10
				required_badges { must_have "premium-member" }
				next_storyline "SL-PREMIUM — Advanced Module"
			}
			conditional_route {
				priority 1
				next_storyline "SL-STANDARD — Standard Module"
			}
		}

		enactment "SL1 — Introduction" {
			level 1
			order 1

			scene "Introduction Scene" {
				subject "[SL1] Your Learning Path Introduction"
				body "<html><body><h2>Welcome to Your Learning Path</h2><p>Click below to continue. You will be routed based on your badge status.</p><a href='https://example.com/complete-intro'>Complete Introduction</a></body></html>"
				from_email "creator@example.com"
				from_name "Learning Platform"
				reply_to "creator@example.com"
			}

			on click "https://example.com/complete-intro" {
				trigger_priority 1
				do advance_to_next_storyline
			}
		}
	}

	storyline "SL-PREMIUM — Advanced Module" {
		order 2

		enactment "SL-PREMIUM Enactment" {
			level 1
			order 1

			scene "Premium Module Welcome" {
				subject "✅ [PREMIUM PATH] Welcome to the Advanced Module!"
				body "<html><body><h2>✅ Premium Module</h2><p>You were routed here because you have the premium-member badge.</p></body></html>"
				from_email "creator@example.com"
				from_name "Learning Platform"
				reply_to "creator@example.com"
			}
		}
	}

	storyline "SL-STANDARD — Standard Module" {
		order 3

		enactment "SL-STANDARD Enactment" {
			level 1
			order 1

			scene "Standard Module Welcome" {
				subject "📚 [STANDARD PATH] Welcome to the Standard Module!"
				body "<html><body><h2>📚 Standard Module</h2><p>You were routed here as the default path.</p></body></html>"
				from_email "creator@example.com"
				from_name "Learning Platform"
				reply_to "creator@example.com"
			}
		}
	}
}
`

// FixtureConditionalTrigger — from e2e-conditional-trigger.sh
// Demonstrates badge-gated triggers AND storyline entry gating.
// Trigger A (priority 10) requires vip-member badge.
// Trigger B (priority 5) also requires vip-member.
// Trigger C (priority 1) has no badge requirement — fallback.
// SL-VIP requires must_have vip-member. SL-STD requires must_not_have vip-member.
const FixtureConditionalTrigger = `
story "VIP Membership Campaign" {
	priority 1
	allow_interruption false
	start_trigger "story-start"

	storyline "VIP Campaign — Storyline 1" {
		order 1

		enactment "VIP Campaign Email" {
			level 1
			order 1

			scene "VIP Campaign Scene" {
				subject "🌟 Your Exclusive VIP Invitation"
				body "<html><body><h2>🌟 VIP Campaign</h2><p>Choose your path:</p><a href='https://example.com/vip-path'>VIP Path</a> <a href='https://example.com/standard-path'>Standard Path</a></body></html>"
				from_email "creator@example.com"
				from_name "VIP Campaign"
				reply_to "creator@example.com"
			}

			on click "https://example.com/vip-path" {
				trigger_priority 10
				required_badges { must_have "vip-member" }
				do advance_to_next_storyline
			}

			on click "https://example.com/standard-path" {
				trigger_priority 5
				required_badges { must_have "vip-member" }
				do advance_to_next_storyline
			}

			on click "https://example.com/standard-path" {
				trigger_priority 1
				do advance_to_next_storyline
			}
		}
	}

	storyline "VIP Response Storyline" {
		order 2
		required_badges { must_have "vip-member" }

		enactment "VIP Confirmation Email" {
			level 1
			order 1

			scene "VIP Confirmation Scene" {
				subject "💎 Welcome to VIP — Confirm Your Membership"
				body "<html><body><h2>💎 VIP Confirmation</h2><p>Click to confirm your VIP membership.</p><a href='https://example.com/vip-complete'>Confirm VIP</a></body></html>"
				from_email "creator@example.com"
				from_name "VIP Campaign"
				reply_to "creator@example.com"
			}

			on click "https://example.com/vip-complete" {
				trigger_priority 1
				mark_complete true
			}
		}
	}

	storyline "Standard Response Storyline" {
		order 3
		required_badges { must_not_have "vip-member" }

		enactment "Standard Enrollment Email" {
			level 1
			order 1

			scene "Standard Enrollment Scene" {
				subject "✅ Welcome — Confirm Your Standard Enrollment"
				body "<html><body><h2>✅ Standard Enrollment</h2><p>Click to confirm your standard enrollment.</p><a href='https://example.com/std-complete'>Confirm Standard</a></body></html>"
				from_email "creator@example.com"
				from_name "VIP Campaign"
				reply_to "creator@example.com"
			}

			on click "https://example.com/std-complete" {
				trigger_priority 1
				mark_complete true
			}
		}
	}
}
`

// FixtureStorylineBadgeGating — from e2e-storyline-badge-gating.sh
// Demonstrates storyline-level badge gating with required_badges on SL2.
// SL1 → everyone enters. SL2 → requires "advanced-learner" badge. SL3 → no requirement.
// AdvanceToNextStoryline skips SL2 if user lacks badge, going directly to SL3.
const FixtureStorylineBadgeGating = `
story "Adaptive Learning Path" {
	priority 1
	allow_interruption false
	start_trigger "story-start"

	storyline "SL1 — Introduction" {
		order 1

		enactment "SL1 — Introduction" {
			level 1
			order 1

			scene "Introduction Scene" {
				subject "[SL1] Introduction to the Adaptive Learning Path"
				body "<html><body><h2>Welcome to the Adaptive Learning Path</h2><p>Click to advance. If you have the advanced-learner badge, you will enter the Advanced Module next.</p><a href='https://example.com/complete-sl1'>Complete Introduction</a></body></html>"
				from_email "creator@example.com"
				from_name "Adaptive Learning"
				reply_to "creator@example.com"
			}

			on click "https://example.com/complete-sl1" {
				trigger_priority 1
				do advance_to_next_storyline
			}
		}
	}

	storyline "SL2 — Advanced Module" {
		order 2
		required_badges { must_have "advanced-learner" }

		enactment "SL2 — Advanced Module" {
			level 1
			order 1

			scene "Advanced Module Scene" {
				subject "[SL2 — ADVANCED] You Qualified for the Advanced Module!"
				body "<html><body><h2>Advanced Module</h2><p>You qualified because you have the advanced-learner badge.</p></body></html>"
				from_email "creator@example.com"
				from_name "Adaptive Learning"
				reply_to "creator@example.com"
			}
		}
	}

	storyline "SL3 — Conclusion" {
		order 3

		enactment "SL3 — Conclusion" {
			level 1
			order 1

			scene "Conclusion Scene" {
				subject "[SL3 — CONCLUSION] Course Complete!"
				body "<html><body><h2>Course Complete!</h2><p>You have completed the adaptive learning path.</p></body></html>"
				from_email "creator@example.com"
				from_name "Adaptive Learning"
				reply_to "creator@example.com"
			}
		}
	}
}
`

// FixtureStoryInterruption — from e2e-story-interruption.sh
// Demonstrates story priority and allow_interruption. Newsletter (priority 1, allow_interruption true)
// can be paused by Cart Recovery (priority 10, allow_interruption false).
// Two separate stories in one script.
const FixtureStoryInterruption = `
story "Monthly Newsletter" {
	priority 1
	allow_interruption true
	start_trigger "newsletter-subscriber"

	storyline "Newsletter Storyline" {
		order 1

		enactment "Newsletter 1" {
			order 1

			scene "Newsletter Scene 1" {
				subject "[NEWSLETTER #1/3] Your Monthly Update"
				body "<html><body><h2>Newsletter #1</h2><p>Your first monthly newsletter.</p></body></html>"
				from_email "creator@example.com"
				from_name "Monthly Newsletter"
				reply_to "creator@example.com"
			}

			on click "https://example.com/newsletter-1-advance" {
				trigger_priority 1
				within 1m
			}
		}

		enactment "Newsletter 2" {
			order 2

			scene "Newsletter Scene 2" {
				subject "[NEWSLETTER #2/3] Your Monthly Update"
				body "<html><body><h2>Newsletter #2</h2><p>Your second monthly newsletter.</p></body></html>"
				from_email "creator@example.com"
				from_name "Monthly Newsletter"
				reply_to "creator@example.com"
			}

			on click "https://example.com/newsletter-2-advance" {
				trigger_priority 1
				within 1m
			}
		}

		enactment "Newsletter 3" {
			order 3

			scene "Newsletter Scene 3" {
				subject "[NEWSLETTER #3/3] Your Monthly Update"
				body "<html><body><h2>Newsletter #3</h2><p>Your third monthly newsletter.</p></body></html>"
				from_email "creator@example.com"
				from_name "Monthly Newsletter"
				reply_to "creator@example.com"
			}
		}
	}
}

story "Cart Abandonment Recovery" {
	priority 10
	allow_interruption false
	start_trigger "cart-abandoned"

	storyline "Cart Recovery Storyline" {
		order 1

		enactment "Cart Recovery Email" {
			level 1
			order 1

			scene "Cart Recovery Scene" {
				subject "🛒 [CART RECOVERY — INTERRUPTING NEWSLETTER] Don't forget your cart!"
				body "<html><body><h2>🛒 Cart Recovery</h2><p>You left items in your cart! Complete your purchase.</p><a href='https://example.com/complete-purchase'>Complete Purchase</a></body></html>"
				from_email "creator@example.com"
				from_name "Cart Recovery"
				reply_to "creator@example.com"
			}

			on click "https://example.com/complete-purchase" {
				trigger_priority 1
				do advance_to_next_storyline
			}
		}
	}
}
`

// FixtureOutboundWebhooks — from e2e-outbound-webhooks.sh
// Simple single-storyline story demonstrating webhook event delivery.
// Note: outbound_webhook configuration is outside the DSL scope (it's an API config),
// but the story structure that triggers the events is captured here.
const FixtureOutboundWebhooks = `
story "Webhook Demo Story" {
	priority 1
	allow_interruption false
	start_trigger "story-start"

	storyline "Webhook Demo Storyline" {
		order 1

		enactment "Webhook Demo Email" {
			level 1
			order 1

			scene "Webhook Demo Scene" {
				subject "🪝 Webhook Demo — Click to Fire Events"
				body "<html><body><h2>🪝 Webhook Demo</h2><p>Clicking the link fires TriggerTriggered and StoryCompleted events.</p><a href='https://example.com/webhook-click'>Fire Webhook</a></body></html>"
				from_email "creator@example.com"
				from_name "Webhook Demo"
				reply_to "creator@example.com"
			}

			on click "https://example.com/webhook-click" {
				trigger_priority 1
				do advance_to_next_storyline
			}
		}
	}
}
`

// FixturePersistentLinks — from e2e-persistent-links.sh
// Demonstrates persist_scope "enactment" on triggers for enactments A and B.
// When a trigger has persist_scope "enactment", clicking a link from an older scene
// (already advanced past) still fires the trigger. Enactments C and D use default
// (scene-scoped) triggers.
const FixturePersistentLinks = `
story "Online Course — Persistent Links Demo" {
	start_trigger "start_persistent_demo"

	storyline "Online Course Funnel" {
		order 1

		enactment "EA-Sc1" {
			order 1
			scene "EA Scene 1" {
				subject "[EA-Sc1] Online Course — More Info"
				body "<html><body><p>More Info scene 1</p><a href='https://example.com/course-ea-1'>Learn More</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-ea-1" {
				trigger_priority 1
				persist_scope "enactment"
				do jump_to_enactment "EC-Sc1"
				within 30s
			}
		}

		enactment "EA-Sc2" {
			order 2
			scene "EA Scene 2" {
				subject "[EA-Sc2] Online Course — More Info"
				body "<html><body><p>More Info scene 2</p><a href='https://example.com/course-ea-2'>Learn More</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-ea-2" {
				trigger_priority 1
				persist_scope "enactment"
				do jump_to_enactment "EC-Sc1"
				within 30s
			}
		}

		enactment "EA-Sc3" {
			order 3
			scene "EA Scene 3" {
				subject "[EA-Sc3] Online Course — More Info"
				body "<html><body><p>More Info scene 3</p><a href='https://example.com/course-ea-3'>Learn More</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-ea-3" {
				trigger_priority 1
				persist_scope "enactment"
				do jump_to_enactment "EC-Sc1"
				within 30s
			}
		}

		enactment "EB-Sc1" {
			order 4
			scene "EB Scene 1" {
				subject "[EB-Sc1] Online Course — Still Interested?"
				body "<html><body><p>Still Interested scene 1</p><a href='https://example.com/course-eb-1'>Learn More</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-eb-1" {
				trigger_priority 1
				persist_scope "enactment"
				do jump_to_enactment "EC-Sc1"
				within 30s
			}
		}

		enactment "EB-Sc2" {
			order 5
			scene "EB Scene 2" {
				subject "[EB-Sc2] Online Course — Still Interested?"
				body "<html><body><p>Still Interested scene 2</p><a href='https://example.com/course-eb-2'>Learn More</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-eb-2" {
				trigger_priority 1
				persist_scope "enactment"
				do jump_to_enactment "EC-Sc1"
				within 30s
			}
		}

		enactment "EB-Sc3" {
			order 6
			scene "EB Scene 3" {
				subject "[EB-Sc3] Online Course — Still Interested?"
				body "<html><body><p>Still Interested scene 3</p><a href='https://example.com/course-eb-3'>Learn More</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-eb-3" {
				trigger_priority 1
				persist_scope "enactment"
				do jump_to_enactment "EC-Sc1"
				within 30s
			}
		}

		enactment "EC-Sc1" {
			order 7
			scene "EC Scene 1" {
				subject "[EC-Sc1] Online Course — Buy Now"
				body "<html><body><p>Buy Now scene 1</p><a href='https://example.com/course-ec-1'>Buy Now</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-ec-1" {
				trigger_priority 1
				do advance_to_next_storyline
				within 30s
			}
		}

		enactment "EC-Sc2" {
			order 8
			scene "EC Scene 2" {
				subject "[EC-Sc2] Online Course — Buy Now"
				body "<html><body><p>Buy Now scene 2</p><a href='https://example.com/course-ec-2'>Buy Now</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-ec-2" {
				trigger_priority 1
				do advance_to_next_storyline
				within 30s
			}
		}

		enactment "EC-Sc3" {
			order 9
			scene "EC Scene 3" {
				subject "[EC-Sc3] Online Course — Buy Now"
				body "<html><body><p>Buy Now scene 3</p><a href='https://example.com/course-ec-3'>Buy Now</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-ec-3" {
				trigger_priority 1
				do advance_to_next_storyline
				within 30s
			}
		}

		enactment "ED-Sc1" {
			order 10
			scene "ED Scene 1" {
				subject "[ED-Sc1] Online Course — Last Chance"
				body "<html><body><p>Last Chance scene 1</p><a href='https://example.com/course-ed-1'>Buy Now - Last Chance</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-ed-1" {
				trigger_priority 1
				do advance_to_next_storyline
				within 30s
			}
		}

		enactment "ED-Sc2" {
			order 11
			scene "ED Scene 2" {
				subject "[ED-Sc2] Online Course — Last Chance"
				body "<html><body><p>Last Chance scene 2</p><a href='https://example.com/course-ed-2'>Buy Now - Last Chance</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-ed-2" {
				trigger_priority 1
				do advance_to_next_storyline
				within 30s
			}
		}

		enactment "ED-Sc3" {
			order 12
			scene "ED Scene 3" {
				subject "[ED-Sc3] Online Course — Last Chance"
				body "<html><body><p>Last Chance scene 3</p><a href='https://example.com/course-ed-3'>Buy Now - Last Chance</a></body></html>"
				from_email "demo@sentanyl-demo.local"
				from_name "Online Course Demo"
				reply_to "demo@sentanyl-demo.local"
			}
			on click "https://example.com/course-ed-3" {
				trigger_priority 1
				do advance_to_next_storyline
				within 30s
			}
		}
	}
}
`

// FixtureDeferredTransitions — from e2e-deferred-transitions.sh
// Demonstrates send_immediate false on all transitions. Three storylines, each with
// 4 enactment types (A=Soft Intrigue, B=Hard Intrigue, C=Soft Sell, D=Hard Sell) × 3 scenes.
// A/B click → jump to EC-Sc1 with send_immediate false (deferred).
// C/D click → advance_to_next_storyline with send_immediate false (deferred).
// B-Sc3 has skip_to_next_storyline_on_expiry true.
// Storyline on_complete gives sl{n}-purchased badge, on_fail gives sl{n}-not-purchased badge.
const FixtureDeferredTransitions = `
story "Buy All Three Manifesting Workshops" {
	start_trigger "start_story_a"

	storyline "Storyline 1 — Manifesting Workshop 1" {
		order 1
		on_complete { give_badge "sl1-purchased" }
		on_fail { give_badge "sl1-not-purchased" }

		enactment "SL1-EA-Sc1" {
			order 1
			scene "SL1 EA Scene 1" {
				subject "[S1-SL1-EA-Sc1] Manifesting Workshop 1 — Want to learn more? (1 of 3)"
				body "<html><body><p>Soft Intrigue 1</p><a href='https://example.com/s1-sl1-ea-1'>Get More Info</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-ea-1" {
				trigger_priority 1
				do jump_to_enactment "SL1-EC-Sc1"
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-EA-Sc2" {
			order 2
			scene "SL1 EA Scene 2" {
				subject "[S1-SL1-EA-Sc2] Manifesting Workshop 1 — Want to learn more? (2 of 3)"
				body "<html><body><p>Soft Intrigue 2</p><a href='https://example.com/s1-sl1-ea-2'>Get More Info</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-ea-2" {
				trigger_priority 1
				do jump_to_enactment "SL1-EC-Sc1"
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-EA-Sc3" {
			order 3
			scene "SL1 EA Scene 3" {
				subject "[S1-SL1-EA-Sc3] Manifesting Workshop 1 — Want to learn more? (3 of 3)"
				body "<html><body><p>Soft Intrigue 3</p><a href='https://example.com/s1-sl1-ea-3'>Get More Info</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-ea-3" {
				trigger_priority 1
				do jump_to_enactment "SL1-EC-Sc1"
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-EB-Sc1" {
			order 4
			scene "SL1 EB Scene 1" {
				subject "[S1-SL1-EB-Sc1] Manifesting Workshop 1 — Are you SURE? (1 of 3)"
				body "<html><body><p>Hard Intrigue 1</p><a href='https://example.com/s1-sl1-eb-1'>Get More Info</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-eb-1" {
				trigger_priority 1
				do jump_to_enactment "SL1-EC-Sc1"
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-EB-Sc2" {
			order 5
			scene "SL1 EB Scene 2" {
				subject "[S1-SL1-EB-Sc2] Manifesting Workshop 1 — Are you SURE? (2 of 3)"
				body "<html><body><p>Hard Intrigue 2</p><a href='https://example.com/s1-sl1-eb-2'>Get More Info</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-eb-2" {
				trigger_priority 1
				do jump_to_enactment "SL1-EC-Sc1"
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-EB-Sc3" {
			order 6
			skip_to_next_storyline_on_expiry true
			scene "SL1 EB Scene 3" {
				subject "[S1-SL1-EB-Sc3] Manifesting Workshop 1 — Are you SURE? (3 of 3)"
				body "<html><body><p>Hard Intrigue 3</p><a href='https://example.com/s1-sl1-eb-3'>Get More Info</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-eb-3" {
				trigger_priority 1
				do jump_to_enactment "SL1-EC-Sc1"
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-EC-Sc1" {
			order 7
			scene "SL1 EC Scene 1" {
				subject "[S1-SL1-EC-Sc1] Manifesting Workshop 1 — Ready to buy? 🛒 (1 of 3)"
				body "<html><body><p>Soft Sell 1</p><a href='https://example.com/s1-sl1-ec-1'>BUY NOW</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-ec-1" {
				trigger_priority 1
				do advance_to_next_storyline
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-EC-Sc2" {
			order 8
			scene "SL1 EC Scene 2" {
				subject "[S1-SL1-EC-Sc2] Manifesting Workshop 1 — Ready to buy? 🛒 (2 of 3)"
				body "<html><body><p>Soft Sell 2</p><a href='https://example.com/s1-sl1-ec-2'>BUY NOW</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-ec-2" {
				trigger_priority 1
				do advance_to_next_storyline
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-EC-Sc3" {
			order 9
			scene "SL1 EC Scene 3" {
				subject "[S1-SL1-EC-Sc3] Manifesting Workshop 1 — Ready to buy? 🛒 (3 of 3)"
				body "<html><body><p>Soft Sell 3</p><a href='https://example.com/s1-sl1-ec-3'>BUY NOW</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-ec-3" {
				trigger_priority 1
				do advance_to_next_storyline
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-ED-Sc1" {
			order 10
			scene "SL1 ED Scene 1" {
				subject "[S1-SL1-ED-Sc1] Manifesting Workshop 1 — LAST CHANCE ⏰ (1 of 3)"
				body "<html><body><p>Hard Sell 1</p><a href='https://example.com/s1-sl1-ed-1'>BUY NOW — FINAL CHANCE</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-ed-1" {
				trigger_priority 1
				do advance_to_next_storyline
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-ED-Sc2" {
			order 11
			scene "SL1 ED Scene 2" {
				subject "[S1-SL1-ED-Sc2] Manifesting Workshop 1 — LAST CHANCE ⏰ (2 of 3)"
				body "<html><body><p>Hard Sell 2</p><a href='https://example.com/s1-sl1-ed-2'>BUY NOW — FINAL CHANCE</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-ed-2" {
				trigger_priority 1
				do advance_to_next_storyline
				send_immediate false
				within 1m
			}
		}

		enactment "SL1-ED-Sc3" {
			order 12
			scene "SL1 ED Scene 3" {
				subject "[S1-SL1-ED-Sc3] Manifesting Workshop 1 — LAST CHANCE ⏰ (3 of 3)"
				body "<html><body><p>Hard Sell 3</p><a href='https://example.com/s1-sl1-ed-3'>BUY NOW — FINAL CHANCE</a></body></html>"
				from_email "creator@example.com"
				from_name "Manifesting Workshop 1"
				reply_to "creator@example.com"
			}
			on click "https://example.com/s1-sl1-ed-3" {
				trigger_priority 1
				do advance_to_next_storyline
				send_immediate false
				within 1m
			}
		}
	}

	storyline "Storyline 2 — Manifesting Workshop 2" {
		order 2
		on_complete { give_badge "sl2-purchased" }
		on_fail { give_badge "sl2-not-purchased" }

		enactment "SL2-EA-Sc1" { order 1 scene "SL2 EA Scene 1" { subject "[S1-SL2-EA-Sc1] Manifesting Workshop 2 — Want to learn more? (1 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ea-1" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EA-Sc2" { order 2 scene "SL2 EA Scene 2" { subject "[S1-SL2-EA-Sc2] Manifesting Workshop 2 — Want to learn more? (2 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ea-2" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EA-Sc3" { order 3 scene "SL2 EA Scene 3" { subject "[S1-SL2-EA-Sc3] Manifesting Workshop 2 — Want to learn more? (3 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ea-3" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EB-Sc1" { order 4 scene "SL2 EB Scene 1" { subject "[S1-SL2-EB-Sc1] Manifesting Workshop 2 — Are you SURE? (1 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-eb-1" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EB-Sc2" { order 5 scene "SL2 EB Scene 2" { subject "[S1-SL2-EB-Sc2] Manifesting Workshop 2 — Are you SURE? (2 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-eb-2" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EB-Sc3" { order 6 skip_to_next_storyline_on_expiry true scene "SL2 EB Scene 3" { subject "[S1-SL2-EB-Sc3] Manifesting Workshop 2 — Are you SURE? (3 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-eb-3" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EC-Sc1" { order 7 scene "SL2 EC Scene 1" { subject "[S1-SL2-EC-Sc1] Manifesting Workshop 2 — Ready to buy? 🛒 (1 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ec-1" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
		enactment "SL2-EC-Sc2" { order 8 scene "SL2 EC Scene 2" { subject "[S1-SL2-EC-Sc2] Manifesting Workshop 2 — Ready to buy? 🛒 (2 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ec-2" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
		enactment "SL2-EC-Sc3" { order 9 scene "SL2 EC Scene 3" { subject "[S1-SL2-EC-Sc3] Manifesting Workshop 2 — Ready to buy? 🛒 (3 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ec-3" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
		enactment "SL2-ED-Sc1" { order 10 scene "SL2 ED Scene 1" { subject "[S1-SL2-ED-Sc1] Manifesting Workshop 2 — LAST CHANCE ⏰ (1 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ed-1" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
		enactment "SL2-ED-Sc2" { order 11 scene "SL2 ED Scene 2" { subject "[S1-SL2-ED-Sc2] Manifesting Workshop 2 — LAST CHANCE ⏰ (2 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ed-2" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
		enactment "SL2-ED-Sc3" { order 12 scene "SL2 ED Scene 3" { subject "[S1-SL2-ED-Sc3] Manifesting Workshop 2 — LAST CHANCE ⏰ (3 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ed-3" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
	}

	storyline "Storyline 3 — Manifesting Workshop 3" {
		order 3
		on_complete { give_badge "sl3-purchased" }
		on_fail { give_badge "sl3-not-purchased" }

		enactment "SL3-EA-Sc1" { order 1 scene "SL3 EA Scene 1" { subject "[S1-SL3-EA-Sc1] Manifesting Workshop 3 — Want to learn more? (1 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ea-1" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EA-Sc2" { order 2 scene "SL3 EA Scene 2" { subject "[S1-SL3-EA-Sc2] Manifesting Workshop 3 — Want to learn more? (2 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ea-2" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EA-Sc3" { order 3 scene "SL3 EA Scene 3" { subject "[S1-SL3-EA-Sc3] Manifesting Workshop 3 — Want to learn more? (3 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ea-3" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EB-Sc1" { order 4 scene "SL3 EB Scene 1" { subject "[S1-SL3-EB-Sc1] Manifesting Workshop 3 — Are you SURE? (1 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-eb-1" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EB-Sc2" { order 5 scene "SL3 EB Scene 2" { subject "[S1-SL3-EB-Sc2] Manifesting Workshop 3 — Are you SURE? (2 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-eb-2" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EB-Sc3" { order 6 skip_to_next_storyline_on_expiry true scene "SL3 EB Scene 3" { subject "[S1-SL3-EB-Sc3] Manifesting Workshop 3 — Are you SURE? (3 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-eb-3" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EC-Sc1" { order 7 scene "SL3 EC Scene 1" { subject "[S1-SL3-EC-Sc1] Manifesting Workshop 3 — Ready to buy? 🛒 (1 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ec-1" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
		enactment "SL3-EC-Sc2" { order 8 scene "SL3 EC Scene 2" { subject "[S1-SL3-EC-Sc2] Manifesting Workshop 3 — Ready to buy? 🛒 (2 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ec-2" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
		enactment "SL3-EC-Sc3" { order 9 scene "SL3 EC Scene 3" { subject "[S1-SL3-EC-Sc3] Manifesting Workshop 3 — Ready to buy? 🛒 (3 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ec-3" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
		enactment "SL3-ED-Sc1" { order 10 scene "SL3 ED Scene 1" { subject "[S1-SL3-ED-Sc1] Manifesting Workshop 3 — LAST CHANCE ⏰ (1 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ed-1" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
		enactment "SL3-ED-Sc2" { order 11 scene "SL3 ED Scene 2" { subject "[S1-SL3-ED-Sc2] Manifesting Workshop 3 — LAST CHANCE ⏰ (2 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ed-2" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
		enactment "SL3-ED-Sc3" { order 12 scene "SL3 ED Scene 3" { subject "[S1-SL3-ED-Sc3] Manifesting Workshop 3 — LAST CHANCE ⏰ (3 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ed-3" { trigger_priority 1 do advance_to_next_storyline send_immediate false within 1m } }
	}
}
`

// FixtureMailhogFullSequence — from e2e-mailhog-full-sequence.sh
// Same structure as FixtureDeferredTransitions but all transitions are INSTANT
// (send_immediate is omitted, defaulting to true). This is the "Instant Transitions" variant.
const FixtureMailhogFullSequence = `
story "Buy All Three Manifesting Workshops" {
	start_trigger "start_story_a"

	storyline "Storyline 1 — Manifesting Workshop 1" {
		order 1
		on_complete { give_badge "sl1-purchased" }
		on_fail { give_badge "sl1-not-purchased" }

		enactment "SL1-EA-Sc1" { order 1 scene "SL1 EA Scene 1" { subject "[S1-SL1-EA-Sc1] Manifesting Workshop 1 — Want to learn more? (1 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ea-1" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" within 1m } }
		enactment "SL1-EA-Sc2" { order 2 scene "SL1 EA Scene 2" { subject "[S1-SL1-EA-Sc2] Manifesting Workshop 1 — Want to learn more? (2 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ea-2" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" within 1m } }
		enactment "SL1-EA-Sc3" { order 3 scene "SL1 EA Scene 3" { subject "[S1-SL1-EA-Sc3] Manifesting Workshop 1 — Want to learn more? (3 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ea-3" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" within 1m } }
		enactment "SL1-EB-Sc1" { order 4 scene "SL1 EB Scene 1" { subject "[S1-SL1-EB-Sc1] Manifesting Workshop 1 — Are you SURE? (1 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-eb-1" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" within 1m } }
		enactment "SL1-EB-Sc2" { order 5 scene "SL1 EB Scene 2" { subject "[S1-SL1-EB-Sc2] Manifesting Workshop 1 — Are you SURE? (2 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-eb-2" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" within 1m } }
		enactment "SL1-EB-Sc3" { order 6 skip_to_next_storyline_on_expiry true scene "SL1 EB Scene 3" { subject "[S1-SL1-EB-Sc3] Manifesting Workshop 1 — Are you SURE? (3 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-eb-3" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" within 1m } }
		enactment "SL1-EC-Sc1" { order 7 scene "SL1 EC Scene 1" { subject "[S1-SL1-EC-Sc1] Manifesting Workshop 1 — Ready to buy? 🛒 (1 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ec-1" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL1-EC-Sc2" { order 8 scene "SL1 EC Scene 2" { subject "[S1-SL1-EC-Sc2] Manifesting Workshop 1 — Ready to buy? 🛒 (2 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ec-2" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL1-EC-Sc3" { order 9 scene "SL1 EC Scene 3" { subject "[S1-SL1-EC-Sc3] Manifesting Workshop 1 — Ready to buy? 🛒 (3 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ec-3" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL1-ED-Sc1" { order 10 scene "SL1 ED Scene 1" { subject "[S1-SL1-ED-Sc1] Manifesting Workshop 1 — LAST CHANCE ⏰ (1 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ed-1" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL1-ED-Sc2" { order 11 scene "SL1 ED Scene 2" { subject "[S1-SL1-ED-Sc2] Manifesting Workshop 1 — LAST CHANCE ⏰ (2 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ed-2" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL1-ED-Sc3" { order 12 scene "SL1 ED Scene 3" { subject "[S1-SL1-ED-Sc3] Manifesting Workshop 1 — LAST CHANCE ⏰ (3 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ed-3" { trigger_priority 1 do advance_to_next_storyline within 1m } }
	}

	storyline "Storyline 2 — Manifesting Workshop 2" {
		order 2
		on_complete { give_badge "sl2-purchased" }
		on_fail { give_badge "sl2-not-purchased" }

		enactment "SL2-EA-Sc1" { order 1 scene "SL2 EA Scene 1" { subject "[S1-SL2-EA-Sc1] Manifesting Workshop 2 — Want to learn more? (1 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ea-1" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" within 1m } }
		enactment "SL2-EA-Sc2" { order 2 scene "SL2 EA Scene 2" { subject "[S1-SL2-EA-Sc2] Manifesting Workshop 2 — Want to learn more? (2 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ea-2" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" within 1m } }
		enactment "SL2-EA-Sc3" { order 3 scene "SL2 EA Scene 3" { subject "[S1-SL2-EA-Sc3] Manifesting Workshop 2 — Want to learn more? (3 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ea-3" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" within 1m } }
		enactment "SL2-EB-Sc1" { order 4 scene "SL2 EB Scene 1" { subject "[S1-SL2-EB-Sc1] Manifesting Workshop 2 — Are you SURE? (1 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-eb-1" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" within 1m } }
		enactment "SL2-EB-Sc2" { order 5 scene "SL2 EB Scene 2" { subject "[S1-SL2-EB-Sc2] Manifesting Workshop 2 — Are you SURE? (2 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-eb-2" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" within 1m } }
		enactment "SL2-EB-Sc3" { order 6 skip_to_next_storyline_on_expiry true scene "SL2 EB Scene 3" { subject "[S1-SL2-EB-Sc3] Manifesting Workshop 2 — Are you SURE? (3 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-eb-3" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" within 1m } }
		enactment "SL2-EC-Sc1" { order 7 scene "SL2 EC Scene 1" { subject "[S1-SL2-EC-Sc1] Manifesting Workshop 2 — Ready to buy? 🛒 (1 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ec-1" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL2-EC-Sc2" { order 8 scene "SL2 EC Scene 2" { subject "[S1-SL2-EC-Sc2] Manifesting Workshop 2 — Ready to buy? 🛒 (2 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ec-2" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL2-EC-Sc3" { order 9 scene "SL2 EC Scene 3" { subject "[S1-SL2-EC-Sc3] Manifesting Workshop 2 — Ready to buy? 🛒 (3 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ec-3" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL2-ED-Sc1" { order 10 scene "SL2 ED Scene 1" { subject "[S1-SL2-ED-Sc1] Manifesting Workshop 2 — LAST CHANCE ⏰ (1 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ed-1" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL2-ED-Sc2" { order 11 scene "SL2 ED Scene 2" { subject "[S1-SL2-ED-Sc2] Manifesting Workshop 2 — LAST CHANCE ⏰ (2 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ed-2" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL2-ED-Sc3" { order 12 scene "SL2 ED Scene 3" { subject "[S1-SL2-ED-Sc3] Manifesting Workshop 2 — LAST CHANCE ⏰ (3 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ed-3" { trigger_priority 1 do advance_to_next_storyline within 1m } }
	}

	storyline "Storyline 3 — Manifesting Workshop 3" {
		order 3
		on_complete { give_badge "sl3-purchased" }
		on_fail { give_badge "sl3-not-purchased" }

		enactment "SL3-EA-Sc1" { order 1 scene "SL3 EA Scene 1" { subject "[S1-SL3-EA-Sc1] Manifesting Workshop 3 — Want to learn more? (1 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ea-1" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" within 1m } }
		enactment "SL3-EA-Sc2" { order 2 scene "SL3 EA Scene 2" { subject "[S1-SL3-EA-Sc2] Manifesting Workshop 3 — Want to learn more? (2 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ea-2" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" within 1m } }
		enactment "SL3-EA-Sc3" { order 3 scene "SL3 EA Scene 3" { subject "[S1-SL3-EA-Sc3] Manifesting Workshop 3 — Want to learn more? (3 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ea-3" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" within 1m } }
		enactment "SL3-EB-Sc1" { order 4 scene "SL3 EB Scene 1" { subject "[S1-SL3-EB-Sc1] Manifesting Workshop 3 — Are you SURE? (1 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-eb-1" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" within 1m } }
		enactment "SL3-EB-Sc2" { order 5 scene "SL3 EB Scene 2" { subject "[S1-SL3-EB-Sc2] Manifesting Workshop 3 — Are you SURE? (2 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-eb-2" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" within 1m } }
		enactment "SL3-EB-Sc3" { order 6 skip_to_next_storyline_on_expiry true scene "SL3 EB Scene 3" { subject "[S1-SL3-EB-Sc3] Manifesting Workshop 3 — Are you SURE? (3 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-eb-3" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" within 1m } }
		enactment "SL3-EC-Sc1" { order 7 scene "SL3 EC Scene 1" { subject "[S1-SL3-EC-Sc1] Manifesting Workshop 3 — Ready to buy? 🛒 (1 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ec-1" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL3-EC-Sc2" { order 8 scene "SL3 EC Scene 2" { subject "[S1-SL3-EC-Sc2] Manifesting Workshop 3 — Ready to buy? 🛒 (2 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ec-2" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL3-EC-Sc3" { order 9 scene "SL3 EC Scene 3" { subject "[S1-SL3-EC-Sc3] Manifesting Workshop 3 — Ready to buy? 🛒 (3 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ec-3" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL3-ED-Sc1" { order 10 scene "SL3 ED Scene 1" { subject "[S1-SL3-ED-Sc1] Manifesting Workshop 3 — LAST CHANCE ⏰ (1 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ed-1" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL3-ED-Sc2" { order 11 scene "SL3 ED Scene 2" { subject "[S1-SL3-ED-Sc2] Manifesting Workshop 3 — LAST CHANCE ⏰ (2 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ed-2" { trigger_priority 1 do advance_to_next_storyline within 1m } }
		enactment "SL3-ED-Sc3" { order 12 scene "SL3 ED Scene 3" { subject "[S1-SL3-ED-Sc3] Manifesting Workshop 3 — LAST CHANCE ⏰ (3 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ed-3" { trigger_priority 1 do advance_to_next_storyline within 1m } }
	}
}
`

// FixtureHybridTransitions — from e2e-hybrid-transitions.sh
// Same as FixtureDeferredTransitions but C/D click → jump to E (Thank You) INSTANT.
// E enactment has no click trigger — only timer-based expiry advances to next storyline.
// Represented as: A/B deferred, C/D instant jump to E, E no triggers (timer only).
// For simplicity, only SL1 is shown fully; SL2/SL3 follow the identical pattern.
const FixtureHybridTransitions = `
story "Buy All Three Manifesting Workshops" {
	start_trigger "start_story_a"

	storyline "Storyline 1 — Manifesting Workshop 1" {
		order 1
		on_complete { give_badge "sl1-purchased" }
		on_fail { give_badge "sl1-not-purchased" }

		enactment "SL1-EA-Sc1" { order 1 scene "SL1 EA Scene 1" { subject "[S1-SL1-EA-Sc1] Manifesting Workshop 1 — Want to learn more? (1 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ea-1" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" send_immediate false within 1m } }
		enactment "SL1-EA-Sc2" { order 2 scene "SL1 EA Scene 2" { subject "[S1-SL1-EA-Sc2] Manifesting Workshop 1 — Want to learn more? (2 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ea-2" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" send_immediate false within 1m } }
		enactment "SL1-EA-Sc3" { order 3 scene "SL1 EA Scene 3" { subject "[S1-SL1-EA-Sc3] Manifesting Workshop 1 — Want to learn more? (3 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ea-3" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" send_immediate false within 1m } }
		enactment "SL1-EB-Sc1" { order 4 scene "SL1 EB Scene 1" { subject "[S1-SL1-EB-Sc1] Manifesting Workshop 1 — Are you SURE? (1 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-eb-1" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" send_immediate false within 1m } }
		enactment "SL1-EB-Sc2" { order 5 scene "SL1 EB Scene 2" { subject "[S1-SL1-EB-Sc2] Manifesting Workshop 1 — Are you SURE? (2 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-eb-2" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" send_immediate false within 1m } }
		enactment "SL1-EB-Sc3" { order 6 skip_to_next_storyline_on_expiry true scene "SL1 EB Scene 3" { subject "[S1-SL1-EB-Sc3] Manifesting Workshop 1 — Are you SURE? (3 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-eb-3" { trigger_priority 1 do jump_to_enactment "SL1-EC-Sc1" send_immediate false within 1m } }
		enactment "SL1-EC-Sc1" { order 7 scene "SL1 EC Scene 1" { subject "[S1-SL1-EC-Sc1] Manifesting Workshop 1 — Ready to buy? 🛒 (1 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ec-1" { trigger_priority 1 do jump_to_enactment "SL1-EE-Sc1" within 1m } }
		enactment "SL1-EC-Sc2" { order 8 scene "SL1 EC Scene 2" { subject "[S1-SL1-EC-Sc2] Manifesting Workshop 1 — Ready to buy? 🛒 (2 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ec-2" { trigger_priority 1 do jump_to_enactment "SL1-EE-Sc1" within 1m } }
		enactment "SL1-EC-Sc3" { order 9 scene "SL1 EC Scene 3" { subject "[S1-SL1-EC-Sc3] Manifesting Workshop 1 — Ready to buy? 🛒 (3 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ec-3" { trigger_priority 1 do jump_to_enactment "SL1-EE-Sc1" within 1m } }
		enactment "SL1-ED-Sc1" { order 10 scene "SL1 ED Scene 1" { subject "[S1-SL1-ED-Sc1] Manifesting Workshop 1 — LAST CHANCE ⏰ (1 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ed-1" { trigger_priority 1 do jump_to_enactment "SL1-EE-Sc1" within 1m } }
		enactment "SL1-ED-Sc2" { order 11 scene "SL1 ED Scene 2" { subject "[S1-SL1-ED-Sc2] Manifesting Workshop 1 — LAST CHANCE ⏰ (2 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ed-2" { trigger_priority 1 do jump_to_enactment "SL1-EE-Sc1" within 1m } }
		enactment "SL1-ED-Sc3" { order 12 scene "SL1 ED Scene 3" { subject "[S1-SL1-ED-Sc3] Manifesting Workshop 1 — LAST CHANCE ⏰ (3 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } on click "https://example.com/s1-sl1-ed-3" { trigger_priority 1 do jump_to_enactment "SL1-EE-Sc1" within 1m } }
		enactment "SL1-EE-Sc1" { order 13 scene "SL1 EE Scene 1" { subject "[S1-SL1-EE-Sc1] Manifesting Workshop 1 — Thank you for your purchase! 🎉" body "<html><body><h2>🎉 Thank you!</h2><p>Your purchase is confirmed.</p></body></html>" from_email "creator@example.com" from_name "Manifesting Workshop 1" reply_to "creator@example.com" } }
	}

	storyline "Storyline 2 — Manifesting Workshop 2" {
		order 2
		on_complete { give_badge "sl2-purchased" }
		on_fail { give_badge "sl2-not-purchased" }

		enactment "SL2-EA-Sc1" { order 1 scene "SL2 EA Scene 1" { subject "[S1-SL2-EA-Sc1] Manifesting Workshop 2 — Want to learn more? (1 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ea-1" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EA-Sc2" { order 2 scene "SL2 EA Scene 2" { subject "[S1-SL2-EA-Sc2] Manifesting Workshop 2 — Want to learn more? (2 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ea-2" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EA-Sc3" { order 3 scene "SL2 EA Scene 3" { subject "[S1-SL2-EA-Sc3] Manifesting Workshop 2 — Want to learn more? (3 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ea-3" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EB-Sc1" { order 4 scene "SL2 EB Scene 1" { subject "[S1-SL2-EB-Sc1] Manifesting Workshop 2 — Are you SURE? (1 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-eb-1" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EB-Sc2" { order 5 scene "SL2 EB Scene 2" { subject "[S1-SL2-EB-Sc2] Manifesting Workshop 2 — Are you SURE? (2 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-eb-2" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EB-Sc3" { order 6 skip_to_next_storyline_on_expiry true scene "SL2 EB Scene 3" { subject "[S1-SL2-EB-Sc3] Manifesting Workshop 2 — Are you SURE? (3 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-eb-3" { trigger_priority 1 do jump_to_enactment "SL2-EC-Sc1" send_immediate false within 1m } }
		enactment "SL2-EC-Sc1" { order 7 scene "SL2 EC Scene 1" { subject "[S1-SL2-EC-Sc1] Manifesting Workshop 2 — Ready to buy? 🛒 (1 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ec-1" { trigger_priority 1 do jump_to_enactment "SL2-EE-Sc1" within 1m } }
		enactment "SL2-EC-Sc2" { order 8 scene "SL2 EC Scene 2" { subject "[S1-SL2-EC-Sc2] Manifesting Workshop 2 — Ready to buy? 🛒 (2 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ec-2" { trigger_priority 1 do jump_to_enactment "SL2-EE-Sc1" within 1m } }
		enactment "SL2-EC-Sc3" { order 9 scene "SL2 EC Scene 3" { subject "[S1-SL2-EC-Sc3] Manifesting Workshop 2 — Ready to buy? 🛒 (3 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ec-3" { trigger_priority 1 do jump_to_enactment "SL2-EE-Sc1" within 1m } }
		enactment "SL2-ED-Sc1" { order 10 scene "SL2 ED Scene 1" { subject "[S1-SL2-ED-Sc1] Manifesting Workshop 2 — LAST CHANCE ⏰ (1 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ed-1" { trigger_priority 1 do jump_to_enactment "SL2-EE-Sc1" within 1m } }
		enactment "SL2-ED-Sc2" { order 11 scene "SL2 ED Scene 2" { subject "[S1-SL2-ED-Sc2] Manifesting Workshop 2 — LAST CHANCE ⏰ (2 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ed-2" { trigger_priority 1 do jump_to_enactment "SL2-EE-Sc1" within 1m } }
		enactment "SL2-ED-Sc3" { order 12 scene "SL2 ED Scene 3" { subject "[S1-SL2-ED-Sc3] Manifesting Workshop 2 — LAST CHANCE ⏰ (3 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } on click "https://example.com/s1-sl2-ed-3" { trigger_priority 1 do jump_to_enactment "SL2-EE-Sc1" within 1m } }
		enactment "SL2-EE-Sc1" { order 13 scene "SL2 EE Scene 1" { subject "[S1-SL2-EE-Sc1] Manifesting Workshop 2 — Thank you for your purchase! 🎉" body "<h2>🎉 Thank you!</h2>" from_email "creator@example.com" from_name "Manifesting Workshop 2" reply_to "creator@example.com" } }
	}

	storyline "Storyline 3 — Manifesting Workshop 3" {
		order 3
		on_complete { give_badge "sl3-purchased" }
		on_fail { give_badge "sl3-not-purchased" }

		enactment "SL3-EA-Sc1" { order 1 scene "SL3 EA Scene 1" { subject "[S1-SL3-EA-Sc1] Manifesting Workshop 3 — Want to learn more? (1 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ea-1" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EA-Sc2" { order 2 scene "SL3 EA Scene 2" { subject "[S1-SL3-EA-Sc2] Manifesting Workshop 3 — Want to learn more? (2 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ea-2" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EA-Sc3" { order 3 scene "SL3 EA Scene 3" { subject "[S1-SL3-EA-Sc3] Manifesting Workshop 3 — Want to learn more? (3 of 3)" body "<p>Soft Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ea-3" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EB-Sc1" { order 4 scene "SL3 EB Scene 1" { subject "[S1-SL3-EB-Sc1] Manifesting Workshop 3 — Are you SURE? (1 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-eb-1" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EB-Sc2" { order 5 scene "SL3 EB Scene 2" { subject "[S1-SL3-EB-Sc2] Manifesting Workshop 3 — Are you SURE? (2 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-eb-2" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EB-Sc3" { order 6 skip_to_next_storyline_on_expiry true scene "SL3 EB Scene 3" { subject "[S1-SL3-EB-Sc3] Manifesting Workshop 3 — Are you SURE? (3 of 3)" body "<p>Hard Intrigue</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-eb-3" { trigger_priority 1 do jump_to_enactment "SL3-EC-Sc1" send_immediate false within 1m } }
		enactment "SL3-EC-Sc1" { order 7 scene "SL3 EC Scene 1" { subject "[S1-SL3-EC-Sc1] Manifesting Workshop 3 — Ready to buy? 🛒 (1 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ec-1" { trigger_priority 1 do jump_to_enactment "SL3-EE-Sc1" within 1m } }
		enactment "SL3-EC-Sc2" { order 8 scene "SL3 EC Scene 2" { subject "[S1-SL3-EC-Sc2] Manifesting Workshop 3 — Ready to buy? 🛒 (2 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ec-2" { trigger_priority 1 do jump_to_enactment "SL3-EE-Sc1" within 1m } }
		enactment "SL3-EC-Sc3" { order 9 scene "SL3 EC Scene 3" { subject "[S1-SL3-EC-Sc3] Manifesting Workshop 3 — Ready to buy? 🛒 (3 of 3)" body "<p>Soft Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ec-3" { trigger_priority 1 do jump_to_enactment "SL3-EE-Sc1" within 1m } }
		enactment "SL3-ED-Sc1" { order 10 scene "SL3 ED Scene 1" { subject "[S1-SL3-ED-Sc1] Manifesting Workshop 3 — LAST CHANCE ⏰ (1 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ed-1" { trigger_priority 1 do jump_to_enactment "SL3-EE-Sc1" within 1m } }
		enactment "SL3-ED-Sc2" { order 11 scene "SL3 ED Scene 2" { subject "[S1-SL3-ED-Sc2] Manifesting Workshop 3 — LAST CHANCE ⏰ (2 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ed-2" { trigger_priority 1 do jump_to_enactment "SL3-EE-Sc1" within 1m } }
		enactment "SL3-ED-Sc3" { order 12 scene "SL3 ED Scene 3" { subject "[S1-SL3-ED-Sc3] Manifesting Workshop 3 — LAST CHANCE ⏰ (3 of 3)" body "<p>Hard Sell</p>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } on click "https://example.com/s1-sl3-ed-3" { trigger_priority 1 do jump_to_enactment "SL3-EE-Sc1" within 1m } }
		enactment "SL3-EE-Sc1" { order 13 scene "SL3 EE Scene 1" { subject "[S1-SL3-EE-Sc1] Manifesting Workshop 3 — Thank you for your purchase! 🎉" body "<h2>🎉 Thank you!</h2>" from_email "creator@example.com" from_name "Manifesting Workshop 3" reply_to "creator@example.com" } }
	}
}
`

// FixtureMultiStorylineEnactmentScene — from multiple-storyline-enactment-scene.sh
// Demonstrates the one-shot massive hierarchy: 3 storylines × 4 enactment types × 3 scenes = 36 scenes.
// Uses mark_complete on all triggers (the original script sets mark_complete: true).
const FixtureMultiStorylineEnactmentScene = `
story "Manifesting Workshops Complete Bundle" {
	storyline "Storyline: Manifesting 101" {
		order 1

		enactment "Manifesting 101 - Enactment A - Scene 1" { order 1 scene "Enactment A Scene 1" { subject "[Manifesting 101] Soft Intrigue (Email 1/3)" body "<h1>Manifesting 101 - Enactment A</h1><p>Email 1 of 3. Soft Intrigue</p><a href='https://example.com/more-info-a'>Click for More Info</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-a" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment A - Scene 2" { order 2 scene "Enactment A Scene 2" { subject "[Manifesting 101] Soft Intrigue (Email 2/3)" body "<h1>Manifesting 101 - Enactment A</h1><p>Email 2 of 3. Soft Intrigue</p><a href='https://example.com/more-info-a'>Click for More Info</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-a" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment A - Scene 3" { order 3 scene "Enactment A Scene 3" { subject "[Manifesting 101] Soft Intrigue (Email 3/3)" body "<h1>Manifesting 101 - Enactment A</h1><p>Email 3 of 3. Soft Intrigue</p><a href='https://example.com/more-info-a'>Click for More Info</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-a" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment B - Scene 1" { order 4 scene "Enactment B Scene 1" { subject "[Manifesting 101] Are you sure you dont want more info!? (Email 1/3)" body "<h1>Manifesting 101 - Enactment B</h1><p>Email 1 of 3. Hard Intrigue</p><a href='https://example.com/more-info-b'>Click for More Info</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-b" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment B - Scene 2" { order 5 scene "Enactment B Scene 2" { subject "[Manifesting 101] Are you sure you dont want more info!? (Email 2/3)" body "<h1>Manifesting 101 - Enactment B</h1><p>Email 2 of 3. Hard Intrigue</p><a href='https://example.com/more-info-b'>Click for More Info</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-b" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment B - Scene 3" { order 6 scene "Enactment B Scene 3" { subject "[Manifesting 101] Are you sure you dont want more info!? (Email 3/3)" body "<h1>Manifesting 101 - Enactment B</h1><p>Email 3 of 3. Hard Intrigue</p><a href='https://example.com/more-info-b'>Click for More Info</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-b" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment C - Scene 1" { order 7 scene "Enactment C Scene 1" { subject "[Manifesting 101] Here is the offer. (Email 1/3)" body "<h1>Manifesting 101 - Enactment C</h1><p>Email 1 of 3. Soft Sell</p><a href='https://example.com/buy-now-soft'>Buy Now</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-soft" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment C - Scene 2" { order 8 scene "Enactment C Scene 2" { subject "[Manifesting 101] Here is the offer. (Email 2/3)" body "<h1>Manifesting 101 - Enactment C</h1><p>Email 2 of 3. Soft Sell</p><a href='https://example.com/buy-now-soft'>Buy Now</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-soft" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment C - Scene 3" { order 9 scene "Enactment C Scene 3" { subject "[Manifesting 101] Here is the offer. (Email 3/3)" body "<h1>Manifesting 101 - Enactment C</h1><p>Email 3 of 3. Soft Sell</p><a href='https://example.com/buy-now-soft'>Buy Now</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-soft" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment D - Scene 1" { order 10 scene "Enactment D Scene 1" { subject "[Manifesting 101] You are missing out bro. Buy now dude. (Email 1/3)" body "<h1>Manifesting 101 - Enactment D</h1><p>Email 1 of 3. Hard Sell</p><a href='https://example.com/buy-now-hard'>Buy Now</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-hard" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment D - Scene 2" { order 11 scene "Enactment D Scene 2" { subject "[Manifesting 101] You are missing out bro. Buy now dude. (Email 2/3)" body "<h1>Manifesting 101 - Enactment D</h1><p>Email 2 of 3. Hard Sell</p><a href='https://example.com/buy-now-hard'>Buy Now</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-hard" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Manifesting 101 - Enactment D - Scene 3" { order 12 scene "Enactment D Scene 3" { subject "[Manifesting 101] You are missing out bro. Buy now dude. (Email 3/3)" body "<h1>Manifesting 101 - Enactment D</h1><p>Email 3 of 3. Hard Sell</p><a href='https://example.com/buy-now-hard'>Buy Now</a>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-hard" { trigger_priority 1 mark_complete true within 1m } }
	}

	storyline "Storyline: Advanced Attraction" {
		order 2

		enactment "Advanced Attraction - Enactment A - Scene 1" { order 1 scene "AA Enactment A Scene 1" { subject "[Advanced Attraction] Soft Intrigue (Email 1/3)" body "<h1>Advanced Attraction - Enactment A</h1><p>Email 1 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-a" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment A - Scene 2" { order 2 scene "AA Enactment A Scene 2" { subject "[Advanced Attraction] Soft Intrigue (Email 2/3)" body "<h1>Advanced Attraction - Enactment A</h1><p>Email 2 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-a" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment A - Scene 3" { order 3 scene "AA Enactment A Scene 3" { subject "[Advanced Attraction] Soft Intrigue (Email 3/3)" body "<h1>Advanced Attraction - Enactment A</h1><p>Email 3 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-a" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment B - Scene 1" { order 4 scene "AA Enactment B Scene 1" { subject "[Advanced Attraction] Hard Intrigue (Email 1/3)" body "<h1>Advanced Attraction - Enactment B</h1><p>Email 1 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-b" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment B - Scene 2" { order 5 scene "AA Enactment B Scene 2" { subject "[Advanced Attraction] Hard Intrigue (Email 2/3)" body "<h1>Advanced Attraction - Enactment B</h1><p>Email 2 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-b" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment B - Scene 3" { order 6 scene "AA Enactment B Scene 3" { subject "[Advanced Attraction] Hard Intrigue (Email 3/3)" body "<h1>Advanced Attraction - Enactment B</h1><p>Email 3 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-b" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment C - Scene 1" { order 7 scene "AA Enactment C Scene 1" { subject "[Advanced Attraction] Soft Sell (Email 1/3)" body "<h1>Advanced Attraction - Enactment C</h1><p>Email 1 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-soft" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment C - Scene 2" { order 8 scene "AA Enactment C Scene 2" { subject "[Advanced Attraction] Soft Sell (Email 2/3)" body "<h1>Advanced Attraction - Enactment C</h1><p>Email 2 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-soft" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment C - Scene 3" { order 9 scene "AA Enactment C Scene 3" { subject "[Advanced Attraction] Soft Sell (Email 3/3)" body "<h1>Advanced Attraction - Enactment C</h1><p>Email 3 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-soft" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment D - Scene 1" { order 10 scene "AA Enactment D Scene 1" { subject "[Advanced Attraction] Hard Sell (Email 1/3)" body "<h1>Advanced Attraction - Enactment D</h1><p>Email 1 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-hard" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment D - Scene 2" { order 11 scene "AA Enactment D Scene 2" { subject "[Advanced Attraction] Hard Sell (Email 2/3)" body "<h1>Advanced Attraction - Enactment D</h1><p>Email 2 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-hard" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Advanced Attraction - Enactment D - Scene 3" { order 12 scene "AA Enactment D Scene 3" { subject "[Advanced Attraction] Hard Sell (Email 3/3)" body "<h1>Advanced Attraction - Enactment D</h1><p>Email 3 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-hard" { trigger_priority 1 mark_complete true within 1m } }
	}

	storyline "Storyline: Quantum Wealth" {
		order 3

		enactment "Quantum Wealth - Enactment A - Scene 1" { order 1 scene "QW Enactment A Scene 1" { subject "[Quantum Wealth] Soft Intrigue (Email 1/3)" body "<h1>Quantum Wealth - Enactment A</h1><p>Email 1 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-a" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment A - Scene 2" { order 2 scene "QW Enactment A Scene 2" { subject "[Quantum Wealth] Soft Intrigue (Email 2/3)" body "<h1>Quantum Wealth - Enactment A</h1><p>Email 2 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-a" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment A - Scene 3" { order 3 scene "QW Enactment A Scene 3" { subject "[Quantum Wealth] Soft Intrigue (Email 3/3)" body "<h1>Quantum Wealth - Enactment A</h1><p>Email 3 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-a" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment B - Scene 1" { order 4 scene "QW Enactment B Scene 1" { subject "[Quantum Wealth] Hard Intrigue (Email 1/3)" body "<h1>Quantum Wealth - Enactment B</h1><p>Email 1 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-b" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment B - Scene 2" { order 5 scene "QW Enactment B Scene 2" { subject "[Quantum Wealth] Hard Intrigue (Email 2/3)" body "<h1>Quantum Wealth - Enactment B</h1><p>Email 2 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-b" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment B - Scene 3" { order 6 scene "QW Enactment B Scene 3" { subject "[Quantum Wealth] Hard Intrigue (Email 3/3)" body "<h1>Quantum Wealth - Enactment B</h1><p>Email 3 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/more-info-b" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment C - Scene 1" { order 7 scene "QW Enactment C Scene 1" { subject "[Quantum Wealth] Soft Sell (Email 1/3)" body "<h1>Quantum Wealth - Enactment C</h1><p>Email 1 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-soft" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment C - Scene 2" { order 8 scene "QW Enactment C Scene 2" { subject "[Quantum Wealth] Soft Sell (Email 2/3)" body "<h1>Quantum Wealth - Enactment C</h1><p>Email 2 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-soft" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment C - Scene 3" { order 9 scene "QW Enactment C Scene 3" { subject "[Quantum Wealth] Soft Sell (Email 3/3)" body "<h1>Quantum Wealth - Enactment C</h1><p>Email 3 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-soft" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment D - Scene 1" { order 10 scene "QW Enactment D Scene 1" { subject "[Quantum Wealth] Hard Sell (Email 1/3)" body "<h1>Quantum Wealth - Enactment D</h1><p>Email 1 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-hard" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment D - Scene 2" { order 11 scene "QW Enactment D Scene 2" { subject "[Quantum Wealth] Hard Sell (Email 2/3)" body "<h1>Quantum Wealth - Enactment D</h1><p>Email 2 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-hard" { trigger_priority 1 mark_complete true within 1m } }
		enactment "Quantum Wealth - Enactment D - Scene 3" { order 12 scene "QW Enactment D Scene 3" { subject "[Quantum Wealth] Hard Sell (Email 3/3)" body "<h1>Quantum Wealth - Enactment D</h1><p>Email 3 of 3</p>" from_email "coach@demo.com" from_name "Manifesting Coach" reply_to "coach@demo.com" } on click "https://example.com/buy-now-hard" { trigger_priority 1 mark_complete true within 1m } }
	}
}
`
