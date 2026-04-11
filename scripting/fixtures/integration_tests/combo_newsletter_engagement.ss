# Combo: Newsletter Engagement
#
# Newsletter engagement combining: open/click branching,
# VIP segmentation via badges, retry on non-opens,
# unsubscribe handling, conditional routing, and multi-scene drip.

default sender {
    from_email "newsletter@example.com"
    from_name "Weekly Digest"
    reply_to "editors@example.com"
}

story "Newsletter Engagement" {
    use sender default
    priority 3

    on_begin {
        give_badge "newsletter_subscriber"
    }

    storyline "Weekly Send" {
        order 1

        on_complete {
            give_badge "weekly_engaged"
            conditional_route {
                required_badges { must_have "vip_reader" }
                next_storyline "VIP Content"
                priority 10
            }
            conditional_route {
                required_badges { must_not_have "vip_reader" }
                next_storyline "Re-Engagement"
                priority 1
            }
        }

        enactment "Issue" {
            level 1
            order 1

            scene "Newsletter Issue 1" {
                subject "This Week: Top Stories"
                body "<h1>Weekly Digest</h1><p>Top story: <a href='https://example.com/story1'>Read</a></p>"
            }
            scene "Newsletter Issue 2" {
                subject "This Week: Deep Dive"
                body "<h1>Deep Dive</h1><p>Featured analysis: <a href='https://example.com/story2'>Read</a></p>"
            }

            on open {
                trigger_priority 4
                do give_badge "opener"
                do next_scene
            }
            on click "https://example.com/story1" {
                trigger_priority 3
                do give_badge "vip_reader"
                do mark_complete
            }
            on not_open {
                within 2d
                trigger_priority 2
                do retry_scene up_to 2 times
                    else do mark_complete
            }
            on unsubscribe {
                trigger_priority 1
                do unsubscribe
                do end_story
            }
        }
    }

    storyline "VIP Content" {
        order 2
        required_badges { must_have "vip_reader" }

        enactment "VIP Issue" {
            level 1
            order 1

            scene "VIP Email" {
                subject "Exclusive: VIP-only content"
                body "<p>Exclusive insights: <a href='https://example.com/vip'>Read VIP Content</a></p>"
            }

            on click "https://example.com/vip" {
                trigger_priority 2
                do give_badge "vip_engaged"
                do mark_complete
            }
            on not_click "https://example.com/vip" {
                within 3d
                do mark_complete
            }
        }
    }

    storyline "Re-Engagement" {
        order 3

        enactment "Win Back" {
            level 1
            order 1

            scene "Re-Engage Email" {
                subject "We miss you! Here is what you are missing"
                body "<p>Come back: <a href='https://example.com/comeback'>See Highlights</a></p>"
            }

            on click "https://example.com/comeback" {
                trigger_priority 2
                do give_badge "re_engaged"
                do mark_complete
            }
            on not_click "https://example.com/comeback" {
                within 5d
                do mark_failed
            }
            on unsubscribe {
                do unsubscribe
                do end_story
            }
        }
    }
}
