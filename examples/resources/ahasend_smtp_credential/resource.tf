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

resource "ahasend_smtp_credential" "example" {
  name    = "app-smtp"
  scope   = "global"
  sandbox = false
}
