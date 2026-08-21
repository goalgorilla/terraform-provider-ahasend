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

resource "ahasend_sub_account" "example" {
  name    = "Customer X"
  website = "customer.example.com"
}

resource "ahasend_sub_account_api_key" "example" {
  sub_account_id = ahasend_sub_account.example.id
  label          = "terraform"
  scopes = [
    "domains:read",
    "domains:write",
    "domains:delete:all",
  ]
}
