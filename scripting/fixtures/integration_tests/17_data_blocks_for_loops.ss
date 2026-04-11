# 17 — Data Blocks + For Loop Generation
#
# Tests data blocks with structured objects and for loops that
# generate storylines dynamically. Each data entry produces
# a storyline with customized content.

default sender {
    from_email "courses@example.com"
    from_name "Course Platform"
    reply_to "courses@example.com"
}

links {
    math_link = "https://example.com/math"
    science_link = "https://example.com/science"
    history_link = "https://example.com/history"
}

data courses = [
    { name: "Math Fundamentals", order: 1, link: math_link },
    { name: "Science Basics", order: 2, link: science_link },
    { name: "History 101", order: 3, link: history_link }
]

story "Course Catalog" {
    use sender default
    priority 1

    for course in courses {
        storyline "${course.name} Track" {
            order course.order

            enactment "${course.name} Lesson" {
                scene "${course.name} Email" {
                    subject "Start learning ${course.name}"
                    body "<p>Begin your journey with ${course.name}. <a href='${course.link}'>Enroll</a></p>"
                }

                on click course.link {
                    trigger_priority 1
                    do give_badge "${course.name}_enrolled"
                    do mark_complete
                }

                on not_click course.link {
                    within 5d
                    do mark_failed
                }
            }
        }
    }
}
