# 06 — Click/Not-Click Branching with jump_to_enactment
#
# Tests click and not_click triggers that branch to different
# enactments within the same storyline. Clicking the info link
# jumps to the Soft Sell; not clicking jumps to Hard Sell.

story "Sales Funnel" {
    priority 1

    storyline "Sales Track" {
        order 1

        enactment "Intro" {
            level 1
            order 1

            scene "Intro Email" {
                subject "Check out what we have for you"
                body "<p>Learn more about our product: <a href='https://example.com/info'>Get Info</a></p>"
                from_email "sales@example.com"
                from_name "Sales Team"
                reply_to "sales@example.com"
            }

            on click "https://example.com/info" {
                trigger_priority 2
                mark_complete true
                within 2d
                do give_badge "showed_interest"
                do jump_to_enactment "Soft Sell"
            }

            on not_click "https://example.com/info" {
                trigger_priority 1
                within 2d
                do jump_to_enactment "Hard Sell"
            }
        }

        enactment "Soft Sell" {
            level 2
            order 2

            scene "Soft Sell Email" {
                subject "Ready to get started?"
                body "<p>Since you showed interest, here is a gentle nudge: <a href='https://example.com/buy'>Buy Now</a></p>"
                from_email "sales@example.com"
                from_name "Sales Team"
                reply_to "sales@example.com"
            }

            on click "https://example.com/buy" {
                do give_badge "purchased"
                do mark_complete
            }

            on not_click "https://example.com/buy" {
                within 3d
                do mark_failed
            }
        }

        enactment "Hard Sell" {
            level 3
            order 3

            scene "Hard Sell Email" {
                subject "Last chance — do not miss out!"
                body "<p>Final opportunity: <a href='https://example.com/buy-now'>Buy Now</a></p>"
                from_email "sales@example.com"
                from_name "Sales Team"
                reply_to "sales@example.com"
            }

            on click "https://example.com/buy-now" {
                do give_badge "purchased"
                do mark_complete
            }

            on not_click "https://example.com/buy-now" {
                within 2d
                do mark_failed
            }
        }
    }
}
