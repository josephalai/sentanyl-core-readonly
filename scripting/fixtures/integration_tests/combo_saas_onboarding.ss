# Combo: SaaS Onboarding
#
# SaaS onboarding flow combining: data blocks + for loops,
# pattern reuse, scene_defaults, deferred transitions, and
# default sender. Flow: Trial -> Activation -> Conversion.

default sender {
    from_email "onboard@saas.example.com"
    from_name "SaaS Platform"
    reply_to "help@saas.example.com"
}

links {
    setup_link = "https://saas.example.com/setup"
    integrate_link = "https://saas.example.com/integrate"
    invite_link = "https://saas.example.com/invite"
    upgrade_link = "https://saas.example.com/upgrade"
}

data activation_steps = [
    { name: "Setup", order: 1, link: setup_link, badge: "setup_done" },
    { name: "Integrate", order: 2, link: integrate_link, badge: "integrated" },
    { name: "Invite Team", order: 3, link: invite_link, badge: "team_invited" }
]

policy step_complete(link, badge) {
    on click link {
        trigger_priority 1
        mark_complete true
        within 2d
        do give_badge "${badge}"
        do send_immediate false
    }
}

scene_defaults {
    on not_open {
        within 1d
        do retry_scene up_to 1
            else do next_scene
    }
}

story "SaaS Onboarding" {
    use sender default
    priority 5

    on_begin {
        give_badge "trial_started"
    }
    on_complete {
        give_badge "onboarding_complete"
        remove_badge "trial_started"
    }

    # Activation storylines generated from data
    for step in activation_steps {
        storyline "${step.name} Track" {
            order step.order

            enactment "${step.name} Step" {
                scene "${step.name} Email" {
                    subject "Next step: ${step.name}"
                    body "<p>Complete this step to continue. <a href='${step.link}'>${step.name}</a></p>"
                }

                use policy step_complete(step.link, step.badge)

                on not_click step.link {
                    within 3d
                    do send_immediate false
                    do wait 1d
                    do advance_to_next_storyline
                }
            }
        }
    }

    # Conversion storyline after all activation steps
    storyline "Upgrade" {
        order 4

        enactment "Upgrade Prompt" {
            level 1
            order 1
            scene "Upgrade Email" {
                subject "Ready to upgrade?"
                body "<p>Your trial is ending. <a href='https://saas.example.com/upgrade'>Upgrade Now</a></p>"
            }
            on click "https://saas.example.com/upgrade" {
                trigger_priority 1
                do give_badge "converted"
                do mark_complete
            }
            on not_click "https://saas.example.com/upgrade" {
                within 5d
                do retry_scene up_to 2 times
                    else do mark_failed
            }
        }
    }
}
