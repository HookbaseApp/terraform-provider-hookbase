# HTTP destination with bearer auth
resource "hookbase_destination" "api" {
  name       = "internal-api"
  slug       = "internal-api"
  url        = "https://api.internal.example.com/webhooks"
  method     = "POST"
  auth_type  = "bearer"
  auth_config = {
    token = var.internal_api_token
  }
  timeout_ms = 15000
}

# Slack notification destination
resource "hookbase_destination" "slack" {
  name = "slack-alerts"
  slug = "slack-alerts"
  url  = var.slack_webhook_url
}

# HTTP destination whose receiver verifies each delivery. Hookbase generates the signing secret
# at create; hand it to the receiving service so it can check the webhook-signature header.
resource "hookbase_destination" "orders_service" {
  name = "orders-service"
  slug = "orders-service"
  url  = "https://orders.example.com/webhooks/hookbase"
}

module "orders_service" {
  source = "./modules/orders-service"

  hookbase_signing_secret = hookbase_destination.orders_service.signing_secret
}

output "orders_service_signing_secret" {
  value     = hookbase_destination.orders_service.signing_secret
  sensitive = true
}

# Bridge destination: publishes each delivery it receives as an outbound order.created event to
# the endpoints of one application. A bridge takes no url, headers or auth.
resource "hookbase_destination" "orders_bridge" {
  name = "orders-bridge"
  slug = "orders-bridge"
  type = "bridge"
  config = jsonencode({
    eventTypeSource = "fixed"
    eventType       = hookbase_event_type.order_created.name
    applicationId   = hookbase_webhook_application.acme.id
  })
}
