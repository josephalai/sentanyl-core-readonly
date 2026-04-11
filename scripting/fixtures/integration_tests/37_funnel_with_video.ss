# 37 — Funnel with Video Tracking
#
# Tests the video block type, source_url, autoplay flag,
# and the on watch trigger with threshold operators.
# Includes companion story triggered by video engagement.

funnel "Video Course" {
    domain "course.example.com"

    route "Main" {
        order 1

        stage "Watch" {
            path "/watch"

            page "Video Page" {
                template "minimal_v1"

                block "intro_video" {
                    type video
                    source_url "https://cdn.example.com/intro.mp4"
                    autoplay false
                }

                block "cta_text" {
                    length short
                    prompt "Call to action below video"
                }

                form "VideoLead" {
                    field email required
                    field first_name
                }
            }

            on watch "intro_video" > 50 {
                do give_badge "engaged_viewer"
            }

            on watch "intro_video" >= 90 {
                do start_story "Video Follow Up"
            }

            on submit "VideoLead" {
                do give_badge "video_lead"
                do jump_to_stage "Offer"
            }
        }

        stage "Offer" {
            path "/offer"

            page "Offer Page" {
                block "offer_content" {
                    length medium
                    prompt "Special offer details"
                }
            }
        }
    }
}

story "Video Follow Up" {
    priority 1

    storyline "Engaged" {
        order 1

        enactment "Thank You" {
            scene "Thanks" {
                subject "Thanks for watching!"
                body "<p>We noticed you watched our video. Here is a special offer.</p>"
                from_email "hello@course.example.com"
                from_name "Course Team"
                reply_to "hello@course.example.com"
            }
        }
    }
}
