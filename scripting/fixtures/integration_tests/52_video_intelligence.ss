# 52 — Video Intelligence Full Pipeline
#
# Tests first-class media entities with badge rules, chapters,
# interactions, player presets, channels, and video triggers.
# Combines media + funnel + story for badge-driven video intelligence.

# -- Player Preset --

player_preset "Brand Player" {
    player_color "#3b82f6"
    show_controls true
    show_big_play_button true
    allow_fullscreen true
    allow_playback_rate true
    end_behavior "stop"
}

# -- Media Entities --

media "Intro Video" {
    title "Welcome to Our Course"
    description "An introduction to the course material"
    source_url "https://storage.googleapis.com/sendhero-videos/VjdBaIC57zk-ufo.webm"
    poster_url "https://storage.googleapis.com/sendhero-videos/poster.jpg"
    player_preset "Brand Player"
    tags "intro" "welcome"

    chapter "Welcome" {
        start_sec 0
        end_sec 30
    }

    chapter "Overview" {
        start_sec 30
        end_sec 120
    }

    turnstile {
        start_sec 15
        required
        field email
        field first_name
    }

    cta {
        start_sec 60
        text "Get the Full Course"
        url "https://example.com/offer"
        button_text "Enroll Now"
    }

    annotation {
        start_sec 45
        text "Important concept"
    }

    badge_rule {
        event progress
        operator ">="
        threshold 25
        badge "video_started"
    }

    badge_rule {
        event progress
        operator ">="
        threshold 75
        badge "video_engaged"
    }

    badge_rule {
        event complete
        badge "video_completed"
    }
}

media "Advanced Module" {
    title "Advanced Techniques"
    source_url "https://storage.googleapis.com/sendhero-videos/advanced.mp4"

    badge_rule {
        event complete
        badge "advanced_complete"
    }
}

# -- Channel --

channel "Course Playlist" {
    title "Complete Course"
    description "All course videos in order"
    layout "playlist_right"
    items "Intro Video" "Advanced Module"
}

# -- Media Webhook --

media_webhook "Analytics Hook" {
    url "https://hooks.example.com/video-events"
    event_types "play" "complete" "turnstile_submit"
    enabled true
}

# -- Funnel with Media Integration --

funnel "Video Sales Funnel" {
    domain "sales.example.com"

    route "Main" {
        order 1

        stage "Watch" {
            path "/watch"

            page "Video Page" {
                template "minimal_v1"

                block "hero_video" {
                    type video
                    media_ref "Intro Video"
                    player_preset "Brand Player"
                }

                block "below_video" {
                    length medium
                    prompt "Benefits and social proof below the video"
                }

                form "VideoLead" {
                    field email required
                    field first_name
                }
            }

            on watch "hero_video" > 50 {
                do give_badge "engaged_viewer"
            }

            on progress "hero_video" >= 90 {
                do give_badge "almost_done"
            }

            on complete "hero_video" {
                do start_story "Video Follow Up"
                do give_badge "watched_full"
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
                    prompt "Premium course offer details"
                }

                form "Checkout" {
                    type checkout
                    field email required
                    field card required
                }
            }

            on purchase "Checkout" {
                do give_badge "customer"
            }
        }
    }
}

# -- Companion Story --

story "Video Follow Up" {
    priority 1

    storyline "Engaged" {
        order 1

        enactment "Thank You" {
            scene "Thanks" {
                subject "Thanks for watching our intro!"
                body "<p>We noticed you completed the intro video. Here is your exclusive offer.</p>"
                from_email "hello@sales.example.com"
                from_name "Course Team"
                reply_to "hello@sales.example.com"
            }
        }
    }
}

# -- Product with Media References --

product "Video Course" {
    description "Complete video course with media integration"
    type "course"

    module "Getting Started" {
        lesson "Introduction" {
            media_ref "Intro Video"
            content "<p>Watch the introduction video above.</p>"
        }

        lesson "Advanced Topics" {
            media_ref "Advanced Module"
            content "<p>Advanced techniques covered in this module.</p>"
        }
    }
}
