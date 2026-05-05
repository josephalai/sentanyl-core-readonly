// Demonstrates the top-level `campaign` block — one-off email send with
// AI-generated subject + body, badge-defined audience, and click→badge hook.
//
// Compiles into a pkgmodels.Campaign with Status=draft (no auto-send).

campaign "Summer Launch v2" {
    from_email "hello@acme.com"
    from_name "Acme Team"
    reply_to "support@acme.com"

    context_pack "brand-tone-v1"
    subject_gen "Tease the v2 launch and the headline benefit"
    body_gen "Lead with one customer outcome, list 3 v2 highlights, end with a single CTA"

    audience {
        must_have ["paid_subscriber"]
        must_not_have ["churned"]
    }

    on_click "https://acme.com/v2" {
        give_badge "v2_announce_click"
    }
}
