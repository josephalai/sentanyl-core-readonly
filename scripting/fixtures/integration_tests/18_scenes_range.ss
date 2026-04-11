# 18 — Scenes Range: scenes 1..N range generation
#
# Tests the scenes 1..N as n { ... } construct that generates
# multiple scenes from a single template. Produces 5 emails
# in a single enactment, each with an interpolated index.

story "Five-Part Series" {
    priority 1

    storyline "Series" {
        order 1

        enactment "Daily Tips" {
            level 1
            order 1

            scenes 1..5 as n {
                scene "Tip ${n}" {
                    subject "Daily Tip ${n} of 5"
                    body "<h1>Tip ${n}</h1><p>Here is tip number ${n} for you.</p>"
                    from_email "tips@example.com"
                    from_name "Tips Team"
                    reply_to "tips@example.com"
                }
            }

            on open {
                trigger_priority 2
                do give_badge "tip_reader"
            }

            on not_open {
                within 2d
                trigger_priority 1
                do retry_scene up_to 1
                    else do next_scene
            }
        }
    }
}
