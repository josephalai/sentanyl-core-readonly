package scripting

// FixtureMultiStorySequence — Full multi-story campaign converted from
// scripts/multi-story-sequence.sh. Demonstrates:
// - 2 stories with different priorities
// - 5 storylines total (3 workshop tracks + 2 coaching tracks)
// - 14 enactments total (4 per workshop track + 1 per coaching track)
// - 40 scenes total (36 workshop + 4 coaching)
// - Data blocks with for-loop generation
// - Dot-access click/not_click triggers
// - Badge awards on purchase
// - Minute-based timing for real-time testing
const FixtureMultiStorySequence = `
# Multi-Story Sequence — Full Campaign Demo
#
# Story A: "Buy Manifesting Workshops" (priority 1)
#   3 Storylines (Wealth, Love, Health), each with 4 Enactments:
#     A_Soft_Intrigue  -> "Get more info" (3 scenes)
#     B_Hard_Intrigue  -> "Are you sure?" (3 scenes)
#     C_Soft_Sell      -> "Buy now" (3 scenes)
#     D_Hard_Sell      -> "Final chance" (3 scenes)
#
# Story B: "Buy Coaching Products" (priority 2)
#   2 Storylines (Executive, Leadership), each with 1 Enactment:
#     Product_Info_and_Buy -> Info + Buy (2 scenes)
#
# All timing uses minutes so you can watch it work in real time.

default sender {
  from_email "coach@manifesting.com"
  from_name  "Manifesting Coach"
  reply_to   "support@manifesting.com"
}

links {
  # Manifesting Workshop Links
  w1_info = "https://example.com/wealth/info"
  w1_buy  = "https://example.com/wealth/buy"
  w2_info = "https://example.com/love/info"
  w2_buy  = "https://example.com/love/buy"
  w3_info = "https://example.com/health/info"
  w3_buy  = "https://example.com/health/buy"

  # Coaching Product Links
  coach_a_info = "https://example.com/coach/executive/info"
  coach_a_buy  = "https://example.com/coach/executive/buy"
  coach_b_info = "https://example.com/coach/leadership/info"
  coach_b_buy  = "https://example.com/coach/leadership/buy"
}

data workshops = [
  { name: "Manifesting Wealth",  order: 1, info: w1_info, buy: w1_buy },
  { name: "Manifesting Love",    order: 2, info: w2_info, buy: w2_buy },
  { name: "Manifesting Health",  order: 3, info: w3_info, buy: w3_buy }
]

data coaching_products = [
  { name: "Executive Coaching",  order: 1, info: coach_a_info, buy: coach_a_buy },
  { name: "Leadership Coaching", order: 2, info: coach_b_info, buy: coach_b_buy }
]

# Story A: Buy Manifesting Workshops
story "Buy Manifesting Workshops" {
  use sender default
  priority 1

  for ws in workshops {
    storyline "${ws.name} Track" {
      order ws.order

      # Enactment A: Soft Intrigue - "get more info" emails
      # Click info link -> jump to C_Soft_Sell
      # No click after 3m -> jump to B_Hard_Intrigue
      enactment "A_Soft_Intrigue" {
        scenes 1..3 as n {
          scene "Intrigue Email ${n}" {
            subject "Curious about ${ws.name}? (Email ${n}/3)"
            body "<p>Get more info about ${ws.name} here: <a href='${ws.info}'>Learn More</a></p>"
          }
        }

        on click ws.info {
          trigger_priority 1
          mark_complete true
          within 3m
          do jump_to_enactment "C_Soft_Sell"
        }

        on not_click ws.info {
          within 3m
          do jump_to_enactment "B_Hard_Intrigue"
        }
      }

      # Enactment B: Hard Intrigue - "Are you sure?"
      # Click info -> jump to C_Soft_Sell
      # No click -> advance to next storyline (failed)
      enactment "B_Hard_Intrigue" {
        scenes 1..3 as n {
          scene "Hard Intrigue Email ${n}" {
            subject "Don't miss out on ${ws.name}! (Email ${n}/3)"
            body "<p>Are you sure you want to miss ${ws.name}? <a href='${ws.info}'>Discover More</a></p>"
          }
        }

        on click ws.info {
          trigger_priority 1
          mark_complete true
          within 3m
          do jump_to_enactment "C_Soft_Sell"
        }

        on not_click ws.info {
          within 3m
          do advance_to_next_storyline
        }
      }

      # Enactment C: Soft Sell - "Buy now" emails
      # Click buy -> badge + advance to next storyline
      # No click -> jump to D_Hard_Sell
      enactment "C_Soft_Sell" {
        scenes 1..3 as n {
          scene "Soft Sell Email ${n}" {
            subject "Ready to start ${ws.name}? (Email ${n}/3)"
            body "<p>Start your journey with ${ws.name}. Limited spots! <a href='${ws.buy}'>Buy Now</a></p>"
          }
        }

        on click ws.buy {
          trigger_priority 1
          mark_complete true
          within 3m
          do give_badge "${ws.name}_purchased"
          do advance_to_next_storyline
        }

        on not_click ws.buy {
          within 3m
          do jump_to_enactment "D_Hard_Sell"
        }
      }

      # Enactment D: Hard Sell - "Final chance" emails
      # Click buy -> badge + advance to next storyline
      # No click -> advance to next storyline (move on)
      enactment "D_Hard_Sell" {
        scenes 1..3 as n {
          scene "Hard Sell Email ${n}" {
            subject "FINAL CHANCE: ${ws.name}! (Email ${n}/3)"
            body "<p>Last chance to join ${ws.name}. <a href='${ws.buy}'>Buy Now</a></p>"
          }
        }

        on click ws.buy {
          trigger_priority 1
          mark_complete true
          within 3m
          do give_badge "${ws.name}_purchased"
          do advance_to_next_storyline
        }

        on not_click ws.buy {
          within 3m
          do advance_to_next_storyline
        }
      }
    }
  }
}

# Story B: Buy Coaching Products
story "Buy Coaching Products" {
  use sender default
  priority 2

  for product in coaching_products {
    storyline "${product.name} Track" {
      order product.order

      enactment "Product_Info_and_Buy" {
        scene "Info Email" {
          subject "Learn about our ${product.name} Program"
          body "<p>Discover how ${product.name} can transform your life. <a href='${product.info}'>Get Details</a></p>"
        }
        scene "Buy Email" {
          subject "Enroll in ${product.name} Today!"
          body "<p>Ready for the next step? <a href='${product.buy}'>Enroll Now</a></p>"
        }

        on click product.buy {
          mark_complete true
          within 7m
          do give_badge "${product.name}_purchased"
          do advance_to_next_storyline
        }

        on not_click product.buy {
          within 7m
          do mark_failed
          do advance_to_next_storyline
        }
      }
    }
  }
}
`
