# 04 — All 20 Action Types
#
# Tests every action the DSL supports:
# next_scene, prev_scene, jump_to_enactment, jump_to_storyline,
# next_enactment, advance_to_next_storyline, end_story,
# mark_complete, mark_failed, unsubscribe, give_badge,
# remove_badge, wait, send_immediate, retry_scene, retry_enactment,
# loop_to_enactment, loop_to_storyline, loop_to_start_enactment,
# loop_to_start_storyline.

default sender {
    from_email "actions@example.com"
    from_name "Action Test"
    reply_to "actions@example.com"
}

story "Action Showcase" {
    use sender default
    priority 1

    storyline "Navigation" {
        order 1

        enactment "Scene Nav" {
            level 1
            order 1
            scene "Scene A" { subject "Nav A" body "<p>A</p>" }
            scene "Scene B" { subject "Nav B" body "<p>B</p>" }

            on click "https://example.com/next" {
                trigger_priority 6
                do next_scene
            }
            on click "https://example.com/prev" {
                trigger_priority 5
                do prev_scene
            }
            on click "https://example.com/jump-e" {
                trigger_priority 4
                do jump_to_enactment "Badge Ops"
            }
            on click "https://example.com/jump-s" {
                trigger_priority 3
                do jump_to_storyline "Loops"
            }
            on click "https://example.com/next-e" {
                trigger_priority 2
                do next_enactment "Badge Ops"
            }
            on click "https://example.com/advance" {
                trigger_priority 1
                do advance_to_next_storyline
            }
        }

        enactment "Badge Ops" {
            level 2
            order 2
            scene "Email" { subject "Badge Ops" body "<p>Badges</p>" }

            on click "https://example.com/give" {
                trigger_priority 5
                do give_badge "earned"
                do remove_badge "pending"
            }
            on click "https://example.com/end" {
                trigger_priority 4
                do end_story
            }
            on click "https://example.com/done" {
                trigger_priority 3
                do mark_complete
            }
            on click "https://example.com/fail" {
                trigger_priority 2
                do mark_failed
            }
            on click "https://example.com/unsub" {
                trigger_priority 1
                do unsubscribe
            }
        }

        enactment "Timing" {
            level 3
            order 3
            scene "Email" { subject "Timing" body "<p>Wait and delay</p>" }

            on click "https://example.com/wait" {
                trigger_priority 2
                do wait 2h
            }
            on click "https://example.com/delay" {
                trigger_priority 1
                do send_immediate false
            }
        }
    }

    storyline "Loops" {
        order 2

        enactment "Retry Actions" {
            level 1
            order 1
            scene "Email" { subject "Retries" body "<p>Retry tests</p>" }

            on not_open {
                within 1d
                trigger_priority 2
                do retry_scene up_to 2 times
                    else do mark_failed
            }
            on not_click "https://example.com/act" {
                within 2d
                trigger_priority 1
                do retry_enactment up_to 3 times
                    else do advance_to_next_storyline
            }
        }

        enactment "Loop Actions" {
            level 2
            order 2
            scene "Email" { subject "Loops" body "<p>Loop tests</p>" }

            on not_open {
                within 1d
                trigger_priority 4
                do loop_to_enactment "Retry Actions" up_to 2
                    else do mark_failed
            }
            on not_click "https://example.com/go" {
                within 2d
                trigger_priority 3
                do loop_to_storyline "Navigation" up_to 1
                    else do end_story
            }
            on click "https://example.com/restart-e" {
                trigger_priority 2
                do loop_to_start_enactment up_to 3
                    else do mark_failed
            }
            on click "https://example.com/restart-s" {
                trigger_priority 1
                do loop_to_start_storyline up_to 2
                    else do end_story
            }
        }
    }
}
