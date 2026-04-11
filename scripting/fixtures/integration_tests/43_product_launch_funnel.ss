# 43 — Product Launch Funnel (PLF-Style)
#
# Product Launch Formula-style funnel: video series → cart open → checkout.
# Tests multiple video blocks with different watch thresholds,
# sequential stages, badge accumulation, and checkout form.

funnel "Product Launch" {
    domain "launch.example.com"

    route "Launch Sequence" {
        order 1

        stage "Video 1" {
            path "/video-1"

            page "PLC Video 1" {
                template "minimal_v1"

                block "plc_video_1" {
                    type video
                    source_url "https://cdn.example.com/plc-1.mp4"
                    autoplay false
                }

                block "video_1_notes" {
                    length medium
                    prompt "Key takeaways from video 1"
                }

                form "Video1Lead" {
                    field email required
                    field first_name
                }
            }

            on watch "plc_video_1" > 50 {
                do give_badge "watched_plc1"
            }

            on submit "Video1Lead" {
                do give_badge "launch_subscriber"
                do start_story "Launch Email Series"
                do jump_to_stage "Video 2"
            }
        }

        stage "Video 2" {
            path "/video-2"

            page "PLC Video 2" {
                block "plc_video_2" {
                    type video
                    source_url "https://cdn.example.com/plc-2.mp4"
                    autoplay false
                }

                block "video_2_notes" {
                    length medium
                    prompt "Transformation stories and case studies"
                }
            }

            on watch "plc_video_2" > 50 {
                do give_badge "watched_plc2"
            }
        }

        stage "Video 3" {
            path "/video-3"

            page "PLC Video 3" {
                block "plc_video_3" {
                    type video
                    source_url "https://cdn.example.com/plc-3.mp4"
                    autoplay false
                }

                block "video_3_notes" {
                    length medium
                    prompt "The big reveal and what is coming"
                }
            }

            on watch "plc_video_3" > 75 {
                do give_badge "watched_plc3"
                do give_badge "launch_ready"
            }
        }

        stage "Cart Open" {
            path "/buy"

            page "Sales Page" {
                block "sales_copy" {
                    length long
                    prompt "Full sales copy with benefits, testimonials, and guarantee"
                }

                form "Purchase" {
                    type checkout
                    product_id "flagship-course"
                    field email required
                    field first_name
                    field card required
                }
            }

            on purchase "Purchase" {
                do give_badge "buyer"
                do start_story "Buyer Onboarding"
                do jump_to_stage "Welcome"
            }
        }

        stage "Welcome" {
            path "/welcome"

            page "Welcome Page" {
                block "welcome_message" {
                    length short
                    prompt "Purchase confirmation and next steps"
                }
            }
        }
    }
}

story "Launch Email Series" {
    priority 1

    storyline "Pre-Launch" {
        order 1

        enactment "Video 1 Sent" {
            scene "Video 1 Ready" {
                subject "Video 1 is live!"
                body "<p>The first video in our series is now available. Watch it now.</p>"
                from_email "launch@example.com"
                from_name "Launch Team"
                reply_to "launch@example.com"
            }
            on sent {
                within 3d
                do next_scene
            }
        }

        enactment "Video 2 Sent" {
            scene "Video 2 Ready" {
                subject "Video 2 is live — do not miss this!"
                body "<p>The second video reveals the transformation stories.</p>"
                from_email "launch@example.com"
                from_name "Launch Team"
                reply_to "launch@example.com"
            }
            on sent {
                within 3d
                do next_scene
            }
        }

        enactment "Video 3 Sent" {
            scene "Video 3 Ready" {
                subject "Final video: The big reveal"
                body "<p>This is it. Watch the final video before the cart opens.</p>"
                from_email "launch@example.com"
                from_name "Launch Team"
                reply_to "launch@example.com"
            }
        }
    }
}

story "Buyer Onboarding" {
    priority 2

    required_badges {
        must_have ["buyer"]
    }

    storyline "Getting Started" {
        order 1

        enactment "Welcome" {
            scene "Buyer Welcome" {
                subject "Welcome aboard! Here is your access."
                body "<p>Congratulations on your purchase! Here is everything you need to get started.</p>"
                from_email "support@example.com"
                from_name "Support Team"
                reply_to "support@example.com"
            }
        }
    }
}
