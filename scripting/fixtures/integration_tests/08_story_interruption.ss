# 08 — Story Interruption: Two stories with different priorities
#
# Tests allow_interruption flag. The low-priority newsletter
# can be interrupted by the high-priority flash sale. The flash
# sale does not allow interruption itself.

story "Weekly Newsletter" {
    priority 1
    allow_interruption true

    storyline "Content" {
        order 1

        enactment "Newsletter" {
            level 1
            order 1

            scene "Newsletter Email" {
                subject "This week in tech"
                body "<p>Your weekly roundup. <a href='https://example.com/read'>Read Now</a></p>"
                from_email "news@example.com"
                from_name "Newsletter"
                reply_to "news@example.com"
            }

            on click "https://example.com/read" {
                do mark_complete
            }

            on not_click "https://example.com/read" {
                within 7d
                do mark_complete
            }
        }
    }
}

story "Flash Sale" {
    priority 10
    allow_interruption false

    on_begin {
        give_badge "flash_sale_entered"
    }
    on_complete {
        remove_badge "flash_sale_entered"
        give_badge "flash_sale_seen"
    }

    storyline "Sale" {
        order 1

        enactment "Sale Announcement" {
            level 1
            order 1

            scene "Sale Email" {
                subject "FLASH SALE — 50% off everything!"
                body "<p>Hurry, limited time! <a href='https://example.com/sale'>Shop Now</a></p>"
                from_email "deals@example.com"
                from_name "Deals Team"
                reply_to "deals@example.com"
            }

            on click "https://example.com/sale" {
                trigger_priority 1
                do give_badge "sale_clicked"
                do mark_complete
            }

            on not_click "https://example.com/sale" {
                within 1d
                do mark_complete
            }
        }
    }
}
