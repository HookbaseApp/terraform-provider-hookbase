# Route Stripe events to internal API with transform and filter
resource "hookbase_route" "stripe_to_api" {
  name           = "stripe-to-internal-api"
  source_id      = hookbase_source.stripe.id
  destination_id = hookbase_destination.api.id
  transform_id   = hookbase_transform.stripe_normalize.id
  filter_id      = hookbase_filter.high_value.id
  priority       = 1

  notify_on_failure = true
  notify_emails     = "oncall@example.com"
  failure_threshold = 5
}

# Simple pass-through route
resource "hookbase_route" "github_to_slack" {
  name           = "github-to-slack"
  source_id      = hookbase_source.github.id
  destination_id = hookbase_destination.slack.id
}
