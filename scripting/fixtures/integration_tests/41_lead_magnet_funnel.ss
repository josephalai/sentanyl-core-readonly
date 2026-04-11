# 41 — Lead Magnet Funnel
#
# Classic lead magnet funnel: opt-in page → download delivery → email nurture.
# Tests funnel + story combination, jump_to_stage, give_badge on submit,
# and start_story to kick off an email drip after web opt-in.

funnel "Free Guide Funnel" {
    domain "guides.example.com"

    route "Public" {
        order 1

        stage "Opt-In" {
            path "/free-guide"

            page "Download Page" {
                template "minimal_v1"

                block "headline" {
                    length short
                    prompt "Attention-grabbing headline for free guide download"
                }

                block "benefits" {
                    length medium
                    prompt "3 key benefits of downloading the guide"
                }

                form "GuideForm" {
                    field email required
                    field first_name
                }
            }

            on submit "GuideForm" {
                do give_badge "guide_subscriber"
                do start_story "Guide Nurture Drip"
                do jump_to_stage "Download"
            }
        }

        stage "Download" {
            path "/download"

            page "Download Confirmation" {
                template "minimal_v1"

                block "thank_you" {
                    length short
                    prompt "Thank you and download instructions"
                }

                block "next_steps" {
                    length short
                    prompt "What to expect next in their inbox"
                }
            }
        }
    }
}

story "Guide Nurture Drip" {
    priority 1
    use sender default

    on_begin {
        give_badge "nurture_enrolled"
    }

    on_complete {
        give_badge "nurture_complete"
        remove_badge "nurture_enrolled"
    }

    storyline "Education" {
        order 1

        enactment "Day 1" {
            scene "Welcome" {
                subject "Your free guide is ready!"
                body "<p>Thanks for downloading our guide. Here is your first tip.</p>"
                from_email "hello@guides.example.com"
                from_name "Guide Team"
                reply_to "hello@guides.example.com"
            }
            on sent {
                within 2d
                do next_scene
            }
        }

        enactment "Day 3" {
            scene "Tip 2" {
                subject "Tip #2 from your guide"
                body "<p>Here is the second key takeaway from the guide.</p>"
                from_email "hello@guides.example.com"
                from_name "Guide Team"
                reply_to "hello@guides.example.com"
            }
            on sent {
                within 2d
                do next_scene
            }
        }

        enactment "Day 5" {
            scene "Offer" {
                subject "Ready for the next step?"
                body "<p>You have learned the basics. Ready to go deeper?</p>"
                from_email "hello@guides.example.com"
                from_name "Guide Team"
                reply_to "hello@guides.example.com"
            }
        }
    }
}

default sender {
    from_email "hello@guides.example.com"
    from_name "Guide Team"
    reply_to "hello@guides.example.com"
}
