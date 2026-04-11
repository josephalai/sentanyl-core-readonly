# 22 — Badge Integration: story/storyline/trigger badge mechanics
#
# Combines all badge-related constructs: story required_badges,
# start_trigger, complete_trigger, on_begin/on_complete/on_fail
# badge transactions, storyline required_badges, and trigger-level
# required_badges with give_badge and remove_badge actions.

story "Academy" {
    priority 1
    start_trigger "academy_enrollment"
    complete_trigger "academy_graduation"

    required_badges {
        must_not_have "academy_graduated"
    }

    on_begin {
        give_badge "academy_enrolled"
    }
    on_complete {
        give_badge "academy_graduated"
        remove_badge "academy_enrolled"
    }
    on_fail {
        give_badge "academy_dropped"
        remove_badge "academy_enrolled"
    }

    storyline "Coursework" {
        order 1
        required_badges { must_have "academy_enrolled" }

        on_begin {
            give_badge "coursework_started"
        }
        on_complete {
            give_badge "coursework_passed"
            next_storyline "Final Exam"
        }
        on_fail {
            give_badge "coursework_failed"
        }

        enactment "Lesson" {
            level 1
            order 1

            scene "Lesson Email" {
                subject "Academy Lesson"
                body "<p>Complete the lesson: <a href='https://example.com/lesson'>Study</a></p>"
                from_email "academy@example.com"
                from_name "Academy"
                reply_to "academy@example.com"
            }

            on click "https://example.com/lesson" {
                trigger_priority 2
                required_badges { must_have "academy_enrolled" must_not_have "suspended" }
                do give_badge "lesson_complete"
                do mark_complete
            }

            on not_click "https://example.com/lesson" {
                trigger_priority 1
                within 5d
                do remove_badge "coursework_started"
                do mark_failed
            }
        }
    }

    storyline "Final Exam" {
        order 2
        required_badges { must_have "coursework_passed" }

        enactment "Exam" {
            level 1
            order 1
            scene "Exam Email" {
                subject "Final Exam"
                body "<p>Take the exam: <a href='https://example.com/exam'>Begin</a></p>"
                from_email "academy@example.com"
                from_name "Academy"
                reply_to "academy@example.com"
            }
            on click "https://example.com/exam" {
                do give_badge "exam_passed"
                do mark_complete
            }
        }
    }
}
