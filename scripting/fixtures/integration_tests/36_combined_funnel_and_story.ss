# 36 — Combined Funnel and Story Script
#
# Demonstrates the unified DSL with both a funnel and an
# email story in a single script. The funnel triggers the
# story via start_story action on form submission.

funnel "Launch Funnel" {
    domain "launch.example.com"

    route "Main" {
        order 1

        stage "OptIn" {
            path "/join"

            page "Join Page" {
                template "minimal_v1"

                block "headline" {
                    length short
                    prompt "Join our community headline"
                }

                form "JoinForm" {
                    field email required
                    field first_name
                }
            }

            on submit "JoinForm" {
                do give_badge "member"
                do start_story "Welcome Sequence"
                do jump_to_stage "Welcome"
            }
        }

        stage "Welcome" {
            path "/welcome"

            page "Welcome Page" {
                template "minimal_v1"

                block "welcome_message" {
                    length medium
                    prompt "Welcome message with next steps"
                }
            }
        }
    }
}

story "Welcome Sequence" {
    priority 1

    storyline "Onboarding" {
        order 1

        enactment "Welcome Email" {
            level 1
            scene "Welcome" {
                subject "Welcome to the community!"
                body "<p>Thanks for joining! Here's what to expect...</p>"
                from_email "hello@launch.example.com"
                from_name "Launch Team"
            }
            on sent {
                within 1d
                do next_scene
            }
        }

        enactment "Follow Up" {
            level 2
            scene "Day 2" {
                subject "Your first step"
                body "<p>Ready to get started? Here's your first step...</p>"
                from_email "hello@launch.example.com"
                from_name "Launch Team"
            }
            on sent {
                within 2d
                do next_scene
            }
        }
    }
}
