# 26 — Dot-Access (var.field) in For Loops with Triggers
#
# Tests dot-access syntax for data fields in for loops.
# Link fields from data objects are used directly in click
# triggers (on click item.link) and in string interpolation.

default sender {
    from_email "dotaccess@example.com"
    from_name "Dot Access Test"
    reply_to "dotaccess@example.com"
}

links {
    guide_a = "https://example.com/guide-a"
    guide_b = "https://example.com/guide-b"
}

data guides = [
    { name: "Beginner Guide", order: 1, link: guide_a },
    { name: "Advanced Guide", order: 2, link: guide_b }
]

story "Guide Series" {
    use sender default
    priority 1

    for g in guides {
        storyline "${g.name} Track" {
            order g.order

            enactment "${g.name}" {
                scene "${g.name} Email" {
                    subject "Read the ${g.name}"
                    body "<p>Start reading: <a href='${g.link}'>Open Guide</a></p>"
                }

                on click g.link {
                    trigger_priority 1
                    mark_complete true
                    within 3d
                    do give_badge "${g.name}_reader"
                }

                on not_click g.link {
                    within 5d
                    do advance_to_next_storyline
                }
            }
        }
    }
}
