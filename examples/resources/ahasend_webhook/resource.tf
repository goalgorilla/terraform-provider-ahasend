terraform {
  required_providers {
    ahasend = {
      source = "goalgorilla/ahasend"
    }
  }
}

provider "ahasend" {
  api_key    = var.ahasend_api_key
  account_id = var.ahasend_account_id
}

variable "ahasend_api_key" {
  type      = string
  sensitive = true
}

variable "ahasend_account_id" {
  type = string
}

resource "ahasend_webhook" "example" {
  name    = "delivery-events"
  url     = "https://example.com/ahasend/webhook"
  scope   = "global"
  enabled = true

  on_delivered = true
  on_bounced   = true
  on_failed    = true
}
