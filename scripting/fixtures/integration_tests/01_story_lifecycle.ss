# 01 — Story Lifecycle: on_begin, on_complete, on_fail
#
# Tests story-level lifecycle hooks with badge transactions
# and next_story chaining. Verifies that badges are granted
# on entry, completion, and failure, and that the engine
# chains to the follow-up story on completion.

story "Onboarding Flow" {
    priority 5

    on_begin {
        give_badge "onboarding_started"
    }
    on_complete {
        give_badge "onboarding_done"
        remove_badge "onboarding_started"
        next_story "Retention Flow"
    }
    on_fail {
        give_badge "onboarding_failed"
        remove_badge "onboarding_started"
    }

    storyline "Welcome" {
        order 1

        enactment "Welcome Email" {
            level 1
            order 1

            scene "Welcome" {
                subject "Welcome aboard!"
                body "<h1>Welcome!</h1><p>Get started with <a href='https://example.com/start'>our guide</a>.</p>"
                from_email "hello@example.com"
                from_name "Onboarding"
                reply_to "support@example.com"
            }

            on click "https://example.com/start" {
                trigger_priority 1
                do mark_complete
            }

            on not_click "https://example.com/start" {
                within 3d
                do mark_failed
            }
        }
    }
}

story "Retention Flow" {
    priority 3

    storyline "Check-In" {
        order 1

        enactment "Check-In Email" {
            level 1
            order 1

            scene "Check-In" {
                subject "How are things going?"
                body "<p>We want to make sure you are getting value.</p>"
                from_email "hello@example.com"
                from_name "Success Team"
                reply_to "support@example.com"
            }
        }
    }
}
