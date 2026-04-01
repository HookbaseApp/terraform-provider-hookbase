terraform {
  required_providers {
    hookbase = {
      source = "registry.terraform.io/hookbase/hookbase"
      version = "0.1.0"
    }
  }
}

provider "hookbase" {
  api_url         = "http://localhost:8787"  # Local dev server
  api_key         = var.api_key
  organization_id = var.org_id
}

variable "api_key" {
  type      = string
  sensitive = true
}

variable "org_id" {
  type = string
}

# ============================================================
# INBOUND: Source → Transform → Filter → Destination → Route
# ============================================================

resource "hookbase_source" "smoke_test" {
  name          = "smoke-test-source"
  slug          = "smoke-test-src"
  provider_type = "custom"
  description   = "Created by Terraform smoke test"

  dedup_enabled  = true
  dedup_strategy = "payload_hash"
}

resource "hookbase_transform" "smoke_test" {
  name           = "smoke-test-transform"
  transform_type = "jsonata"
  code           = <<-EOT
    {
      "event": type,
      "data": data
    }
  EOT
}

resource "hookbase_filter" "smoke_test" {
  name  = "smoke-test-filter"
  logic = "AND"
  conditions = jsonencode([
    {
      field    = "type"
      operator = "exists"
      value    = true
    }
  ])
}

resource "hookbase_destination" "smoke_test" {
  name = "smoke-test-destination"
  slug = "smoke-test-dest"
  url  = "https://httpbin.org/post"
}

resource "hookbase_route" "smoke_test" {
  name           = "smoke-test-route"
  source_id      = hookbase_source.smoke_test.id
  destination_id = hookbase_destination.smoke_test.id
  transform_id   = hookbase_transform.smoke_test.id
  filter_id      = hookbase_filter.smoke_test.id
}

# ============================================================
# OUTBOUND: Event Type → Application → Endpoint → Subscription
# ============================================================

resource "hookbase_event_type" "smoke_test" {
  name             = "smoke.test.event"
  display_name     = "Smoke Test Event"
  description      = "Created by Terraform smoke test"
  category         = "testing"
  default_priority = 2
}

resource "hookbase_webhook_application" "smoke_test" {
  name        = "Smoke Test App"
  external_id = "smoke-test-ext-001"

  rate_limit_per_second = 10
  rate_limit_per_minute = 100
  rate_limit_per_hour   = 1000
}

resource "hookbase_webhook_endpoint" "smoke_test" {
  application_id  = hookbase_webhook_application.smoke_test.id
  url             = "https://httpbin.org/post"
  description     = "Smoke test endpoint"
  timeout_seconds = 10

  circuit_failure_threshold = 3
  circuit_cooldown_seconds  = 30
}

resource "hookbase_webhook_subscription" "smoke_test" {
  endpoint_id   = hookbase_webhook_endpoint.smoke_test.id
  event_type_id = hookbase_event_type.smoke_test.id
}

# ============================================================
# OUTPUTS — verify everything was created
# ============================================================

output "source_id" {
  value = hookbase_source.smoke_test.id
}

output "source_ingest_url" {
  value = hookbase_source.smoke_test.ingest_url
}

output "route_id" {
  value = hookbase_route.smoke_test.id
}

output "webhook_app_id" {
  value = hookbase_webhook_application.smoke_test.id
}

output "webhook_endpoint_id" {
  value = hookbase_webhook_endpoint.smoke_test.id
}

output "subscription_id" {
  value = hookbase_webhook_subscription.smoke_test.id
}

output "event_type_id" {
  value = hookbase_event_type.smoke_test.id
}
