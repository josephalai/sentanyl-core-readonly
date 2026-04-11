# 21 — Storyline on_fail with conditional_route Routing
#
# Tests on_fail blocks with conditional_route entries on storylines.
# When the main flow fails, premium users are routed to a premium
# recovery path while standard users go to a basic recovery path.

story "Recovery Flows" {
    priority 1

    storyline "Main Offer" {
        order 1

        on_fail {
            give_badge "offer_failed"
            conditional_route {
                required_badges { must_have "premium" }
                next_storyline "Premium Recovery"
                priority 2
            }
            conditional_route {
                required_badges { must_not_have "premium" }
                next_storyline "Basic Recovery"
                priority 1
            }
        }

        enactment "Main Offer Email" {
            level 1
            order 1

            scene "Offer" {
                subject "Exclusive offer just for you"
                body "<p>Grab this deal: <a href='https://example.com/deal'>Claim</a></p>"
                from_email "deals@example.com"
                from_name "Deals"
                reply_to "deals@example.com"
            }

            on click "https://example.com/deal" {
                trigger_priority 1
                do mark_complete
            }

            on not_click "https://example.com/deal" {
                within 2d
                do mark_failed
            }
        }
    }

    storyline "Premium Recovery" {
        order 2
        required_badges { must_have "premium" }

        enactment "Premium Rescue" {
            level 1
            order 1
            scene "Premium Email" {
                subject "Premium member — we have a better deal"
                body "<p>As a premium member: <a href='https://example.com/premium-deal'>Special Offer</a></p>"
                from_email "deals@example.com"
                from_name "Premium Team"
                reply_to "deals@example.com"
            }
            on click "https://example.com/premium-deal" {
                do mark_complete
            }
        }
    }

    storyline "Basic Recovery" {
        order 3

        enactment "Basic Rescue" {
            level 1
            order 1
            scene "Basic Email" {
                subject "We noticed you missed our offer"
                body "<p>Here is another chance: <a href='https://example.com/basic-deal'>Try Again</a></p>"
                from_email "deals@example.com"
                from_name "Deals"
                reply_to "deals@example.com"
            }
            on click "https://example.com/basic-deal" {
                do mark_complete
            }
        }
    }
}
