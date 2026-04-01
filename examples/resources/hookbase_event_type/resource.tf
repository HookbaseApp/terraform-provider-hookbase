# Define event types for outbound webhooks
resource "hookbase_event_type" "order_created" {
  name             = "order.created"
  display_name     = "Order Created"
  description      = "Fired when a new order is placed"
  category         = "orders"
  default_priority = 2  # Normal
  schema           = file("${path.module}/schemas/order-created.json")
  example_payload  = file("${path.module}/schemas/order-created-example.json")
}

resource "hookbase_event_type" "order_failed" {
  name             = "order.failed"
  display_name     = "Order Failed"
  description      = "Fired when an order processing fails"
  category         = "orders"
  default_priority = 0  # Critical
}
