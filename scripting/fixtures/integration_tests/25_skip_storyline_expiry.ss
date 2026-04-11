# 25 — Skip-to-Next-Storyline on Expiry
#
# Tests the skip_to_next_storyline_on_expiry flag on enactments.
# When set to true, if the enactment expires (all triggers time out),
# the engine skips to the next storyline instead of failing.

story "Expiry Handling" {
    priority 1

    storyline "Time-Limited Offer" {
        order 1

        enactment "Flash Deal" {
            level 1
            order 1
            skip_to_next_storyline_on_expiry true

            scene "Flash Email" {
                subject "24-hour flash deal"
                body "<p>This deal expires soon! <a href='https://example.com/flash'>Buy Now</a></p>"
                from_email "flash@example.com"
                from_name "Flash Sales"
                reply_to "flash@example.com"
            }

            on click "https://example.com/flash" {
                trigger_priority 1
                do give_badge "flash_buyer"
                do mark_complete
            }

            on not_click "https://example.com/flash" {
                within 1d
                do mark_failed
            }
        }
    }

    storyline "Regular Catalog" {
        order 2

        enactment "Catalog Browse" {
            level 1
            order 1

            scene "Catalog Email" {
                subject "Browse our full catalog"
                body "<p>See everything we offer: <a href='https://example.com/catalog'>Browse</a></p>"
                from_email "shop@example.com"
                from_name "Shop Team"
                reply_to "shop@example.com"
            }

            on click "https://example.com/catalog" {
                do mark_complete
            }
        }
    }
}
