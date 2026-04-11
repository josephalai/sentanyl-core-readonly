# 02 — Storyline Badge Gating: required_badges with must_have / must_not_have
#
# Tests that storylines can gate entry based on user badges.
# "Advanced Track" requires the "basics_done" badge and forbids
# the "banned" badge, ensuring only qualified users enter.

story "Gated Course" {
    priority 1

    required_badges {
        must_not_have "already_completed"
    }

    storyline "Basics" {
        order 1

        on_complete {
            give_badge "basics_done"
        }

        enactment "Basics Lesson" {
            level 1
            order 1

            scene "Basics Email" {
                subject "Start with the basics"
                body "<p>Learn the fundamentals. <a href='https://example.com/basics'>Begin</a></p>"
                from_email "learn@example.com"
                from_name "Course Team"
                reply_to "learn@example.com"
            }

            on click "https://example.com/basics" {
                do mark_complete
            }
        }
    }

    storyline "Advanced Track" {
        order 2
        required_badges { must_have "basics_done" must_not_have "banned" }

        enactment "Advanced Lesson" {
            level 1
            order 1

            scene "Advanced Email" {
                subject "Ready for advanced material"
                body "<p>Dive deeper. <a href='https://example.com/advanced'>Continue</a></p>"
                from_email "learn@example.com"
                from_name "Course Team"
                reply_to "learn@example.com"
            }

            on click "https://example.com/advanced" {
                do mark_complete
            }
        }
    }
}
