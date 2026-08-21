terraform {
  required_providers {
    ahasend = {
      source = "goalgorilla/ahasend"
    }
  }
}

# Configure with AHASEND_API_KEY and AHASEND_ACCOUNT_ID, or set explicitly:
provider "ahasend" {
  # api_key    = var.ahasend_api_key
  # account_id = var.ahasend_account_id
}
