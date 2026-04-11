# 24 — Multi-Scene Drip: multi-scene enactment (drip sequence)
#
# Tests a single enactment with multiple scenes that form a
# drip sequence. Each scene is sent in order, with triggers
# controlling progression between scenes.

story "Drip Campaign" {
    priority 1

    storyline "Education Drip" {
        order 1

        enactment "Three-Part Series" {
            level 1
            order 1

            scene "Part 1" {
                subject "Part 1: Getting Started"
                body "<h1>Welcome</h1><p>Let us start with the basics. <a href='https://example.com/part1'>Read</a></p>"
                from_email "edu@example.com"
                from_name "Education Team"
                reply_to "edu@example.com"
            }

            scene "Part 2" {
                subject "Part 2: Going Deeper"
                body "<h1>Intermediate</h1><p>Now let us explore more. <a href='https://example.com/part2'>Continue</a></p>"
                from_email "edu@example.com"
                from_name "Education Team"
                reply_to "edu@example.com"
            }

            scene "Part 3" {
                subject "Part 3: Mastery"
                body "<h1>Advanced</h1><p>You are almost there! <a href='https://example.com/part3'>Finish</a></p>"
                from_email "edu@example.com"
                from_name "Education Team"
                reply_to "edu@example.com"
            }

            on open {
                trigger_priority 3
                do next_scene
            }

            on click "https://example.com/part3" {
                trigger_priority 2
                do give_badge "series_complete"
                do mark_complete
            }

            on not_open {
                within 2d
                trigger_priority 1
                do retry_scene up_to 1
                    else do next_scene
            }
        }
    }
}
