# 13 — Enactment Defaults: enactment_defaults with policy application
#
# Tests the enactment_defaults block that applies a policy
# to every enactment. The click_completes policy is inherited
# automatically, so enactments only need to define scenes.

links {
    main_cta = "https://example.com/main-cta"
}

policy click_completes(link) {
    on click link {
        trigger_priority 1
        mark_complete true
        within 3d
    }
}

enactment_defaults {
    use policy click_completes(main_cta)
}

story "Enactment Defaults Demo" {
    priority 1

    storyline "Content" {
        order 1

        enactment "Phase 1" {
            level 1
            order 1
            scene "Intro" {
                subject "Getting started"
                body "<p>Click to begin: <a href='https://example.com/main-cta'>Start</a></p>"
                from_email "content@example.com"
                from_name "Content Team"
                reply_to "content@example.com"
            }
            # click_completes policy is inherited from enactment_defaults
        }

        enactment "Phase 2" {
            level 2
            order 2
            scene "Deep Dive" {
                subject "Going deeper"
                body "<p>Continue learning: <a href='https://example.com/main-cta'>Continue</a></p>"
                from_email "content@example.com"
                from_name "Content Team"
                reply_to "content@example.com"
            }
            # click_completes policy is inherited from enactment_defaults
        }
    }
}
