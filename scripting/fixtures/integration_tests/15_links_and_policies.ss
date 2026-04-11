# 15 — Links Block + Policy Definitions
#
# Tests the links registry and policy definitions. Links provide
# named URL references, and policies define reusable trigger logic
# that can be applied to enactments via "use policy".

links {
    signup_link = "https://example.com/signup"
    pricing_link = "https://example.com/pricing"
}

policy click_and_badge(link, badge_name) {
    on click link {
        trigger_priority 1
        mark_complete true
        within 2d
        do give_badge "${badge_name}"
    }
}

policy fallback_retry() {
    on not_open {
        within 1d
        do retry_scene up_to 2
            else do mark_failed
    }
}

story "Links and Policies Demo" {
    priority 1

    storyline "Signup Flow" {
        order 1

        enactment "Signup Prompt" {
            level 1
            order 1

            scene "Signup Email" {
                subject "Join us today"
                body "<p>Sign up now: <a href='https://example.com/signup'>Join</a></p>"
                from_email "growth@example.com"
                from_name "Growth Team"
                reply_to "growth@example.com"
            }

            use policy click_and_badge(signup_link, "signed_up")
            use policy fallback_retry()
        }

        enactment "Pricing Intro" {
            level 2
            order 2

            scene "Pricing Email" {
                subject "Check our pricing"
                body "<p>See what fits: <a href='https://example.com/pricing'>View Plans</a></p>"
                from_email "growth@example.com"
                from_name "Growth Team"
                reply_to "growth@example.com"
            }

            use policy click_and_badge(pricing_link, "saw_pricing")
            use policy fallback_retry()
        }
    }
}
