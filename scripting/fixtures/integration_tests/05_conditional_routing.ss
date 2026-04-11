# 05 — Conditional Routing: storyline on_complete with conditional_route
#
# Tests conditional_route entries in on_complete blocks.
# After the Evaluation storyline completes, users are routed
# based on badges: VIP users go to "VIP Offer", others to "Standard Offer".

story "Offer Campaign" {
    priority 1

    storyline "Evaluation" {
        order 1

        on_complete {
            give_badge "evaluated"
            conditional_route {
                required_badges { must_have "vip" }
                next_storyline "VIP Offer"
                priority 10
            }
            conditional_route {
                required_badges { must_not_have "vip" }
                next_storyline "Standard Offer"
                priority 1
            }
        }

        enactment "Survey" {
            level 1
            order 1

            scene "Survey Email" {
                subject "Quick survey for you"
                body "<p>Tell us about yourself. <a href='https://example.com/survey'>Take Survey</a></p>"
                from_email "surveys@example.com"
                from_name "Survey Team"
                reply_to "surveys@example.com"
            }

            on click "https://example.com/survey" {
                trigger_priority 1
                do mark_complete
            }

            on not_click "https://example.com/survey" {
                within 5d
                do mark_complete
            }
        }
    }

    storyline "VIP Offer" {
        order 2
        required_badges { must_have "vip" }

        enactment "VIP Deal" {
            level 1
            order 1
            scene "VIP Email" {
                subject "Exclusive VIP offer"
                body "<p>Premium deal just for VIPs. <a href='https://example.com/vip-deal'>Claim</a></p>"
                from_email "vip@example.com"
                from_name "VIP Team"
                reply_to "vip@example.com"
            }
            on click "https://example.com/vip-deal" {
                do mark_complete
            }
        }
    }

    storyline "Standard Offer" {
        order 3

        enactment "Standard Deal" {
            level 1
            order 1
            scene "Standard Email" {
                subject "A special offer for you"
                body "<p>Check out this deal. <a href='https://example.com/deal'>View Deal</a></p>"
                from_email "offers@example.com"
                from_name "Offers Team"
                reply_to "offers@example.com"
            }
            on click "https://example.com/deal" {
                do mark_complete
            }
        }
    }
}
