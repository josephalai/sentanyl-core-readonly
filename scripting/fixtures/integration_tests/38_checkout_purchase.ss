# 38 — Checkout and Purchase Flow
#
# Tests checkout form type, product_id configuration,
# on purchase trigger, and badge granting on purchase.
# Includes companion story for post-purchase follow-up.

funnel "Product Sales" {
    domain "shop.example.com"

    route "Main" {
        order 1

        stage "Sales Page" {
            path "/buy"

            page "Buy Page" {
                template "tripwire_v1"

                block "sales_copy" {
                    length long
                    prompt "Persuasive sales copy for digital product"
                }

                form "PurchaseForm" {
                    type checkout
                    product_id "digital-course-v1"
                    field email required
                    field first_name
                    field card required
                }
            }

            on purchase "PurchaseForm" {
                do give_badge "buyer"
                do jump_to_stage "Thank You"
                do start_story "Post-Purchase Onboarding"
            }

            on abandon {
                do give_badge "cart_abandoner"
            }
        }

        stage "Thank You" {
            path "/thank-you"

            page "Confirmation" {
                block "thanks_message" {
                    length short
                    prompt "Purchase confirmation and next steps"
                }
            }
        }
    }
}

story "Post-Purchase Onboarding" {
    priority 1

    storyline "Welcome" {
        order 1

        enactment "Day 1" {
            scene "Welcome Buyer" {
                subject "Welcome to your course!"
                body "<p>Thank you for your purchase. Here is how to get started.</p>"
                from_email "support@shop.example.com"
                from_name "Shop Support"
                reply_to "support@shop.example.com"
            }

            on sent {
                within 2d
                do next_scene
            }
        }

        enactment "Day 3" {
            scene "Check In" {
                subject "How is it going?"
                body "<p>Just checking in on your progress.</p>"
                from_email "support@shop.example.com"
                from_name "Shop Support"
                reply_to "support@shop.example.com"
            }
        }
    }
}
