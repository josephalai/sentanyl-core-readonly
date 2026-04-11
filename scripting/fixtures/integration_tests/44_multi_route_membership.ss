# 44 — Multi-Route Membership Funnel
#
# Membership site with badge-gated routes: public → free member → paid member.
# Tests must_have_badge and must_not_have_badge on routes,
# multiple forms, and progressive badge accumulation.

funnel "Membership Site" {
    domain "members.example.com"

    route "Public" {
        order 1
        must_not_have_badge "free_member"

        stage "Home" {
            path "/"

            page "Home Page" {
                template "minimal_v1"

                block "hero" {
                    length short
                    prompt "Welcome headline and value proposition"
                }

                block "features" {
                    length medium
                    prompt "Key features and benefits of membership"
                }

                form "FreeSignup" {
                    field email required
                    field first_name
                }
            }

            on submit "FreeSignup" {
                do give_badge "free_member"
                do start_story "Member Welcome"
                do jump_to_stage "Free Dashboard"
            }
        }

        stage "Free Dashboard" {
            path "/free"

            page "Free Area" {
                block "free_content" {
                    length medium
                    prompt "Free tier content and upgrade teaser"
                }
            }
        }
    }

    route "Free Members" {
        order 2
        must_have_badge "free_member"
        must_not_have_badge "paid_member"

        stage "Free Content" {
            path "/members/free"

            page "Free Member Area" {
                template "minimal_v1"

                block "free_resources" {
                    length medium
                    prompt "Free tier resources and content"
                }

                block "upgrade_cta" {
                    length short
                    prompt "Why upgrade to paid membership"
                }

                form "UpgradeForm" {
                    type checkout
                    product_id "paid-membership"
                    field email required
                    field card required
                }
            }

            on purchase "UpgradeForm" {
                do give_badge "paid_member"
                do start_story "Paid Member Welcome"
                do jump_to_stage "Paid Content"
            }
        }

        stage "Paid Content" {
            path "/members/paid"

            page "Paid Member Area" {
                block "premium_content" {
                    length long
                    prompt "Premium member-only content and resources"
                }
            }
        }
    }

    route "Paid Members" {
        order 3
        must_have_badge "paid_member"

        stage "Premium Dashboard" {
            path "/members/premium"

            page "Premium Area" {
                block "premium_dashboard" {
                    length long
                    prompt "Full premium member dashboard with all features"
                }

                block "training_video" {
                    type video
                    source_url "https://cdn.example.com/premium-training.mp4"
                    autoplay false
                }
            }

            on watch "training_video" > 50 {
                do give_badge "training_started"
            }

            on watch "training_video" >= 90 {
                do give_badge "training_complete"
            }
        }
    }
}

story "Member Welcome" {
    priority 1

    storyline "Onboarding" {
        order 1

        enactment "Welcome" {
            scene "Free Member Welcome" {
                subject "Welcome to the community!"
                body "<p>Thanks for joining. Here is what you get as a free member.</p>"
                from_email "hello@members.example.com"
                from_name "Membership Team"
                reply_to "hello@members.example.com"
            }
            on sent {
                within 3d
                do next_scene
            }
        }

        enactment "Upgrade Pitch" {
            scene "Go Premium" {
                subject "Ready to unlock everything?"
                body "<p>Upgrade to paid membership and get access to premium content.</p>"
                from_email "hello@members.example.com"
                from_name "Membership Team"
                reply_to "hello@members.example.com"
            }
        }
    }
}

story "Paid Member Welcome" {
    priority 2

    required_badges {
        must_have ["paid_member"]
    }

    storyline "Premium Onboarding" {
        order 1

        enactment "Premium Welcome" {
            scene "Premium Access" {
                subject "You are now a premium member!"
                body "<p>Welcome to premium! Here is everything you now have access to.</p>"
                from_email "support@members.example.com"
                from_name "Support"
                reply_to "support@members.example.com"
            }
        }
    }
}
