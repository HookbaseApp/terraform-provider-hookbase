# Basic Stripe webhook source
resource "hookbase_source" "stripe" {
  name          = "stripe-production"
  slug          = "stripe-prod"
  provider_type = "stripe"
  signing_secret = var.stripe_webhook_secret
  reject_invalid_signatures = true
}

# GitHub source with IP filtering
resource "hookbase_source" "github" {
  name           = "github-webhooks"
  slug           = "github"
  provider_type  = "github"
  ip_filter_mode = "allowlist"
  ip_allowlist   = ["140.82.112.0/20", "185.199.108.0/22"]
}

# High-security source with deduplication and transient mode
resource "hookbase_source" "payments" {
  name               = "payment-events"
  slug               = "payments"
  provider_type      = "custom"
  dedup_enabled      = true
  dedup_strategy     = "idempotency_key"
  dedup_window_hours = 48
  transient_mode     = true  # HIPAA compliance — no payload storage
}

# Source restricted to specific HTTP methods (omit allowed_methods to accept any verb)
resource "hookbase_source" "poller" {
  name            = "status-poller"
  slug            = "status-poller"
  provider_type   = "generic"
  allowed_methods = ["GET", "HEAD"]
}

# Source for a catalog vendor that has no verifier of its own (Salesforce), checked as Custom
# HMAC. catalog_provider records the vendor, so a bridge destination does not count an IP
# allowlist on this source as proof of the sender: Salesforce sends every customer's webhooks
# from the same addresses.
resource "hookbase_source" "salesforce" {
  name             = "salesforce"
  slug             = "salesforce"
  provider_type    = "custom"
  catalog_provider = "salesforce"
}

# WhatsApp source: Hookbase generates the Meta "Verify token" unless verify_token is set.
# signing_secret is the Meta App Secret, which Meta issues.
resource "hookbase_source" "whatsapp" {
  name           = "whatsapp"
  slug           = "whatsapp"
  provider_type  = "whatsapp"
  signing_secret = var.meta_app_secret
}

output "whatsapp_verify_token" {
  value     = hookbase_source.whatsapp.verify_token
  sensitive = true
}
