terraform {
  required_providers {
    hookbase = {
      source = "hookbase/hookbase"
    }
  }
}

provider "hookbase" {
  api_key         = var.hookbase_api_key    # or set HOOKBASE_API_KEY env var
  organization_id = var.hookbase_org_id     # or set HOOKBASE_ORG_ID env var
  # api_url       = "https://api.hookbase.app"  # default, override for self-hosted
}

variable "hookbase_api_key" {
  type      = string
  sensitive = true
}

variable "hookbase_org_id" {
  type = string
}
