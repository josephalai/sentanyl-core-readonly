# 49 — Order Bump and Upsell Funnel
#
# Tests advanced funnel economics:
# - order_bump inside checkout form
# - one_click_upsell form type
# - on purchase and on decline triggers
# - custom CRM fields in forms
# Includes companion story for integration test validation.

product "Zero Point Audio" {
    type "download"
    description "The zero point gravity audio experience."
}

product "Advanced Masterclass" {
    type "course"
    description "The advanced manifestation masterclass."
}

offer "Tripwire Audio Offer" {
    pricing_model one_time
    price 7.00
    currency "usd"

    includes_product "Zero Point Audio"
    grants_badge "audio_buyer"
}

offer "Etheric PDF Offer" {
    pricing_model one_time
    price 19.00
    currency "usd"

    grants_badge "pdf_buyer"
}

offer "Masterclass Upsell" {
    pricing_model one_time
    price 497.00
    currency "usd"

    includes_product "Advanced Masterclass"
    grants_badge "masterclass_student"
}

funnel "Tripwire Funnel" {
    domain "tripwire.example.com"

    route "Main" {
        order 1

        stage "Checkout" {
            path "/"

            page "Checkout Page" {
                block "sales_copy" {
                    length medium
                    prompt "Tripwire sales copy for the audio"
                }

                form "MainCheckout" {
                    type checkout
                    offer "Tripwire Audio Offer"
                    field email required
                    field card required

                    order_bump "Add the PDF" {
                        offer "Etheric PDF Offer"
                        text "Yes, add the PDF for only $19!"
                    }
                }

                form "Application" {
                    type lead_capture
                    field custom "manifestation_goal" required
                    field custom "current_roadblocks"
                }
            }

            on purchase "MainCheckout" {
                do give_badge "buyer"
                do jump_to_stage "Upsell"
            }
        }

        stage "Upsell" {
            path "/upsell"

            page "Upsell Page" {
                block "upsell_copy" {
                    length medium
                    prompt "Upsell copy for the masterclass"
                }

                form "OneClickBuy" {
                    type one_click_upsell
                    offer "Masterclass Upsell"
                }
            }

            on purchase "OneClickBuy" {
                do give_badge "vip"
                do jump_to_stage "Thanks"
            }

            on decline "OneClickBuy" {
                do jump_to_stage "Thanks"
            }
        }

        stage "Thanks" {
            path "/thank-you"

            page "Thank You Page" {
                block "thanks" {
                    length short
                    prompt "Thank you message"
                }
            }
        }
    }
}

story "Tripwire Follow-up" {
    priority 1

    storyline "Welcome" {
        order 1

        enactment "Welcome Email" {
            scene "Welcome" {
                subject "Welcome! Your audio is ready"
                body "<p>Thank you for your purchase.</p>"
                from_email "sales@tripwire.example.com"
                from_name "Sales Team"
            }
            on sent {
                within 1d
                do next_scene
            }
        }
    }
}
