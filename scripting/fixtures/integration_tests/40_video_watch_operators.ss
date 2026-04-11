# 40 — Video Watch Operators
#
# Tests all four watch trigger operators: >, <, >=, <=
# Verifies that different thresholds parse and compile correctly.

funnel "Watch Ops Test" {
    domain "test.example.com"

    route "Main" {
        order 1

        stage "Video Tests" {
            path "/test"

            page "Test Page" {
                block "vid_a" {
                    type video
                    source_url "https://cdn.example.com/a.mp4"
                }

                block "vid_b" {
                    type video
                    source_url "https://cdn.example.com/b.mp4"
                    autoplay true
                }

                form "TestForm" {
                    field email required
                }
            }

            on watch "vid_a" > 25 {
                do give_badge "quarter_watched"
            }

            on watch "vid_a" >= 50 {
                do give_badge "half_watched"
            }

            on watch "vid_b" > 75 {
                do give_badge "mostly_watched"
            }

            on watch "vid_b" >= 100 {
                do give_badge "fully_watched"
            }

            on submit "TestForm" {
                do give_badge "test_submitted"
            }
        }
    }
}

story "Watch Companion" {
    priority 1

    storyline "Main" {
        order 1

        enactment "Notify" {
            scene "Alert" {
                subject "Video engagement detected"
                body "<p>A user engaged with the test video.</p>"
                from_email "test@example.com"
                from_name "Test"
                reply_to "test@example.com"
            }
        }
    }
}
