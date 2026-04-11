# 16 — Pattern Reuse: pattern definitions reused across storylines
#
# Tests the pattern construct. A pattern defines a reusable
# enactment template with parameterized subjects and links.
# Two storylines each invoke the same pattern with different arguments.

links {
    product_a = "https://example.com/product-a"
    product_b = "https://example.com/product-b"
}

policy buy_click(link) {
    on click link {
        trigger_priority 1
        mark_complete true
        within 3d
    }
}

pattern product_pitch(name, pitch_subject, pitch_body, buy_link) {
    enactment name {
        scenes 1..3 as n {
            scene "Pitch ${n}" {
                subject "${pitch_subject} (${n}/3)"
                body "<h1>${pitch_body}</h1><p>Email ${n} of 3</p>"
            }
        }
        use policy buy_click(buy_link)
    }
}

story "Product Launch" {
    priority 1

    storyline "Product A" {
        order 1
        use pattern product_pitch("Product A Pitch", "Introducing Product A", "Product A is here", product_a)
    }

    storyline "Product B" {
        order 2
        use pattern product_pitch("Product B Pitch", "Discover Product B", "Product B awaits", product_b)
    }
}
