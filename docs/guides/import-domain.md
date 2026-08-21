# Import an existing domain

Import by domain name (uses the provider `account_id`), or `ACCOUNT_ID/DOMAIN` for a specific account:

```bash
terraform import ahasend_domain.sending mail.example.com
terraform import ahasend_domain.child 00000000-0000-0000-0000-000000000004/mail.customer.example.com
```

After import, run `terraform plan` and align configuration with remote settings (`tracking_subdomain`, `dkim_selector`, and so on).
