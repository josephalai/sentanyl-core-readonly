# 34 — Funnel with Multiple Routes
#
# Tests multi-route funnel with badge gating
# (must_have_badge / must_not_have_badge), abandon trigger
# with timeout, checkout forms, and redirect actions.
# Includes companion story for integration test.

funnel "Multi Route Funnel" {
    domain "marketing.example.com"

    route "Cold Traffic" {
        order 1
        must_not_have_badge "lead"

        stage "OptIn" {
            path "/free-guide"

            page "Squeeze Page" {
                template "minimal_v1"

                block "headline" {
                    length short
                    prompt "Attention grabbing headline"
                }

                form "LeadCapture" {
                    field email required
                    field first_name
                }
            }

            on submit "LeadCapture" {
                do give_badge "lead"
                do jump_to_stage "Tripwire"
            }

            on abandon {
                within 30m
                do send_email "reminder"
            }
        }

        stage "Tripwire" {
            path "/special-offer"

            page "Tripwire Offer" {
                template "tripwire_v1"

                form "TripwireCheckout" {
                    type checkout
                    product_id "audio-7"
                    field email required
                    field card required
                }
            }

            on purchase "TripwireCheckout" {
                do give_badge "buyer"
                do redirect "/thank-you"
            }
        }
    }

    route "Warm Traffic" {
        order 2
        must_have_badge "lead"

        stage "DirectOffer" {
            path "/vip-offer"

            page "VIP Offer" {
                template "tripwire_v1"

                block "vip_content" {
                    length long
                    prompt "VIP exclusive offer content"
                }

                form "VIPCheckout" {
                    type checkout
                    product_id "vip-bundle"
                    field email required
                    field card required
                }
            }

            on purchase "VIPCheckout" {
                do give_badge "vip"
                do start_story "VIP Sequence"
            }
        }
    }
}

story "VIP Sequence" {
    priority 1

    storyline "VIP Welcome" {
        order 1

        enactment "VIP Email" {
            scene "VIP Welcome" {
                subject "Welcome to the VIP club!"
                body "<p>You now have exclusive VIP access.</p>"
                from_email "vip@marketing.example.com"
                from_name "VIP Team"
            }
            on sent {
                within 1d
                do next_scene
            }
        }
    }
}
