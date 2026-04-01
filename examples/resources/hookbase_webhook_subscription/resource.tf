# Subscribe endpoint to order events
resource "hookbase_webhook_subscription" "acme_order_created" {
  endpoint_id   = hookbase_webhook_endpoint.acme_orders.id
  event_type_id = hookbase_event_type.order_created.id
}

# Subscription with filter and priority
resource "hookbase_webhook_subscription" "acme_order_failed" {
  endpoint_id       = hookbase_webhook_endpoint.acme_orders.id
  event_type_id     = hookbase_event_type.order_failed.id
  filter_expression = "event.data.amount > 100"
  priority_override = 0  # Critical
}
