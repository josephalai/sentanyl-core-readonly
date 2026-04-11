# 23 — Condition Guards: when has_badge, has_tag, and/or/not
#
# Tests all condition types on triggers: has_badge, not_has_badge,
# has_tag, not_has_tag, and compound conditions using and/or/not.

default sender {
    from_email "cond@example.com"
    from_name "Condition Test"
    reply_to "cond@example.com"
}

story "Condition Guards" {
    use sender default
    priority 1

    storyline "Guarded Actions" {
        order 1

        enactment "Simple Guards" {
            level 1
            order 1
            scene "Email" {
                subject "Condition test"
                body "<a href='https://example.com/a'>A</a> <a href='https://example.com/b'>B</a>"
            }

            on click "https://example.com/a" {
                trigger_priority 4
                when has_badge "vip"
                do give_badge "vip_clicked"
                do mark_complete
            }
            on click "https://example.com/b" {
                trigger_priority 3
                when not_has_badge "vip"
                do mark_complete
            }
            on click "https://example.com/a" {
                trigger_priority 2
                when has_tag "subscriber"
                do give_badge "sub_clicked"
            }
            on click "https://example.com/b" {
                trigger_priority 1
                when not_has_tag "subscriber"
                do next_scene
            }
        }

        enactment "Compound Guards" {
            level 2
            order 2
            scene "Email" {
                subject "Compound conditions"
                body "<a href='https://example.com/c'>C</a>"
            }

            on click "https://example.com/c" {
                trigger_priority 3
                when and {
                    has_badge "vip"
                    has_tag "subscriber"
                }
                do give_badge "elite"
                do mark_complete
            }
            on click "https://example.com/c" {
                trigger_priority 2
                when or {
                    has_badge "vip"
                    has_tag "early_adopter"
                }
                do give_badge "priority_user"
                do mark_complete
            }
            on click "https://example.com/c" {
                trigger_priority 1
                when not has_badge "banned"
                do mark_complete
            }
        }
    }
}
