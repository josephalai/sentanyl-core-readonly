# 11 — Scene Template + Vars Block + Handlebars
#
# Tests the template keyword on scenes combined with vars blocks.
# The vars block provides data for Handlebars {{...}} placeholders
# in the subject and body.

story "Template Demo" {
    priority 1

    storyline "Templated Content" {
        order 1

        enactment "Welcome Template" {
            level 1
            order 1

            scene "Welcome Email" {
                subject "Welcome, {{first_name}}!"
                body "<h1>Hello {{first_name}}!</h1><p>Your code: {{promo_code}}</p>"
                from_email "hello@example.com"
                from_name "{{company_name}}"
                reply_to "support@example.com"
                template "welcome_v2"
                vars {
                    first_name: "Jane"
                    company_name: "Acme Corp"
                    promo_code: "WELCOME10"
                    hero_image: "https://example.com/hero.png"
                }
            }

            on click "https://example.com/start" {
                do mark_complete
            }
        }

        enactment "Offer Template" {
            level 2
            order 2

            scene "Offer Email" {
                subject "{{first_name}}, a deal on {{product_name}}"
                body "<p>Get {{discount}}% off {{product_name}}!</p>"
                from_email "deals@example.com"
                from_name "Deals"
                reply_to "deals@example.com"
                template "offer_v1"
                vars {
                    first_name: "Jane"
                    product_name: "Premium Widget"
                    discount: "25"
                }
            }

            on click "https://example.com/offer" {
                do mark_complete
            }
        }
    }
}
