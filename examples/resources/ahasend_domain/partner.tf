resource "ahasend_domain" "partner" {
  domain        = "mail.example.com"
  dkim_selector = "os" # Platform Partner accounts only
}
