# adserver.local — Full Manifestation Lead Funnel + Email Campaign
#
# Opt-In Page → Thank You Page → 7-email nurture drip
# Lead magnet PDF auto-generated from Neville Goddard reference material.

default sender {
    from_email "joseph@adserver.local"
    from_name "Joseph | The Manifestation Lab"
    reply_to "joseph@adserver.local"
}

# ─────────────────────────────────────────────
#  FUNNEL
# ─────────────────────────────────────────────

funnel "Manifestation Lab" {
    domain "adserver.local"

    route "Main" {
        order 1

        # ── Stage 1: Opt-In ──────────────────────────
        stage "Opt-In" {
            path "/free-guide"

            page "Free Manifestation Worksheet" {
                template "minimal_v1"

                block "hero" {
                    section_id "hero"
                    length medium
                    prompt "Compelling headline and sub-headline for a free manifestation worksheet based on Neville Goddard's 'Feeling is the Secret'. Focus on the power of feeling your desired reality now."
                    context "https://archive.org/stream/feeling-is-the-secret-neville-goddard/feeling-is-the-secret-neville-goddard_djvu.txt"
                }

                block "benefits" {
                    section_id "benefits"
                    length medium
                    prompt "Three powerful benefits of downloading this manifestation worksheet. Each benefit should connect the reader's emotional state to manifesting their dream life."
                }

                block "testimonials" {
                    section_id "testimonials"
                    length short
                    prompt "Two short social proof testimonials from people who used manifestation techniques from Neville Goddard to transform their lives."
                }

                form "LeadCapture" {
                    field first_name
                    field email required
                }

                lead_magnet {
                    type "worksheet"
                    reference "https://archive.org/stream/feeling-is-the-secret-neville-goddard/feeling-is-the-secret-neville-goddard_djvu.txt"
                    context "Create a practical step-by-step manifestation worksheet based on Neville Goddard's core teachings. Include: 1) A section for writing your desire in present tense, 2) Feeling visualization prompts, 3) A daily practice checklist, 4) A gratitude as if column. Make it beautiful, structured, and immediately actionable."
                    theme "executive"
                }
            }

            on submit "LeadCapture" {
                do give_badge "manifestation_subscriber"
                do start_story "Manifestation Mastery Drip"
                do jump_to_stage "Thank You"
            }
        }

        # ── Stage 2: Thank You ───────────────────────
        stage "Thank You" {
            path "/thank-you"

            page "Thank You — Your Worksheet is on the Way" {
                template "minimal_v1"

                block "thank_you" {
                    section_id "hero"
                    length short
                    prompt "Warm, celebratory thank you message. Tell them their free Manifestation Worksheet is being sent to their inbox right now and to check their email. Keep energy high and affirming."
                }

                block "what_to_expect" {
                    section_id "benefits"
                    length medium
                    prompt "Tell them exactly what to expect over the next 7 days: they will receive a powerful email series walking them through Neville Goddard's core manifestation techniques — one per day. Build anticipation."
                }

                block "quick_tip" {
                    section_id "about"
                    length short
                    prompt "A single powerful Neville Goddard quote about feeling as the secret, with a brief 2-sentence reflection on why it changes everything."
                }
            }
        }
    }
}

# ─────────────────────────────────────────────
#  EMAIL STORY — 7-Day Manifestation Drip
# ─────────────────────────────────────────────

story "Manifestation Mastery Drip" {
    priority 1

    on_begin {
        give_badge "drip_enrolled"
    }

    on_complete {
        give_badge "drip_complete"
        remove_badge "drip_enrolled"
    }

    storyline "7-Day Core Training" {
        order 1

        enactment "Day 0 — Welcome + Worksheet" {
            level 1
            scene "Welcome" {
                subject "Your free Manifestation Worksheet is inside 🎯"
                body "<p>Hi {{first_name}},</p><p>Your <strong>Free Manifestation Worksheet</strong> is attached to this email — based directly on Neville Goddard's <em>Feeling is the Secret</em>.</p><p>Print it out, or work through it digitally. Either way, set aside 15 minutes today to complete it. What you write down has power.</p><p>Here's the one thing I want you to hold onto right now:</p><blockquote><em>\"The world is yourself pushed out. Ask yourself what assumptions you are making about yourself and the world, and you will find the answer to all your problems.\"</em><br>— Neville Goddard</blockquote><p>Tomorrow I'll share the single most important technique Neville ever taught. It takes under 5 minutes and most people feel results the first time they try it.</p><p>Talk soon,<br>Joseph</p>"
                from_name "Joseph | The Manifestation Lab"
                reply_to "joseph@adserver.local"
            }
            on sent {
                within 1d
                do next_scene
            }
        }

        enactment "Day 1 — The Feeling Technique" {
            level 2
            scene "Feeling is the Secret" {
                subject "The 5-minute technique that changes everything"
                body "<p>Hi {{first_name}},</p><p>Neville's core insight is radically simple:</p><p><strong>Feeling is the creative force. Not wanting. Not hoping. Feeling as if it's already done.</strong></p><p>Most people manifest backwards — they think about what they want, feel the lack of it, and wonder why nothing changes. Neville flips this entirely.</p><p><strong>Try this right now (5 minutes):</strong></p><ol><li>Close your eyes.</li><li>Pick one desire — one thing you want most.</li><li>Ask: <em>How would I feel if this were already true?</em></li><li>Let that feeling arise. Don't force it. Just invite it.</li><li>Stay in that feeling for 60 seconds. Breathe it in.</li></ol><p>That's it. That's the entire method.</p><p>The worksheet I sent you yesterday has a dedicated section for this — use it daily.</p><p>Tomorrow: the revision technique. It's how you rewrite your past to change your future.</p><p>Joseph</p>"
            }
            on sent {
                within 1d
                do next_scene
            }
        }

        enactment "Day 2 — Revision" {
            level 3
            scene "Rewrite Your Past" {
                subject "You can literally rewrite what happened to you"
                body "<p>Hi {{first_name}},</p><p>This one surprises people.</p><p>Neville taught that you can <strong>revise</strong> events that already happened — and that doing so changes their impact on your future.</p><p>He called it the Revision Technique:</p><p><strong>Tonight, before bed:</strong></p><ol><li>Review your day.</li><li>Find one moment that didn't go how you wanted.</li><li>In your mind, replay it — but this time, the way you WISH it had gone.</li><li>Feel the satisfaction of that version. Hold it for 30 seconds.</li><li>Let yourself drift to sleep in that feeling.</li></ol><p>Your subconscious doesn't distinguish between what happened and what you vividly imagine. Revision plants new seeds.</p><p>Do this every night for a week and notice what shifts.</p><p>See you tomorrow — we're going into SATS (State Akin to Sleep). It's the most powerful time to reprogram your reality.</p><p>Joseph</p>"
            }
            on sent {
                within 2d
                do next_scene
            }
        }

        enactment "Day 4 — SATS" {
            level 4
            scene "State Akin to Sleep" {
                subject "The hypnagogic secret Neville used every night"
                body "<p>Hi {{first_name}},</p><p>There's a brief window each night — right as you're falling asleep — where your mind is uniquely receptive to new beliefs.</p><p>Neville called it <strong>SATS: State Akin to Sleep</strong>.</p><p>In this drowsy, hypnagogic state, your critical factor (the part of your mind that argues with new ideas) goes quiet. What you impress on your mind in this state goes directly into your subconscious.</p><p><strong>The SATS method:</strong></p><ol><li>As you lie in bed tonight, let your body relax completely.</li><li>When you feel that heavy, drifting feeling — you're in SATS.</li><li>Construct a short scene that implies your wish is fulfilled. Maybe a friend congratulating you. A text that confirms good news. One clear, joyful image.</li><li>Loop that scene slowly. Let it feel natural. Don't force it.</li><li>Fall asleep in the feeling of that scene.</li></ol><p>Do this for 7 nights in a row and track what happens in your waking life.</p><p>Tomorrow I'll share a real story about how this worked for someone who thought manifestation was nonsense.</p><p>Joseph</p>"
            }
            on sent {
                within 2d
                do next_scene
            }
        }

        enactment "Day 6 — Story + Social Proof" {
            level 5
            scene "It Works When You Work It" {
                subject "She manifested her dream job in 11 days"
                body "<p>Hi {{first_name}},</p><p>I want to share a story.</p><p>A woman named Sarah had been job-hunting for 8 months. Nothing was landing. She was exhausted, starting to doubt herself.</p><p>She found Neville's work and decided to run a 10-day experiment. Every night, she would fall asleep imagining a specific scene: her best friend calling her, saying <em>\"Oh my God, you got the job! I'm so proud of you!\"</em></p><p>On day 11, she got an unexpected call from a company she'd applied to months ago — one she'd completely forgotten about.</p><p>She started the following Monday.</p><p>I'm not telling you this is magic. I'm telling you this is psychology meeting intention. When you align your subconscious expectations with your desires, your actions, perception, and opportunities all shift.</p><p>You've been doing the work this week. Stay consistent.</p><p>Tomorrow's email is the final one in this series — and I want to share something I've put together that takes everything we've covered and gives you a complete system.</p><p>Joseph</p>"
            }
            on sent {
                within 1d
                do next_scene
            }
        }

        enactment "Day 7 — Final + Offer" {
            level 6
            scene "Complete System" {
                subject "Day 7: the complete picture (and what's next)"
                body "<p>Hi {{first_name}},</p><p>You made it to Day 7. That matters.</p><p>Here's the complete Neville Goddard system in one place:</p><ul><li><strong>The Law:</strong> Consciousness is the only reality. What you are conscious of, you experience.</li><li><strong>The Technique:</strong> Feel the wish fulfilled — not someday, now.</li><li><strong>The Practice:</strong> SATS nightly. Revision of unwanted events. Living from the end.</li><li><strong>The Trust:</strong> Let go. Don't lust after results. Plant the seed, water it with feeling, and trust the harvest.</li></ul><p>If this week has resonated with you and you want to go deeper — I've built a complete 30-day Manifestation Practice that walks through every one of Neville's techniques with daily exercises, audio meditations, and a live community.</p><p>You can check it out here: <a href=\"http://adserver.local/upgrade\">The 30-Day Manifestation Practice →</a></p><p>Whether or not that's for you, keep using your worksheet. Keep doing SATS. The work is simple. The consistency is the discipline.</p><p>Thank you for trusting me with your time this week.</p><p>Joseph</p>"
            }
        }
    }
}
