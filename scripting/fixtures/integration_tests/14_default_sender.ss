# 14 — Default Sender: default sender block inheritance
#
# Tests the default sender block. Stories using "use sender default"
# inherit from_email, from_name, and reply_to on all scenes,
# so individual scenes do not need to repeat sender info.

default sender {
    from_email "noreply@company.com"
    from_name "Company Updates"
    reply_to "support@company.com"
}

story "Sender Inheritance Demo" {
    use sender default
    priority 1

    storyline "Announcements" {
        order 1

        enactment "Announcement 1" {
            level 1
            order 1

            scene "Email 1" {
                subject "Big announcement incoming"
                body "<p>We have exciting news! <a href='https://example.com/news'>Read</a></p>"
            }

            on click "https://example.com/news" {
                do mark_complete
            }
        }

        enactment "Announcement 2" {
            level 2
            order 2

            scene "Email 2" {
                subject "Follow-up on our announcement"
                body "<p>Here are the details. <a href='https://example.com/details'>Details</a></p>"
            }

            on click "https://example.com/details" {
                do mark_complete
            }
        }
    }
}
