# 39 — Full Pipeline: Funnel + Video + Checkout + Email Story
#
# End-to-end test combining all new features:
# - Video tracking with on watch triggers
# - Lead capture form with badge assignment
# - Checkout form with purchase trigger
# - Email story launched from funnel actions
# - Badge-gated route for buyers
# - Multiple stages with jump_to_stage navigation

funnel "Launch Funnel" {
    domain "launch.example.com"

    route "Public" {
        order 1

        stage "Landing" {
            path "/start"

            page "Landing Page" {
                template "minimal_v1"

                block "hero_video" {
                    type video
                    source_url "https://cdn.example.com/launch.mp4"
                    autoplay false
                }

                block "headline" {
                    length short
                    prompt "Compelling headline for the offer"
                }

                form "OptIn" {
                    field email required
                    field first_name
                }
            }

            on watch "hero_video" > 75 {
                do give_badge "watched_video"
            }

            on submit "OptIn" {
                do give_badge "lead"
                do start_story "Nurture Sequence"
                do jump_to_stage "Special Offer"
            }
        }

        stage "Special Offer" {
            path "/special-offer"

            page "Offer Page" {
                block "offer_copy" {
                    length medium
                    prompt "Limited-time offer details"
                }

                form "BuyNow" {
                    type checkout
                    product_id "launch-special"
                    field email required
                    field card required
                }
            }

            on purchase "BuyNow" {
                do give_badge "buyer"
                do jump_to_stage "Thank You"
                do start_story "Buyer Onboarding"
            }
        }

        stage "Thank You" {
            path "/thanks"

            page "Thanks Page" {
                block "confirmation" {
                    length short
                    prompt "Thank you message with next steps"
                }
            }
        }
    }

    route "Members" {
        order 2
        must_have_badge "buyer"

        stage "Dashboard" {
            path "/members"

            page "Members Area" {
                block "welcome_back" {
                    length medium
                    prompt "Members-only content and resources"
                }
            }
        }
    }
}

story "Nurture Sequence" {
    priority 1

    storyline "Warm Up" {
        order 1

        enactment "Intro" {
            scene "Welcome" {
                subject "Welcome to our community!"
                body "<p>Thanks for joining. Here is what to expect.</p>"
                from_email "hello@launch.example.com"
                from_name "Launch Team"
                reply_to "hello@launch.example.com"
            }

            on sent {
                within 1d
                do next_scene
            }
        }

        enactment "Follow Up" {
            scene "Day 2" {
                subject "Did you see our special offer?"
                body "<p>Check out the limited-time deal we have for you.</p>"
                from_email "hello@launch.example.com"
                from_name "Launch Team"
                reply_to "hello@launch.example.com"
            }
        }
    }
}

story "Buyer Onboarding" {
    priority 2

    required_badges {
        must_have ["buyer"]
    }

    storyline "Getting Started" {
        order 1

        enactment "Welcome" {
            scene "Buyer Welcome" {
                subject "Your purchase is confirmed!"
                body "<p>Welcome aboard! Here is how to access your content.</p>"
                from_email "support@launch.example.com"
                from_name "Support"
                reply_to "support@launch.example.com"
            }
        }
    }
}
