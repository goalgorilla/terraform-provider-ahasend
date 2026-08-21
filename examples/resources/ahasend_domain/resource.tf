terraform {
  required_providers {
    ahasend = {
      source = "goalgorilla/ahasend"
    }
  }
}

provider "ahasend" {
  # Prefer environment variables in local development:
  #   AHASEND_API_KEY
  #   AHASEND_ACCOUNT_ID
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

resource "ahasend_domain" "example" {
  domain                = "mail.example.com"
  tracking_subdomain    = "t"
  return_path_subdomain = "rp"
}
