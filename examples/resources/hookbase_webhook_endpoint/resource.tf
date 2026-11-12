# Outbound webhook endpoint for a customer
resource "hookbase_webhook_endpoint" "acme_orders" {
  application_id  = hookbase_webhook_application.acme.id
  url             = "https://acme.com/webhooks/orders"
  description     = "Acme Corp order notifications"
  timeout_seconds = 15

  circuit_failure_threshold = 10
  circuit_cooldown_seconds  = 120

  # Hold messages while the circuit is open and send them once the endpoint recovers. Left unset,
  # hold_on_circuit_open keeps the endpoint's current value (true for a new endpoint).
  hold_on_circuit_open = true
}
