# 48 — Quiz Assessment
#
# Tests quiz declaration with questions, answers, score tracking,
# and badge assignment based on score thresholds.
# Includes companion story for integration test validation.

quiz "Consciousness Assessment" {
    question "What limits you most?" {
        answer "Mainstream science limits" add_score 1
        answer "My own perception" add_score 5
        answer "Nothing, I am limitless" add_score 10
    }
    question "Have you experienced zero point gravity?" {
        answer "Yes" add_score 10
        answer "No" add_score 0
        answer "Unsure" add_score 3
    }
    question "How often do you meditate?" {
        answer "Daily" add_score 8
        answer "Weekly" add_score 4
        answer "Never" add_score 0
    }

    on complete {
        if score > 15 {
            do give_badge "advanced_practitioner"
        }
        if score > 5 {
            do give_badge "intermediate"
        }
        else {
            do give_badge "novice"
        }
    }
}

funnel "Quiz Funnel" {
    domain "quiz.example.com"

    route "Main" {
        order 1

        stage "Quiz Page" {
            path "/"

            page "Assessment" {
                block "intro" {
                    length short
                    prompt "Introduction to the consciousness quiz"
                }
            }
        }
    }
}

story "Quiz Follow-up" {
    priority 1

    storyline "Follow-up" {
        order 1

        enactment "Follow-up Email" {
            scene "Results" {
                subject "Your Assessment Results"
                body "<p>Based on your answers, here is your path.</p>"
                from_email "quiz@quiz.example.com"
                from_name "Quiz Team"
            }
            on sent {
                within 1d
                do next_scene
            }
        }
    }
}
