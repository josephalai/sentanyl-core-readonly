# 42 — Webinar Registration Funnel
#
# Webinar funnel: registration → confirmation → replay (with video tracking).
# Tests video block with on watch trigger, form submission, badge-gated routes,
# and email story integration for webinar reminders.

funnel "Webinar Funnel" {
    domain "webinar.example.com"

    route "Registration" {
        order 1

        stage "Register" {
            path "/register"

            page "Registration Page" {
                template "minimal_v1"

                block "webinar_info" {
                    length medium
                    prompt "Webinar title, date, and what attendees will learn"
                }

                form "RegisterForm" {
                    field email required
                    field first_name
                }
            }

            on submit "RegisterForm" {
                do give_badge "webinar_registered"
                do start_story "Webinar Reminders"
                do jump_to_stage "Confirmation"
            }
        }

        stage "Confirmation" {
            path "/confirmed"

            page "Confirmation Page" {
                block "confirmed" {
                    length short
                    prompt "Registration confirmed with calendar add instructions"
                }
            }
        }
    }

    route "Replay" {
        order 2
        must_have_badge "webinar_registered"

        stage "Watch Replay" {
            path "/replay"

            page "Replay Page" {
                template "minimal_v1"

                block "replay_video" {
                    type video
                    source_url "https://cdn.example.com/webinar-replay.mp4"
                    autoplay false
                }

                block "replay_cta" {
                    length short
                    prompt "Call to action after watching the replay"
                }

                form "ReplayOffer" {
                    type checkout
                    product_id "webinar-course"
                    field email required
                    field card required
                }
            }

            on watch "replay_video" > 75 {
                do give_badge "replay_engaged"
            }

            on purchase "ReplayOffer" {
                do give_badge "webinar_buyer"
                do start_story "Course Welcome"
                do jump_to_stage "Thank You"
            }
        }

        stage "Thank You" {
            path "/thank-you"

            page "Purchase Thanks" {
                block "thanks" {
                    length short
                    prompt "Purchase confirmation and course access"
                }
            }
        }
    }
}

story "Webinar Reminders" {
    priority 1

    storyline "Countdown" {
        order 1

        enactment "Registered" {
            scene "Confirm Email" {
                subject "You are registered for the webinar!"
                body "<p>Mark your calendar. We will send you a reminder before we go live.</p>"
                from_email "webinar@example.com"
                from_name "Webinar Team"
                reply_to "webinar@example.com"
            }
            on sent {
                within 1d
                do next_scene
            }
        }

        enactment "Reminder" {
            scene "Day Before" {
                subject "Webinar starts tomorrow!"
                body "<p>Do not forget — the webinar is tomorrow. Here is your link to join.</p>"
                from_email "webinar@example.com"
                from_name "Webinar Team"
                reply_to "webinar@example.com"
            }
        }
    }
}

story "Course Welcome" {
    priority 2

    required_badges {
        must_have ["webinar_buyer"]
    }

    storyline "Onboarding" {
        order 1

        enactment "Welcome" {
            scene "Course Access" {
                subject "Your course is ready!"
                body "<p>Welcome! Here is how to access your course materials.</p>"
                from_email "support@example.com"
                from_name "Course Support"
                reply_to "support@example.com"
            }
        }
    }
}
