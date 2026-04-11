// Full marketing campaign demonstrating all SentanylScript features:
// - Multiple storylines with ordering
// - Conditional routing by badges
// - Click/not-click branching
// - Open/not-open branching  
// - Bounded retry with fallback
// - Loop to prior enactment
// - Badge transactions on lifecycle events
// - Multi-scene enactments
// - Completion and failure paths
// - Start/complete triggers
// - Unsubscribe handling
// - Persist scopes and trigger priorities

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

    // --- Storyline 1: Awareness ---
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

            // If user opens, mark them as engaged
            on open {
                do give_badge "engaged_teaser"
            }

            // If user clicks the teaser link within 1 day, fast-track them
            on click "teaser_link" {
                within 1d
                do give_badge "high_intent"
                do jump_to_enactment "Reveal"
            }

            // If user doesn't open within 2 days, retry up to 2 times
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

    // --- Storyline 2: Hard Sell (for high-intent users) ---
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

    // --- Storyline 3: Soft Sell (for everyone else) ---
    storyline "Soft Sell" {
        order 3

        enactment "Education" {
            level 1
            order 1

            // Multi-scene enactment: two educational emails in sequence
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

            // Handle unsubscribe
            on unsubscribe {
                do unsubscribe
                do end_story
            }
        }
    }
}
