# 10 — Persistent Links: persist_scope on triggers
#
# Tests persist_scope values: "enactment", "storyline", "story", "forever".
# Each trigger demonstrates a different scope, controlling how long
# the click tracking remains active.

default sender {
    from_email "persist@example.com"
    from_name "Persist Test"
    reply_to "persist@example.com"
}

story "Persistence Demo" {
    use sender default
    priority 1

    storyline "Scoped Triggers" {
        order 1

        enactment "Enactment Scope" {
            level 1
            order 1
            scene "Email" {
                subject "Enactment-scoped link"
                body "<a href='https://example.com/e-scope'>Click</a>"
            }
            on click "https://example.com/e-scope" {
                trigger_priority 4
                persist_scope "enactment"
                do mark_complete
            }
        }

        enactment "Storyline Scope" {
            level 2
            order 2
            scene "Email" {
                subject "Storyline-scoped link"
                body "<a href='https://example.com/sl-scope'>Click</a>"
            }
            on click "https://example.com/sl-scope" {
                trigger_priority 3
                persist_scope "storyline"
                do mark_complete
            }
        }

        enactment "Story Scope" {
            level 3
            order 3
            scene "Email" {
                subject "Story-scoped link"
                body "<a href='https://example.com/st-scope'>Click</a>"
            }
            on click "https://example.com/st-scope" {
                trigger_priority 2
                persist_scope "story"
                do mark_complete
            }
        }

        enactment "Forever Scope" {
            level 4
            order 4
            scene "Email" {
                subject "Forever-scoped link"
                body "<a href='https://example.com/forever'>Click</a>"
            }
            on click "https://example.com/forever" {
                trigger_priority 1
                persist_scope "forever"
                do mark_complete
            }
        }
    }
}
