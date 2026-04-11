package scripting

// ========== Coverage Gap Fixtures ==========
// These fixtures fill in coverage gaps from the Feature Coverage Matrix,
// exercising DSL constructs that are not yet covered by existing fixtures.

// FixtureNextStoryHopping — Dedicated e2e for next_story chaining.
// Two stories: "Welcome Series" chains into "Post-Purchase Follow-Up"
// via next_story in both on_complete and on_fail. Demonstrates that
// regardless of outcome, the engine advances to the next story.
const FixtureNextStoryHopping = `
# next_story hopping: Welcome Series -> Post-Purchase Follow-Up
# on_complete and on_fail both chain to the same follow-up story.

story "Welcome Series" {
	priority 1

	on_complete {
		give_badge "welcome_done"
		next_story "Post-Purchase Follow-Up"
	}
	on_fail {
		give_badge "welcome_failed"
		next_story "Post-Purchase Follow-Up"
	}

	storyline "Welcome SL1" {
		order 1

		enactment "Welcome Email" {
			level 1
			order 1

			scene "Welcome Scene" {
				subject "Welcome to Our Platform!"
				body "<html><body><h2>Welcome!</h2><p>We are glad you signed up. Click below to get started.</p><a href='https://example.com/get-started'>Get Started</a></body></html>"
				from_email "onboarding@example.com"
				from_name "Onboarding Team"
				reply_to "support@example.com"
			}

			on click "https://example.com/get-started" {
				trigger_priority 1
				do mark_complete
			}

			on not_click "https://example.com/get-started" {
				within 3d
				do mark_failed
			}
		}
	}
}

story "Post-Purchase Follow-Up" {
	priority 2

	storyline "Follow-Up SL1" {
		order 1

		enactment "Follow-Up Email" {
			level 1
			order 1

			scene "Follow-Up Scene" {
				subject "How Are You Enjoying Your Purchase?"
				body "<html><body><h2>We Hope You Love It!</h2><p>Let us know how your experience has been so far.</p><a href='https://example.com/feedback'>Leave Feedback</a></body></html>"
				from_email "support@example.com"
				from_name "Customer Success"
				reply_to "support@example.com"
			}

			on click "https://example.com/feedback" {
				trigger_priority 1
				do mark_complete
			}
		}
	}
}
`

// FixtureStorylineOnFailRoutes — Storyline on_fail with conditional routes.
// One story with 3 storylines. The first storyline's on_fail block awards
// a badge and uses two conditional_route entries to route to different
// recovery storylines based on whether the user has the "premium" badge.
const FixtureStorylineOnFailRoutes = `
# Storyline on_fail conditional routing.
# SL1 fails -> conditional_route checks "premium" badge:
#   premium user    -> "Premium Recovery" (priority 2)
#   non-premium     -> "Standard Recovery" (priority 1)

story "Recovery Campaign" {
	priority 1

	storyline "Main Flow" {
		order 1

		on_fail {
			give_badge "sl1_failed"
			conditional_route {
				required_badges { must_have "premium" }
				next_storyline "Premium Recovery"
				priority 2
			}
			conditional_route {
				required_badges { must_not_have "premium" }
				next_storyline "Standard Recovery"
				priority 1
			}
		}

		enactment "Initial Offer" {
			level 1
			order 1

			scene "Offer Scene" {
				subject "Exclusive Offer Just for You"
				body "<html><body><h2>Special Deal</h2><p>Act now before this offer expires!</p><a href='https://example.com/claim-offer'>Claim Offer</a></body></html>"
				from_email "deals@example.com"
				from_name "Deals Team"
				reply_to "deals@example.com"
			}

			on click "https://example.com/claim-offer" {
				trigger_priority 1
				do mark_complete
			}

			on not_click "https://example.com/claim-offer" {
				within 2d
				do mark_failed
			}
		}
	}

	storyline "Premium Recovery" {
		order 2
		required_badges { must_have "premium" }

		enactment "Premium Recovery Email" {
			level 1
			order 1

			scene "Premium Recovery Scene" {
				subject "We Have a Special Recovery Offer for Premium Members"
				body "<html><body><h2>Premium Recovery</h2><p>As a premium member, here is an enhanced offer just for you.</p><a href='https://example.com/premium-recovery'>Recover Now</a></body></html>"
				from_email "deals@example.com"
				from_name "Premium Support"
				reply_to "premium@example.com"
			}

			on click "https://example.com/premium-recovery" {
				trigger_priority 1
				do mark_complete
			}
		}
	}

	storyline "Standard Recovery" {
		order 3

		enactment "Standard Recovery Email" {
			level 1
			order 1

			scene "Standard Recovery Scene" {
				subject "We Noticed You Missed Our Offer"
				body "<html><body><h2>Standard Recovery</h2><p>No worries — here is another chance to take advantage of our deal.</p><a href='https://example.com/standard-recovery'>Try Again</a></body></html>"
				from_email "deals@example.com"
				from_name "Deals Team"
				reply_to "deals@example.com"
			}

			on click "https://example.com/standard-recovery" {
				trigger_priority 1
				do mark_complete
			}
		}
	}
}
`

// FixtureSceneTemplateName — template keyword in scene blocks.
// One story, 1 storyline, 2 enactments. Each enactment's scene uses
// the `template` keyword to reference a named template, and includes
// a vars block with Handlebars-style data for rendering.
const FixtureSceneTemplateName = `
# Scene template_name demonstration.
# Two enactments each reference a different template with vars.

story "Template Campaign" {
	priority 1

	storyline "Template SL1" {
		order 1

		enactment "Welcome Enactment" {
			level 1
			order 1

			scene "Welcome Templated Scene" {
				subject "Welcome to {{company_name}}"
				body "<p>Rendered via template</p>"
				from_email "marketing@example.com"
				from_name "Marketing Team"
				reply_to "marketing@example.com"
				template "welcome_template"
				vars {
					company_name: "Acme Corp"
					hero_image: "https://example.com/welcome-hero.png"
					cta_url: "https://example.com/get-started"
				}
			}

			on click "https://example.com/get-started" {
				trigger_priority 1
				do mark_complete
			}
		}

		enactment "Follow-Up Enactment" {
			level 2
			order 2

			scene "Follow-Up Templated Scene" {
				subject "A Quick Follow-Up from {{sender_name}}"
				body "<p>Rendered via template</p>"
				from_email "marketing@example.com"
				from_name "Marketing Team"
				reply_to "marketing@example.com"
				template "followup_template"
				vars {
					sender_name: "Jane"
					offer_details: "20% off your next order"
					cta_url: "https://example.com/shop-now"
				}
			}

			on click "https://example.com/shop-now" {
				trigger_priority 1
				do mark_complete
			}
		}
	}
}
`

// FixtureSceneDefaultsTriggers — scene_defaults injecting triggers.
// A top-level scene_defaults block defines a not_open trigger with
// retry_scene that is automatically inherited by every enactment in
// the script. One story, 1 storyline, 2 enactments.
const FixtureSceneDefaultsTriggers = `
# scene_defaults trigger injection.
# The not_open retry trigger below is inherited by every enactment.

scene_defaults {
	on not_open {
		within 1d
		do retry_scene up_to 2
			else do mark_failed
	}
}

story "Defaults Demo Campaign" {
	priority 1

	storyline "Defaults SL1" {
		order 1

		enactment "First Touch" {
			level 1
			order 1

			scene "First Touch Scene" {
				subject "Introducing Our Latest Product"
				body "<html><body><h2>New Arrival</h2><p>Check out what just launched.</p><a href='https://example.com/new-product'>See Product</a></body></html>"
				from_email "launches@example.com"
				from_name "Product Team"
				reply_to "launches@example.com"
			}

			on click "https://example.com/new-product" {
				trigger_priority 1
				do mark_complete
			}
		}

		enactment "Second Touch" {
			level 2
			order 2

			scene "Second Touch Scene" {
				subject "Still Interested in Our New Product?"
				body "<html><body><h2>Reminder</h2><p>Do not miss out — take another look.</p><a href='https://example.com/new-product'>View Again</a></body></html>"
				from_email "launches@example.com"
				from_name "Product Team"
				reply_to "launches@example.com"
			}

			on click "https://example.com/new-product" {
				trigger_priority 1
				do mark_complete
			}
		}
	}
}
`

// FixtureHandlebarsVars — Dedicated Handlebars {{ var }} demonstration.
// One story, 1 storyline, 1 enactment, 1 scene with a vars block
// containing multiple key-value pairs. The subject and body reference
// the vars via Handlebars {{...}} syntax for template rendering.
const FixtureHandlebarsVars = `
# Handlebars vars demonstration.
# Scene vars feed into {{...}} placeholders in subject and body.

story "Handlebars Demo" {
	priority 1

	storyline "Handlebars SL1" {
		order 1

		enactment "Personalized Offer" {
			level 1
			order 1

			scene "Personalized Offer Scene" {
				subject "{{first_name}}, Your Exclusive Deal on {{product_name}}"
				body "<h1>Hello {{first_name}}!</h1><p>Your exclusive {{discount_code}} code for {{product_name}} from {{company_name}}.</p>"
				from_email "offers@example.com"
				from_name "Offers Team"
				reply_to "offers@example.com"
				vars {
					first_name: "John"
					product_name: "Premium Widget"
					discount_code: "SAVE20"
					company_name: "Acme Corp"
				}
			}

			on click "https://example.com/redeem" {
				trigger_priority 1
				do mark_complete
			}
		}
	}
}
`
