# Combo: Event Promotion
#
# Event promotion combining: story interruption, persistent links,
# templates with vars, scenes range, and deferred transitions.
# Flow: Early Bird -> Regular -> Last Chance.

default sender {
    from_email "events@example.com"
    from_name "Event Team"
    reply_to "events@example.com"
}

story "Annual Conference" {
    priority 10
    allow_interruption false

    on_begin {
        give_badge "conf_prospect"
    }
    on_complete {
        give_badge "conf_registered"
        remove_badge "conf_prospect"
    }

    storyline "Early Bird" {
        order 1

        enactment "Early Bird Offer" {
            level 1
            order 1

            scene "Early Bird Email" {
                subject "{{first_name}}, early bird pricing ends soon!"
                body "<h1>Annual Conference</h1><p>Save 40% with early bird pricing. <a href='https://events.example.com/earlybird'>Register Now</a></p>"
                from_email "events@example.com"
                from_name "Event Team"
                reply_to "events@example.com"
                template "event_promo_v1"
                vars {
                    first_name: "Attendee"
                    discount: "40"
                    event_date: "March 15"
                }
            }

            on click "https://events.example.com/earlybird" {
                trigger_priority 2
                persist_scope "story"
                do give_badge "conf_registered"
                do mark_complete
            }

            on not_click "https://events.example.com/earlybird" {
                within 7d
                do send_immediate false
                do mark_complete
            }
        }
    }

    storyline "Regular Pricing" {
        order 2

        enactment "Regular Promo" {
            level 1
            order 1

            scenes 1..3 as n {
                scene "Regular Email ${n}" {
                    subject "Annual Conference — Register today (${n}/3)"
                    body "<p>Standard pricing available. <a href='https://events.example.com/register'>Register</a></p>"
                    from_email "events@example.com"
                    from_name "Event Team"
                    reply_to "events@example.com"
                }
            }

            on click "https://events.example.com/register" {
                trigger_priority 2
                persist_scope "storyline"
                do give_badge "conf_registered"
                do mark_complete
            }

            on not_open {
                within 2d
                trigger_priority 1
                do retry_scene up_to 1
                    else do next_scene
            }
        }
    }

    storyline "Last Chance" {
        order 3

        enactment "Urgency Push" {
            level 1
            order 1

            scene "Last Chance Email" {
                subject "LAST CHANCE — Annual Conference sells out tomorrow"
                body "<h1>Final Call</h1><p>Only a few spots left! <a href='https://events.example.com/lastchance'>Get Your Spot</a></p>"
                from_email "events@example.com"
                from_name "Event Team"
                reply_to "events@example.com"
                template "urgency_template"
                vars {
                    spots_remaining: "12"
                    deadline: "Tomorrow at midnight"
                }
            }

            on click "https://events.example.com/lastchance" {
                trigger_priority 2
                persist_scope "forever"
                do give_badge "conf_registered"
                do mark_complete
                send_immediate true
            }

            on not_click "https://events.example.com/lastchance" {
                within 1d
                do mark_failed
            }
        }
    }
}

# Lower-priority story that can be interrupted by conference promo
story "Monthly Newsletter" {
    priority 1
    allow_interruption true

    storyline "Monthly Content" {
        order 1

        enactment "Monthly Issue" {
            level 1
            order 1

            scene "Monthly Email" {
                subject "Your monthly update"
                body "<p>This month highlights: <a href='https://example.com/monthly'>Read</a></p>"
                from_email "news@example.com"
                from_name "Newsletter"
                reply_to "news@example.com"
            }

            on click "https://example.com/monthly" {
                do mark_complete
            }

            on not_click "https://example.com/monthly" {
                within 7d
                do mark_complete
            }
        }
    }
}
