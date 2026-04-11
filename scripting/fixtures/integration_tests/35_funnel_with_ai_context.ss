# 35 — Funnel with AI Context
#
# Tests AI context directives at funnel, page, and block levels.
# Verifies global context declaration and extend references.
# Includes companion story for integration test.

funnel "AI Context Funnel" {
    domain "ai.example.com"
    ai context global "https://example.com/transcript.txt" "main_transcript"

    route "Main" {
        order 1

        stage "Landing" {
            path "/landing"

            page "AI Landing Page" {
                template "minimal_v1"
                ai context extend "main_transcript"

                block "hero_headline" {
                    length short
                    ai context extend "main_transcript"
                    prompt "High-converting curiosity headline"
                }

                block "body_copy" {
                    length medium
                    prompt "Compelling body copy with benefits"
                }

                form "Subscribe" {
                    field email required
                }
            }

            on submit "Subscribe" {
                do give_badge "subscriber"
            }
        }
    }
}

story "AI Companion" {
    priority 1

    storyline "Follow Up" {
        order 1

        enactment "AI Follow Up" {
            scene "Follow Up" {
                subject "Thanks for subscribing"
                body "<p>Here is your AI-curated content.</p>"
                from_email "ai@ai.example.com"
                from_name "AI Team"
            }
            on sent {
                within 1d
                do next_scene
            }
        }
    }
}
