# Outbound webhook endpoint for a customer
resource "hookbase_webhook_endpoint" "acme_orders" {
  application_id  = hookbase_webhook_application.acme.id
  url             = "https://acme.com/webhooks/orders"
  description     = "Acme Corp order notifications"
  timeout_seconds = 15

  circuit_failure_threshold = 10
  circuit_cooldown_seconds  = 120
}
