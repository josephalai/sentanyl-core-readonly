package scripting

// ========== Atomic Feature Coverage Fixtures ==========
// Each fixture exercises specific atomic features to ensure 100%
// coverage of all DSL constructs.

// FixtureAtomicAllTriggerTypes exercises ALL 15 trigger types:
// click, not_click, open, not_open, sent, webhook, nothing, else,
// bounce, spam, unsubscribe, failure, email_validated, user_has_tag, badge.
// Also exercises trigger-level properties: trigger_priority, within,
// persist_scope (all 5 values), mark_complete, mark_failed, send_immediate,
// and required_badges on triggers.
const FixtureAtomicAllTriggerTypes = `
default sender {
	from_email "triggers@example.com"
	from_name "Trigger Test"
	reply_to "triggers@example.com"
}

story "All Trigger Types" {
	use sender default

	storyline "Triggers" {
		order 1

		enactment "Click and Open" {
			level 1
			order 1
			scene "Email" {
				subject "Click and Open Test"
				body "<a href='https://example.com/go'>Go</a>"
			}
			on click "https://example.com/go" {
				trigger_priority 5
				persist_scope "enactment"
				mark_complete true
				send_immediate true
				do give_badge "clicked"
			}
			on not_click "https://example.com/go" {
				trigger_priority 4
				within 1d
				persist_scope "storyline"
				do next_scene
			}
			on open {
				trigger_priority 3
				persist_scope "story"
				do give_badge "opened"
			}
			on not_open {
				trigger_priority 2
				within 2d
				persist_scope "forever"
				mark_failed false
				do retry_scene up_to 1
					else do mark_failed
			}
			on sent {
				trigger_priority 1
				do give_badge "delivery_confirmed"
			}
		}

		enactment "Event Triggers" {
			level 2
			order 2
			scene "Email" {
				subject "Event Trigger Test"
				body "<p>Event triggers</p>"
			}
			on webhook "status_update" {
				trigger_priority 5
				do next_scene
			}
			on nothing {
				within 3d
				do mark_complete
			}
			on bounce {
				do mark_failed
			}
			on spam {
				do mark_failed
			}
			on else {
				do next_scene
			}
		}

		enactment "Status Triggers" {
			level 3
			order 3
			scene "Email" {
				subject "Status Trigger Test"
				body "<p>Status triggers</p>"
			}
			on unsubscribe {
				do unsubscribe
				do end_story
			}
			on failure {
				do mark_failed
			}
			on email_validated {
				do give_badge "email_valid"
			}
			on user_has_tag "premium" {
				trigger_priority 2
				required_badges { must_have "active" must_not_have "suspended" }
				do give_badge "premium_verified"
			}
			on badge "loyalty_member" {
				trigger_priority 1
				do mark_complete
			}
		}
	}
}
`

// FixtureAtomicAllActionTypes exercises ALL 20 action types:
// next_scene, prev_scene, jump_to_enactment, jump_to_storyline,
// next_enactment, advance_to_next_storyline, end_story, mark_complete,
// mark_failed, unsubscribe, give_badge, remove_badge, retry_scene,
// retry_enactment, loop_to_enactment, loop_to_storyline,
// loop_to_start_enactment, loop_to_start_storyline, wait, send_immediate.
// Also exercises: skip_to_next_storyline_on_expiry and multi-line else blocks.
const FixtureAtomicAllActionTypes = `
default sender {
	from_email "actions@example.com"
	from_name "Action Test"
	reply_to "actions@example.com"
}

story "All Action Types" {
	use sender default

	storyline "Nav Actions" {
		order 1

		enactment "Scene Navigation" {
			level 1
			order 1
			skip_to_next_storyline_on_expiry true
			scene "Scene A" {
				subject "Navigation A"
				body "<p>Nav A</p>"
			}
			scene "Scene B" {
				subject "Navigation B"
				body "<p>Nav B</p>"
			}
			on click "https://example.com/next" {
				trigger_priority 6
				do next_scene
			}
			on click "https://example.com/prev" {
				trigger_priority 5
				do prev_scene
			}
			on click "https://example.com/jump-e" {
				trigger_priority 4
				do jump_to_enactment "Badge Actions"
			}
			on click "https://example.com/jump-s" {
				trigger_priority 3
				do jump_to_storyline "Loop Actions"
			}
			on click "https://example.com/next-e" {
				trigger_priority 2
				do next_enactment "Badge Actions"
			}
			on click "https://example.com/advance" {
				trigger_priority 1
				do advance_to_next_storyline
			}
		}

		enactment "Badge Actions" {
			level 2
			order 2
			scene "Email" {
				subject "Badge Actions"
				body "<p>Badges</p>"
			}
			on click "https://example.com/give" {
				trigger_priority 5
				do give_badge "earned"
				do remove_badge "pending"
			}
			on click "https://example.com/end" {
				trigger_priority 4
				do end_story
			}
			on click "https://example.com/complete" {
				trigger_priority 3
				do mark_complete
			}
			on click "https://example.com/fail" {
				trigger_priority 2
				do mark_failed
			}
			on click "https://example.com/unsub" {
				trigger_priority 1
				do unsubscribe
			}
		}

		enactment "Timing Actions" {
			level 3
			order 3
			scene "Email" {
				subject "Timing Actions"
				body "<p>Timing</p>"
			}
			on click "https://example.com/wait" {
				trigger_priority 2
				do wait 2h
			}
			on click "https://example.com/delay" {
				trigger_priority 1
				do send_immediate false
			}
		}

		enactment "Retry Actions" {
			level 4
			order 4
			scene "Email" {
				subject "Retry Actions"
				body "<p>Retry</p>"
			}
			on not_open {
				within 1d
				trigger_priority 2
				do retry_scene up_to 2 times
					else do next_scene
			}
			on not_click "https://example.com/act" {
				within 2d
				trigger_priority 1
				do retry_enactment up_to 3 times
					else {
						do give_badge "exhausted"
						do advance_to_next_storyline
					}
			}
		}
	}

	storyline "Loop Actions" {
		order 2

		enactment "Named Loops" {
			level 1
			order 1
			scene "Email" {
				subject "Named Loops"
				body "<p>Loop</p>"
			}
			on not_open {
				within 1d
				trigger_priority 2
				do loop_to_enactment "Named Loops" up_to 2
					else do mark_failed
			}
			on not_click "https://example.com/go" {
				within 2d
				trigger_priority 1
				do loop_to_storyline "Nav Actions" up_to 1
					else do end_story
			}
		}

		enactment "Start Loops" {
			level 2
			order 2
			scene "Email" {
				subject "Start Loops"
				body "<p>Start loop</p>"
			}
			on not_open {
				within 1d
				trigger_priority 2
				do loop_to_start_enactment up_to 3
					else do mark_failed
			}
			on not_click "https://example.com/go" {
				within 2d
				trigger_priority 1
				do loop_to_start_storyline up_to 2
					else {
						do give_badge "loop_exhausted"
						do end_story
					}
			}
		}
	}
}
`

// FixtureAtomicBadgeIntegration exercises ALL badge-related constructs:
// story-level: required_badges, on_begin/on_complete/on_fail with give_badge/remove_badge,
//   start_trigger, complete_trigger, next_story
// storyline-level: required_badges, on_begin/on_complete/on_fail with give_badge,
//   next_storyline in on_complete and on_fail
// trigger-level: required_badges (must_have + must_not_have), do give_badge, do remove_badge
const FixtureAtomicBadgeIntegration = `
default sender {
	from_email "badges@example.com"
	from_name "Badge Test"
	reply_to "badges@example.com"
}

story "Badge Integration" {
	use sender default
	priority 1
	allow_interruption true
	start_trigger "enrollment"
	complete_trigger "graduation"

	required_badges {
		must_not_have "already_graduated"
	}

	on_begin {
		give_badge "enrolled"
	}
	on_complete {
		give_badge "graduated"
		remove_badge "enrolled"
	}
	on_fail {
		give_badge "dropped_out"
		remove_badge "enrolled"
	}

	storyline "Coursework" {
		order 1
		required_badges { must_have "enrolled" }

		on_begin {
			give_badge "coursework_started"
		}
		on_complete {
			give_badge "coursework_done"
			next_storyline "Final Exam"
		}
		on_fail {
			give_badge "coursework_failed"
			next_storyline "Remedial"
		}

		enactment "Lesson 1" {
			level 1
			scene "Email" {
				subject "Lesson 1"
				body "<p>First lesson</p>"
			}
			on click "https://example.com/pass" {
				trigger_priority 2
				required_badges { must_have "enrolled" must_not_have "banned" }
				do give_badge "lesson1_passed"
				do mark_complete
			}
			on not_click "https://example.com/pass" {
				trigger_priority 1
				within 3d
				do remove_badge "coursework_started"
				do mark_failed
			}
		}
	}

	storyline "Final Exam" {
		order 2
		required_badges { must_have "coursework_done" }

		enactment "Exam" {
			level 1
			scene "Email" {
				subject "Final Exam"
				body "<p>Take your exam</p>"
			}
			on click "https://example.com/submit" {
				do give_badge "exam_passed"
				do mark_complete
			}
		}
	}

	storyline "Remedial" {
		order 3

		enactment "Extra Help" {
			level 1
			scene "Email" {
				subject "Extra Help"
				body "<p>Additional support</p>"
			}
			on click "https://example.com/retry" {
				do remove_badge "coursework_failed"
				do give_badge "remedial_done"
				do mark_complete
			}
		}
	}
}
`

// FixtureAtomicConditionsAndRouting exercises ALL condition types and routing:
// when has_badge, when not_has_badge, when has_tag, when not_has_tag,
// when and { ... }, when or { ... }, when not <condition>,
// conditional_route with priority and required_badges.
const FixtureAtomicConditionsAndRouting = `
default sender {
	from_email "conditions@example.com"
	from_name "Condition Test"
	reply_to "conditions@example.com"
}

story "Conditions And Routing" {
	use sender default

	storyline "Evaluation" {
		order 1

		on_complete {
			conditional_route {
				priority 10
				required_badges { must_have "premium" }
				next_storyline "Premium Path"
			}
			conditional_route {
				priority 1
				next_storyline "Standard Path"
			}
		}

		enactment "Badge Conditions" {
			level 1
			order 1
			scene "Email" {
				subject "Badge Conditions"
				body "<p>Badge check</p>"
			}
			on click "https://example.com/a" {
				trigger_priority 2
				when has_badge "vip"
				do give_badge "premium"
				do mark_complete
			}
			on click "https://example.com/b" {
				trigger_priority 1
				when not_has_badge "vip"
				do mark_complete
			}
		}

		enactment "Tag Conditions" {
			level 2
			order 2
			scene "Email" {
				subject "Tag Conditions"
				body "<p>Tag check</p>"
			}
			on click "https://example.com/c" {
				trigger_priority 2
				when has_tag "subscriber"
				do give_badge "premium"
				do mark_complete
			}
			on click "https://example.com/d" {
				trigger_priority 1
				when not_has_tag "subscriber"
				do mark_complete
			}
		}

		enactment "Compound Conditions" {
			level 3
			order 3
			scene "Email" {
				subject "Compound Conditions"
				body "<p>Compound check</p>"
			}
			on click "https://example.com/and" {
				trigger_priority 3
				when and {
					has_badge "vip"
					has_tag "subscriber"
				}
				do give_badge "premium"
				do mark_complete
			}
			on click "https://example.com/or" {
				trigger_priority 2
				when or {
					has_badge "vip"
					has_tag "subscriber"
				}
				do give_badge "premium"
				do mark_complete
			}
			on click "https://example.com/not" {
				trigger_priority 1
				when not has_badge "banned"
				do mark_complete
			}
		}
	}

	storyline "Premium Path" {
		order 2
		required_badges { must_have "premium" }

		enactment "Premium" {
			level 1
			scene "Email" {
				subject "Premium Content"
				body "<p>Exclusive content</p>"
			}
		}
	}

	storyline "Standard Path" {
		order 3

		enactment "Standard" {
			level 1
			scene "Email" {
				subject "Standard Content"
				body "<p>Regular content</p>"
			}
		}
	}
}
`

// FixtureAtomicSceneFeatures exercises scene-specific features:
// template, vars (map syntax with colon separators), tags (array syntax).
const FixtureAtomicSceneFeatures = `
default sender {
	from_email "scenes@example.com"
	from_name "Scene Test"
	reply_to "scenes@example.com"
}

story "Scene Features" {
	use sender default

	storyline "Main" {
		order 1

		enactment "Rich Scene" {
			level 1
			scene "Templated Email" {
				subject "Feature Rich Email"
				body "<p>All scene features</p>"
				template "marketing-blast"
				vars {
					hero_image: "https://example.com/hero.png"
					cta_text: "Shop Now"
					footer_note: "Unsubscribe anytime"
				}
				tags ["promo", "q4-launch", "featured"]
			}
		}
	}
}
`

// FixtureAtomicBadgeCampaign exercises generative features combined
// with badge mechanics: data blocks with structured objects, for loops
// generating storylines, and badge give/remove in generated content.
const FixtureAtomicBadgeCampaign = `
default sender {
	from_email "v3@example.com"
	from_name "V3 Badge Test"
	reply_to "v3@example.com"
}

data modules = [
	{ name: "Fundamentals", badge: "fundamentals_done", subject: "Learn the basics" },
	{ name: "Intermediate", badge: "intermediate_done", subject: "Level up" },
	{ name: "Advanced", badge: "advanced_done", subject: "Master it" }
]

story "V3 Badge Campaign" {
	use sender default

	on_begin {
		give_badge "v3_enrolled"
	}
	on_complete {
		give_badge "v3_graduated"
		remove_badge "v3_enrolled"
	}

	for mod in modules {
		storyline "${mod.name}" {
			enactment "${mod.name} Lesson" {
				scene "${mod.name} Email" {
					subject "${mod.subject}"
					body "<p>${mod.name} content</p>"
				}
				on click "https://example.com/${mod.name}" {
					do give_badge "${mod.badge}"
					do mark_complete
				}
			}
		}
	}
}
`
