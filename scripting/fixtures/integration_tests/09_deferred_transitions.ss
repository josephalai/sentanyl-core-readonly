# 09 — Deferred Transitions: send_immediate false + wait actions
#
# Tests deferred email sending. After a click, the engine waits
# before sending the next email rather than firing immediately.
# Combines send_immediate false with explicit wait durations.

story "Drip Sequence" {
    priority 1

    storyline "Nurture" {
        order 1

        enactment "Day 1" {
            level 1
            order 1

            scene "Day 1 Email" {
                subject "Welcome — Day 1"
                body "<p>Thanks for signing up! <a href='https://example.com/tip1'>Tip 1</a></p>"
                from_email "drip@example.com"
                from_name "Drip Team"
                reply_to "drip@example.com"
            }

            on click "https://example.com/tip1" {
                trigger_priority 2
                do send_immediate false
                do wait 1d
                do mark_complete
            }

            on not_click "https://example.com/tip1" {
                within 2d
                do send_immediate false
                do mark_complete
            }
        }

        enactment "Day 3" {
            level 2
            order 2

            scene "Day 3 Email" {
                subject "Day 3 — Going deeper"
                body "<p>Here is your next tip. <a href='https://example.com/tip2'>Tip 2</a></p>"
                from_email "drip@example.com"
                from_name "Drip Team"
                reply_to "drip@example.com"
            }

            on click "https://example.com/tip2" {
                trigger_priority 2
                do wait 2d
                do mark_complete
            }

            on not_click "https://example.com/tip2" {
                within 3d
                do send_immediate false
                do mark_complete
            }
        }

        enactment "Day 7" {
            level 3
            order 3

            scene "Day 7 Email" {
                subject "Day 7 — Ready to level up?"
                body "<p>Take the next step: <a href='https://example.com/upgrade'>Upgrade</a></p>"
                from_email "drip@example.com"
                from_name "Drip Team"
                reply_to "drip@example.com"
            }

            on click "https://example.com/upgrade" {
                do mark_complete
            }
        }
    }
}
