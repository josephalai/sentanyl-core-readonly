# 19 — Multiple Stories in One Script
#
# Tests defining multiple independent stories in a single file.
# Each story has its own priority and runs independently.

default sender {
    from_email "multi@example.com"
    from_name "Multi Story"
    reply_to "multi@example.com"
}

story "Story Alpha" {
    use sender default
    priority 1

    on_complete {
        give_badge "alpha_done"
    }

    storyline "Alpha Track" {
        order 1

        enactment "Alpha Step" {
            level 1
            order 1
            scene "Alpha Email" {
                subject "Welcome to Alpha"
                body "<p>You are in the Alpha track. <a href='https://example.com/alpha'>Go</a></p>"
            }
            on click "https://example.com/alpha" {
                do mark_complete
            }
        }
    }
}

story "Story Beta" {
    use sender default
    priority 2

    on_complete {
        give_badge "beta_done"
    }

    storyline "Beta Track" {
        order 1

        enactment "Beta Step" {
            level 1
            order 1
            scene "Beta Email" {
                subject "Welcome to Beta"
                body "<p>You are in the Beta track. <a href='https://example.com/beta'>Go</a></p>"
            }
            on click "https://example.com/beta" {
                do mark_complete
            }
        }
    }
}

story "Story Gamma" {
    use sender default
    priority 3

    storyline "Gamma Track" {
        order 1

        enactment "Gamma Step" {
            level 1
            order 1
            scene "Gamma Email" {
                subject "Welcome to Gamma"
                body "<p>You are in the Gamma track. <a href='https://example.com/gamma'>Go</a></p>"
            }
            on click "https://example.com/gamma" {
                do mark_complete
            }
        }
    }
}
