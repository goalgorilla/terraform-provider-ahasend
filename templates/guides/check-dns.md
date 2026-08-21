# Trigger a DNS check

`check_dns` (default `true`) calls AhaSend check-dns after create, update, and read. Results may be cached for about 60 seconds.

```hcl
resource "ahasend_domain" "sending" {
  domain    = "mail.example.com"
  check_dns = true
}
```

- Set `check_dns = false` to skip the check-dns call and only GET the domain.
- A failed check-dns HTTP call becomes a **warning**, not an apply error.
- `dns_valid = false` never fails apply. Publish `dns_records`, then `terraform apply` or `terraform refresh` again.
