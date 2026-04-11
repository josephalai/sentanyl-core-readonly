# Combo: E-Commerce Funnel
#
# Full e-commerce funnel combining: story chaining (next_story),
# badge gating, conditional routing, retries, click branching,
# and lifecycle hooks. Flow: Welcome -> Browse -> Cart Abandon
# -> Purchase -> Post-Purchase Follow-Up.

default sender {
    from_email "shop@example.com"
    from_name "ShopCo"
    reply_to "support@example.com"
}

story "Welcome to ShopCo" {
    use sender default
    priority 1

    on_begin {
        give_badge "shopper_enrolled"
    }
    on_complete {
        give_badge "welcome_done"
        next_story "Browse and Buy"
    }
    on_fail {
        give_badge "welcome_failed"
        next_story "Browse and Buy"
    }

    storyline "Welcome" {
        order 1

        enactment "Welcome Email" {
            level 1
            order 1
            scene "Welcome" {
                subject "Welcome to ShopCo!"
                body "<h1>Welcome!</h1><p>Start shopping: <a href='https://shop.example.com/browse'>Browse</a></p>"
            }
            on click "https://shop.example.com/browse" {
                trigger_priority 1
                do give_badge "browsed"
                do mark_complete
            }
            on not_click "https://shop.example.com/browse" {
                within 3d
                do retry_scene up_to 1
                    else do mark_failed
            }
        }
    }
}

story "Browse and Buy" {
    use sender default
    priority 2

    on_complete {
        give_badge "purchased"
        next_story "Post-Purchase"
    }

    storyline "Product Discovery" {
        order 1

        on_complete {
            conditional_route {
                required_badges { must_have "cart_added" }
                next_storyline "Cart Recovery"
                priority 10
            }
            conditional_route {
                required_badges { must_not_have "cart_added" }
                next_storyline "Direct Buy"
                priority 1
            }
        }

        enactment "Featured Products" {
            level 1
            order 1
            scene "Products Email" {
                subject "Top picks for you"
                body "<p>Check these out: <a href='https://shop.example.com/featured'>See Products</a></p>"
            }
            on click "https://shop.example.com/featured" {
                trigger_priority 2
                do give_badge "cart_added"
                do mark_complete
            }
            on not_click "https://shop.example.com/featured" {
                within 2d
                do mark_complete
            }
        }
    }

    storyline "Cart Recovery" {
        order 2
        required_badges { must_have "cart_added" }

        enactment "Cart Reminder" {
            level 1
            order 1
            scene "Cart Email" {
                subject "You left something in your cart"
                body "<p>Complete your purchase: <a href='https://shop.example.com/cart'>View Cart</a></p>"
            }
            on click "https://shop.example.com/cart" {
                do mark_complete
            }
            on not_click "https://shop.example.com/cart" {
                within 1d
                do retry_scene up_to 2 times
                    else do mark_failed
            }
        }
    }

    storyline "Direct Buy" {
        order 3

        enactment "Buy Prompt" {
            level 1
            order 1
            scene "Buy Email" {
                subject "Ready to buy?"
                body "<p>Complete your order: <a href='https://shop.example.com/buy'>Buy Now</a></p>"
            }
            on click "https://shop.example.com/buy" {
                do mark_complete
            }
            on not_click "https://shop.example.com/buy" {
                within 3d
                do mark_failed
            }
        }
    }
}

story "Post-Purchase" {
    use sender default
    priority 3

    storyline "Follow-Up" {
        order 1

        enactment "Thank You" {
            level 1
            order 1
            scene "Thanks Email" {
                subject "Thanks for your purchase!"
                body "<p>We hope you love it. <a href='https://shop.example.com/review'>Leave a Review</a></p>"
            }
            on click "https://shop.example.com/review" {
                do give_badge "reviewer"
                do mark_complete
            }
            on not_click "https://shop.example.com/review" {
                within 7d
                do mark_complete
            }
        }
    }
}
