// Simple one-storyline campaign
// Demonstrates minimal viable script structure

story "Simple Welcome" {
    priority 1

    storyline "Main" {
        order 1

        enactment "Welcome" {
            level 1
            order 1

            scene "Welcome Email" {
                subject "Welcome to our platform!"
                body "<h1>Welcome!</h1><p>We are glad you joined.</p>"
                from_email "hello@example.com"
                from_name "The Team"
                reply_to "support@example.com"
            }
        }
    }
}
