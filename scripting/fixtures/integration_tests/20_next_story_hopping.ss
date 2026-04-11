# 20 — Next Story Hopping: next_story in on_complete / on_fail
#
# Tests next_story chaining across stories. Story A chains to
# Story B on both completion and failure, ensuring the user
# always progresses to the follow-up regardless of outcome.

story "Welcome Series" {
    priority 1

    on_complete {
        give_badge "welcome_completed"
        next_story "Engagement Series"
    }
    on_fail {
        give_badge "welcome_failed"
        next_story "Engagement Series"
    }

    storyline "Welcome" {
        order 1

        enactment "Welcome Email" {
            level 1
            order 1

            scene "Welcome" {
                subject "Welcome to our platform"
                body "<p>Get started: <a href='https://example.com/start'>Begin</a></p>"
                from_email "hello@example.com"
                from_name "Welcome Team"
                reply_to "hello@example.com"
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

story "Engagement Series" {
    priority 2

    on_complete {
        give_badge "engagement_done"
        next_story "Loyalty Series"
    }

    storyline "Engage" {
        order 1

        enactment "Engagement Email" {
            level 1
            order 1

            scene "Engage" {
                subject "Stay engaged with us"
                body "<p>Check out new features: <a href='https://example.com/features'>Explore</a></p>"
                from_email "engage@example.com"
                from_name "Engagement Team"
                reply_to "engage@example.com"
            }

            on click "https://example.com/features" {
                do mark_complete
            }
        }
    }
}

story "Loyalty Series" {
    priority 3

    storyline "Loyalty" {
        order 1

        enactment "Loyalty Reward" {
            level 1
            order 1

            scene "Loyalty Email" {
                subject "Thank you for being loyal"
                body "<p>Here is a reward: <a href='https://example.com/reward'>Claim</a></p>"
                from_email "loyalty@example.com"
                from_name "Loyalty Team"
                reply_to "loyalty@example.com"
            }

            on click "https://example.com/reward" {
                do give_badge "loyal_member"
                do mark_complete
            }
        }
    }
}
