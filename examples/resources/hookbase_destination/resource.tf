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
