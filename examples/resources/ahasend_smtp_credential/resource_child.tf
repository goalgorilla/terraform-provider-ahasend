resource "ahasend_sub_account_api_key" "smtp" {
  sub_account_id = ahasend_sub_account.project.id
  label          = "terraform-smtp"
  scopes = [
    "smtp-credentials:read:all",
    "smtp-credentials:write:all",
    "smtp-credentials:delete:all",
  ]
}

resource "ahasend_smtp_credential" "child" {
  account_id = ahasend_sub_account.project.id
  api_key    = ahasend_sub_account_api_key.smtp.secret_key
  name       = "opensocial-smtp"
  scope      = "scoped"
  domains    = ["mail.customer.example.com"]
  sandbox    = false
}
