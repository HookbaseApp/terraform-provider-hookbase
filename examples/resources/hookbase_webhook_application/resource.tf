# Outbound webhook application for a customer
resource "hookbase_webhook_application" "acme" {
  name        = "Acme Corp"
  external_id = "cust_acme_123"

  rate_limit_per_second = 50
  rate_limit_per_minute = 500
}
