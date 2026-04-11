# 50 — Global Website with SEO and Navigation
#
# Tests the Site declaration (parallel to Funnel):
# - Site with domain, theme, SEO metadata, and navigation
# - Pages with blocks
# - Companion story for integration test validation

site "Joseph's Coaching Hub" {
    domain "coaching.example.com"
    theme "modern_light"

    seo {
        title "Manifestation Coaching"
        description "Unlock your potential with 1-on-1 coaching."
    }

    navigation {
        header { "Home" = "/", "About" = "/about", "Library" = "/library" }
        footer { "Terms" = "/terms", "Privacy" = "/privacy" }
    }

    page "Home" {
        block "hero" {
            length medium
            prompt "Write an inspiring hero section for a coaching site."
        }

        block "testimonials" {
            type text
            prompt "Write 3 testimonials from satisfied clients."
        }
    }

    page "About Us" {
        block "team" {
            type text
            prompt "Write an about page for a coaching team."
        }
    }
}

story "Coaching Welcome" {
    priority 1

    storyline "Onboard" {
        order 1

        enactment "Welcome" {
            scene "Welcome Email" {
                subject "Welcome to Joseph's Coaching"
                body "<p>Thank you for signing up.</p>"
                from_email "joseph@coaching.example.com"
                from_name "Joseph"
            }
            on sent {
                within 1d
                do next_scene
            }
        }
    }
}
