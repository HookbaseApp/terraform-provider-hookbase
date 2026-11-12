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

# Route that dead-letters deliveries while the destination's circuit is open, instead of holding them
# until it recovers. Left unset, hold_on_circuit_open keeps the route's current value (true for a new
# route).
resource "hookbase_route" "audit_log" {
  name                 = "audit-log"
  source_id            = hookbase_source.github.id
  destination_id       = hookbase_destination.api.id
  hold_on_circuit_open = false

  circuit_failure_threshold = 10
  circuit_cooldown_seconds  = 300
}
