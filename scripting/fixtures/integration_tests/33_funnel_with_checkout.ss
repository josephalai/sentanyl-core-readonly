# 33 — Funnel with Checkout
#
# Tests a funnel with checkout form type, product_id,
# purchase trigger, and multiple actions including
# start_story. Includes companion story for integration test.

funnel "Checkout Funnel" {
    domain "shop.example.com"

    route "Buyers" {
        order 1

        stage "Product" {
            path "/product"

            page "Product Page" {
                template "tripwire_v1"

                block "product_info" {
                    length medium
                    prompt "Product description"
                }

                form "Checkout" {
                    type checkout
                    product_id "prod-001"
                    field email required
                    field card required
                }
            }

            on purchase "Checkout" {
                do give_badge "buyer"
                do jump_to_stage "Confirmation"
                do start_story "Upsell Sequence"
            }
        }

        stage "Confirmation" {
            path "/confirmation"

            page "Order Confirmed" {
                template "minimal_v1"

                block "thank_you" {
                    length short
                    prompt "Order confirmation"
                }
            }
        }
    }
}

story "Upsell Sequence" {
    priority 1

    storyline "Upsell" {
        order 1

        enactment "Upsell Email" {
            scene "Upsell" {
                subject "Special offer just for you"
                body "<p>As a valued buyer, here is an exclusive deal.</p>"
                from_email "shop@shop.example.com"
                from_name "Shop Team"
            }
            on sent {
                within 1d
                do next_scene
            }
        }
    }
}
