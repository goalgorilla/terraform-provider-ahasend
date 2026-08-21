# Custom DKIM, return-path, and tracking

Configure optional subdomain and Partner DKIM fields on `ahasend_domain`.

```hcl
resource "ahasend_domain" "sending" {
  domain                     = "mail.example.com"
  tracking_subdomain         = "t"
  return_path_subdomain      = "rp"
  subscription_subdomain     = "prefs"
  media_subdomain            = "media"
  dkim_selector              = "os"
  dkim_rotation_interval_days = 45

  # Platform Partner + Terraform 1.11+: write-only, not stored in state
  # dkim_private_key = file("${path.module}/dkim.pem")
}
```

`dkim_selector` and `dkim_private_key` are Platform Partner features. On HTTP 403 the provider surfaces the API `message` (AhaSend has no stable machine error codes).

Omit fields you do not need so AhaSend keeps product defaults.
