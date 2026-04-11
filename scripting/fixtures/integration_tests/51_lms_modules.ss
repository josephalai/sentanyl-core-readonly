# 51 — LMS Structure with Modules and Lessons
#
# Tests the Product expansion with Module/Lesson structure:
# - Product with nested modules and lessons
# - Lessons with video_url, content, and draft status
# - Companion story for integration test validation

product "Zero Point Gravity Masterclass" {
    type "course"
    description "A 6-module deep-dive into zero-point energy."

    module "Module 1: Introduction" {
        lesson "Welcome & Overview" {
            video_url "https://cdn.example.com/videos/zpg-intro.mp4"
            content "<p>Welcome to the Zero Point Gravity Masterclass.</p>"
        }

        lesson "What is Zero Point Energy?" {
            video_url "https://cdn.example.com/videos/zpg-what.mp4"
            content "<p>An introduction to zero point energy concepts.</p>"
        }
    }

    module "Module 2: Advanced Concepts" {
        lesson "Quantum Field Theory Basics" {
            video_url "https://cdn.example.com/videos/zpg-qft.mp4"
            content "<p>Understanding quantum fields.</p>"
        }

        lesson "Practical Exercises" {
            video_url "https://cdn.example.com/videos/zpg-exercises.mp4"
            content "<p>Hands-on exercises for mastery.</p>"
            draft true
        }
    }
}

offer "ZPG Full Access" {
    pricing_model one_time
    price 97.00
    currency "usd"

    includes_product "Zero Point Gravity Masterclass"
    grants_badge "zpg_student"
}

funnel "ZPG Sales Funnel" {
    domain "zpg.example.com"

    route "Main" {
        order 1

        stage "Sales" {
            path "/"

            page "Sales Page" {
                block "vsl" {
                    type video
                    source_url "https://cdn.example.com/videos/zpg-vsl.mp4"
                    autoplay true
                }

                form "Buy" {
                    type checkout
                    offer "ZPG Full Access"
                    field email required
                    field card required
                }
            }

            on purchase "Buy" {
                do give_badge "zpg_student"
                do jump_to_stage "ThankYou"
            }
        }

        stage "ThankYou" {
            path "/thank-you"

            page "Thank You" {
                block "confirmation" {
                    length short
                    prompt "Thank you for purchasing the ZPG Masterclass."
                }
            }
        }
    }
}

story "ZPG Onboarding" {
    priority 1

    storyline "Welcome" {
        order 1

        enactment "WelcomeEmail" {
            scene "Welcome" {
                subject "Welcome to Zero Point Gravity"
                body "<p>You now have access to your course.</p>"
                from_email "team@zpg.example.com"
                from_name "ZPG Team"
            }
            on sent {
                within 1d
                do next_scene
            }
        }
    }
}
