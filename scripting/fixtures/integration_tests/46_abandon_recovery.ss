# 46 — Abandon Recovery Funnel
#
# Tests the on abandon trigger with timed recovery.
# Landing page with opt-in → abandonment triggers email recovery story.
# Also tests video engagement badge for watch threshold.

funnel "Abandon Recovery" {
    domain "recovery.example.com"

    route "Main" {
        order 1

        stage "Sales Page" {
            path "/offer"

            page "Offer Page" {
                template "minimal_v1"

                block "pitch_video" {
                    type video
                    source_url "https://cdn.example.com/sales-pitch.mp4"
                    autoplay false
                }

                block "sales_copy" {
                    length long
                    prompt "Persuasive sales copy with urgency"
                }

                form "BuyForm" {
                    type checkout
                    product_id "digital-product"
                    field email required
                    field first_name
                    field card required
                }
            }

            on watch "pitch_video" > 25 {
                do give_badge "video_viewer"
            }

            on watch "pitch_video" >= 75 {
                do give_badge "engaged_prospect"
            }

            on abandon {
                do give_badge "cart_abandoner"
                do start_story "Cart Recovery"
            }

            on purchase "BuyForm" {
                do give_badge "customer"
                do remove_badge "cart_abandoner"
                do start_story "Customer Welcome"
                do jump_to_stage "Thank You"
            }
        }

        stage "Thank You" {
            path "/thank-you"

            page "Thanks" {
                block "confirmation" {
                    length short
                    prompt "Purchase confirmation"
                }
            }
        }
    }
}

story "Cart Recovery" {
    priority 3

    required_badges {
        must_have ["cart_abandoner"]
        must_not_have ["customer"]
    }

    storyline "Recovery Sequence" {
        order 1

        enactment "Hour 1" {
            scene "Forgot Something" {
                subject "Did you forget something?"
                body "<p>We noticed you did not complete your purchase. Your cart is still waiting.</p>"
                from_email "support@recovery.example.com"
                from_name "Support Team"
                reply_to "support@recovery.example.com"
            }
            on sent {
                within 1d
                do next_scene
            }
        }

        enactment "Day 1" {
            scene "Still Available" {
                subject "Your offer is still available"
                body "<p>The special pricing we offered is still active. Do not miss out.</p>"
                from_email "support@recovery.example.com"
                from_name "Support Team"
                reply_to "support@recovery.example.com"
            }
            on sent {
                within 2d
                do next_scene
            }
        }

        enactment "Day 3" {
            scene "Last Chance" {
                subject "Last chance: offer expires soon"
                body "<p>This is your final reminder. The special pricing expires at midnight.</p>"
                from_email "support@recovery.example.com"
                from_name "Support Team"
                reply_to "support@recovery.example.com"
            }
        }
    }
}

story "Customer Welcome" {
    priority 1

    required_badges {
        must_have ["customer"]
    }

    storyline "Welcome" {
        order 1

        enactment "Thanks" {
            scene "Welcome Email" {
                subject "Welcome! Your purchase is confirmed."
                body "<p>Thank you for your purchase! Here is how to get started.</p>"
                from_email "hello@recovery.example.com"
                from_name "Product Team"
                reply_to "hello@recovery.example.com"
            }
        }
    }
}
