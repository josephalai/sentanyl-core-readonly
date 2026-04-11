# 03 — All 15 Trigger Types
#
# Tests every trigger type the DSL supports:
# click, not_click, open, not_open, sent, webhook, nothing,
# else, bounce, spam, unsubscribe, failure, email_validated,
# user_has_tag, badge.

default sender {
    from_email "triggers@example.com"
    from_name "Trigger Test"
    reply_to "triggers@example.com"
}

story "Trigger Showcase" {
    use sender default
    priority 1

    storyline "Click Triggers" {
        order 1

        enactment "Click Tests" {
            level 1
            order 1
            scene "Email" {
                subject "Click and Open Triggers"
                body "<a href='https://example.com/cta'>Click here</a>"
            }

            on click "https://example.com/cta" {
                trigger_priority 7
                do give_badge "clicked_cta"
                do mark_complete
            }
            on not_click "https://example.com/cta" {
                trigger_priority 6
                within 2d
                do next_scene
            }
            on open {
                trigger_priority 5
                do give_badge "opened"
            }
            on not_open {
                trigger_priority 4
                within 1d
                do retry_scene up_to 1
                    else do mark_failed
            }
            on sent {
                trigger_priority 3
                do give_badge "delivered"
            }
        }
    }

    storyline "Event Triggers" {
        order 2

        enactment "Event Tests" {
            level 1
            order 1
            scene "Email" {
                subject "Event Triggers"
                body "<p>Testing event-based triggers</p>"
            }

            on webhook "payment_received" {
                trigger_priority 5
                do give_badge "paid"
                do mark_complete
            }
            on nothing {
                within 5d
                do mark_failed
            }
            on else {
                do next_scene
            }
            on bounce {
                do mark_failed
            }
            on spam {
                do mark_failed
            }
        }
    }

    storyline "Status Triggers" {
        order 3

        enactment "Status Tests" {
            level 1
            order 1
            scene "Email" {
                subject "Status Triggers"
                body "<p>Testing status triggers</p>"
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
            on user_has_tag "high_value" {
                trigger_priority 2
                do give_badge "priority_user"
            }
            on badge "loyalty_100" {
                trigger_priority 1
                do mark_complete
            }
        }
    }
}
