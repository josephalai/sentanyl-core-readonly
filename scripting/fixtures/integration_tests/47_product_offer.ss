# 47 — Product and Offer Declarations
#
# Tests the Product vs Offer architecture:
# - Products have NO price (deliverable only)
# - Offers define pricing, badge grants, and product bundles
# - Includes companion story for integration test validation.

product "Manifestation Masterclass Content" {
    type "course"
    description "The core 6-week program."
}

product "Etheric Science PDF" {
    type "download"
    description "A deep-dive PDF on etheric science."
}

offer "Masterclass VIP Bundle" {
    pricing_model one_time
    price 497.00
    currency "usd"

    includes_product "Manifestation Masterclass Content"
    includes_product "Etheric Science PDF"
    grants_badge "vip_student"
    grants_badge "buyer"

    on purchase {
        do give_badge "purchased_masterclass"
    }
}

offer "Free Intro Offer" {
    pricing_model free
    currency "usd"

    includes_product "Etheric Science PDF"
    grants_badge "free_user"
}

funnel "Product Funnel" {
    domain "products.example.com"

    route "Main" {
        order 1

        stage "Sales" {
            path "/"

            page "Sales Page" {
                block "hero" {
                    length medium
                    prompt "Sales copy for manifestation course"
                }

                form "BuyNow" {
                    type checkout
                    offer "Masterclass VIP Bundle"
                    field email required
                    field card required
                }
            }

            on purchase "BuyNow" {
                do give_badge "buyer"
                do jump_to_stage "Thanks"
            }
        }

        stage "Thanks" {
            path "/thank-you"

            page "Thank You" {
                block "confirmation" {
                    length short
                    prompt "Thank you for purchasing"
                }
            }
        }
    }
}

story "Product Welcome" {
    priority 1

    storyline "Welcome" {
        order 1

        enactment "Welcome Email" {
            scene "Welcome" {
                subject "Welcome to the Masterclass"
                body "<p>Thank you for purchasing.</p>"
                from_email "team@products.example.com"
                from_name "Manifestation Team"
            }
            on sent {
                within 1d
                do next_scene
            }
        }
    }
}
