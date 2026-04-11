# 45 — Upsell Pipeline Funnel
#
# Multi-stage upsell: main offer → order bump → one-time offer → downsell.
# Tests sequential checkout stages, multiple purchase triggers,
# badge accumulation through the funnel, and conditional routing.

funnel "Upsell Pipeline" {
    domain "offers.example.com"

    route "Main" {
        order 1

        stage "Main Offer" {
            path "/offer"

            page "Main Offer Page" {
                template "minimal_v1"

                block "offer_headline" {
                    length short
                    prompt "Main product headline and value proposition"
                }

                block "offer_details" {
                    length long
                    prompt "Full product description with benefits and testimonials"
                }

                form "MainPurchase" {
                    type checkout
                    product_id "main-product"
                    field email required
                    field first_name
                    field card required
                }
            }

            on purchase "MainPurchase" {
                do give_badge "main_buyer"
                do start_story "Main Product Onboarding"
                do jump_to_stage "Order Bump"
            }
        }

        stage "Order Bump" {
            path "/bump"

            page "Order Bump Page" {
                block "bump_offer" {
                    length short
                    prompt "Complementary add-on offer at a discount"
                }

                form "BumpPurchase" {
                    type checkout
                    product_id "order-bump"
                    field email required
                    field card required
                }
            }

            on purchase "BumpPurchase" {
                do give_badge "bump_buyer"
                do jump_to_stage "OTO"
            }

            on submit "BumpPurchase" {
                do jump_to_stage "OTO"
            }
        }

        stage "OTO" {
            path "/special"

            page "One Time Offer" {
                block "oto_pitch" {
                    length medium
                    prompt "Exclusive one-time offer only available now"
                }

                block "oto_video" {
                    type video
                    source_url "https://cdn.example.com/oto-pitch.mp4"
                    autoplay true
                }

                form "OTOPurchase" {
                    type checkout
                    product_id "premium-upgrade"
                    field email required
                    field card required
                }
            }

            on watch "oto_video" > 50 {
                do give_badge "oto_engaged"
            }

            on purchase "OTOPurchase" {
                do give_badge "premium_buyer"
                do start_story "Premium Onboarding"
                do jump_to_stage "Thank You"
            }
        }

        stage "Thank You" {
            path "/thank-you"

            page "Final Thank You" {
                block "thanks" {
                    length short
                    prompt "Thank you message with all purchase confirmations"
                }

                block "next_steps" {
                    length medium
                    prompt "Getting started guide with login credentials"
                }
            }
        }
    }
}

story "Main Product Onboarding" {
    priority 1

    storyline "Getting Started" {
        order 1

        enactment "Welcome" {
            scene "Purchase Confirmed" {
                subject "Your purchase is confirmed!"
                body "<p>Welcome! Here is how to access your product.</p>"
                from_email "support@offers.example.com"
                from_name "Support"
                reply_to "support@offers.example.com"
            }
            on sent {
                within 1d
                do next_scene
            }
        }

        enactment "Day 2" {
            scene "Getting Started" {
                subject "Getting started with your product"
                body "<p>Here are 3 things to do in your first 24 hours.</p>"
                from_email "support@offers.example.com"
                from_name "Support"
                reply_to "support@offers.example.com"
            }
        }
    }
}

story "Premium Onboarding" {
    priority 2

    required_badges {
        must_have ["premium_buyer"]
    }

    storyline "Premium Setup" {
        order 1

        enactment "Premium Welcome" {
            scene "Premium Access" {
                subject "Premium access unlocked!"
                body "<p>You now have premium access. Here is everything included.</p>"
                from_email "vip@offers.example.com"
                from_name "VIP Support"
                reply_to "vip@offers.example.com"
            }
        }
    }
}
