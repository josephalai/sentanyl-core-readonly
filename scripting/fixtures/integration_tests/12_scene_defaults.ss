# 12 — Scene Defaults: scene_defaults block injecting triggers
#
# Tests the scene_defaults block that automatically injects
# triggers into every enactment. Here, a not_open retry trigger
# is inherited by all enactments without explicit declaration.

scene_defaults {
    on not_open {
        within 1d
        do retry_scene up_to 2
            else do mark_failed
    }
}

story "Defaults Campaign" {
    priority 1

    storyline "Outreach" {
        order 1

        enactment "Email 1" {
            level 1
            order 1

            scene "First Email" {
                subject "Exciting news for you"
                body "<p>Check it out: <a href='https://example.com/news'>Read More</a></p>"
                from_email "news@example.com"
                from_name "News Team"
                reply_to "news@example.com"
            }

            on click "https://example.com/news" {
                trigger_priority 1
                do mark_complete
            }
            # not_open retry is inherited from scene_defaults
        }

        enactment "Email 2" {
            level 2
            order 2

            scene "Second Email" {
                subject "More updates for you"
                body "<p>Do not miss this: <a href='https://example.com/updates'>See Updates</a></p>"
                from_email "news@example.com"
                from_name "News Team"
                reply_to "news@example.com"
            }

            on click "https://example.com/updates" {
                trigger_priority 1
                do mark_complete
            }
            # not_open retry is inherited from scene_defaults
        }
    }
}
