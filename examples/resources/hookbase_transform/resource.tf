# JSONata transform to normalize Stripe events
resource "hookbase_transform" "stripe_normalize" {
  name           = "normalize-stripe"
  transform_type = "jsonata"
  code           = <<-EOT
    {
      "event_type": type,
      "customer_id": data.object.customer,
      "amount": data.object.amount / 100,
      "currency": data.object.currency,
      "timestamp": created
    }
  EOT
}

# JavaScript transform for complex logic
resource "hookbase_transform" "enrich_payload" {
  name           = "enrich-payload"
  transform_type = "javascript"
  code           = file("${path.module}/transforms/enrich.js")
}
