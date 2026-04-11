# 27 — Compound Conditions: combined badge AND/OR on triggers
#
# Tests compound conditions using and, or, and not blocks
# to create complex trigger guards combining badge and tag checks.

default sender {
    from_email "compound@example.com"
    from_name "Compound Test"
    reply_to "compound@example.com"
}

story "Compound Conditions" {
    use sender default
    priority 1

    storyline "Segmented Actions" {
        order 1

        enactment "Multi-Condition" {
            level 1
            order 1
            scene "Email" {
                subject "Segmented offer"
                body "<a href='https://example.com/offer'>Claim Offer</a>"
            }

            # AND: must be VIP and subscriber
            on click "https://example.com/offer" {
                trigger_priority 5
                when and {
                    has_badge "vip"
                    has_badge "verified"
                    has_tag "subscriber"
                }
                do give_badge "elite_offer"
                do mark_complete
            }

            # OR: VIP or early adopter
            on click "https://example.com/offer" {
                trigger_priority 4
                when or {
                    has_badge "vip"
                    has_tag "early_adopter"
                }
                do give_badge "priority_offer"
                do mark_complete
            }

            # NOT: not banned
            on click "https://example.com/offer" {
                trigger_priority 3
                when not has_badge "banned"
                do give_badge "standard_offer"
                do mark_complete
            }

            # Chained: has_badge AND not_has_badge
            on click "https://example.com/offer" {
                trigger_priority 2
                when has_badge "member" and not_has_badge "churned"
                do mark_complete
            }

            # Chained: has_badge OR has_tag
            on click "https://example.com/offer" {
                trigger_priority 1
                when has_badge "trial" or has_tag "interested"
                do next_scene
            }
        }
    }
}
