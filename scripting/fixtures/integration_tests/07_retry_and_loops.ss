# 07 — Retry and Loop Actions with else Fallbacks
#
# Tests retry_scene, retry_enactment, and loop_to_enactment
# with bounded retries and else fallback actions.

default sender {
    from_email "retry@example.com"
    from_name "Retry Test"
    reply_to "retry@example.com"
}

story "Retry Campaign" {
    use sender default
    priority 1

    storyline "Engagement" {
        order 1

        enactment "First Touch" {
            level 1
            order 1
            scene "Email" {
                subject "Hey, check this out"
                body "<p>Open me! <a href='https://example.com/read'>Read More</a></p>"
            }

            # Retry the scene up to 3 times if user does not open
            on not_open {
                within 1d
                do retry_scene up_to 3 times
                    else do jump_to_enactment "Last Chance"
            }

            on open {
                do give_badge "engaged"
                do mark_complete
            }
        }

        enactment "Follow-Up" {
            level 2
            order 2
            scene "Email" {
                subject "Following up"
                body "<p>We noticed you opened. <a href='https://example.com/act'>Take Action</a></p>"
            }

            # Retry the entire enactment up to 2 times on no-click
            on not_click "https://example.com/act" {
                within 2d
                do retry_enactment up_to 2 times
                    else do mark_failed
            }

            on click "https://example.com/act" {
                do mark_complete
            }
        }

        enactment "Last Chance" {
            level 3
            order 3
            scene "Email" {
                subject "Final reminder"
                body "<p>Last chance: <a href='https://example.com/final'>Act Now</a></p>"
            }

            # Loop back to First Touch up to 1 time
            on not_click "https://example.com/final" {
                within 1d
                do loop_to_enactment "First Touch" up_to 1
                    else do mark_failed
            }

            on click "https://example.com/final" {
                do mark_complete
            }
        }
    }
}
