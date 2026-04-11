# 32 — Basic Funnel Test
#
# Tests the simplest funnel with a single route, two stages,
# a form with submit trigger, and badge action.
# Includes a companion story so the integration test passes.

funnel "Basic Funnel Test" {
    domain "test.example.com"

    route "Main" {
        order 1

        stage "Landing" {
            path "/landing"

            page "Landing Page" {
                template "minimal_v1"

                block "hero" {
                    length short
                    prompt "Test headline"
                }

                form "SignUp" {
                    field email required
                    field first_name
                }
            }

            on submit "SignUp" {
                do give_badge "lead"
                do jump_to_stage "ThankYou"
            }
        }

        stage "ThankYou" {
            path "/thank-you"

            page "Thank You Page" {
                template "minimal_v1"

                block "confirmation" {
                    length short
                    prompt "Thank you message"
                }
            }
        }
    }
}

story "Basic Funnel Companion" {
    priority 1

    storyline "Notify" {
        order 1

        enactment "Welcome" {
            scene "Welcome" {
                subject "Welcome!"
                body "<p>Thanks for signing up.</p>"
                from_email "hello@test.example.com"
                from_name "Test"
            }
            on sent {
                within 1d
                do next_scene
            }
        }
    }
}
