# Import an existing domain

Import by domain name (uses the provider `account_id`), or `ACCOUNT_ID/DOMAIN` for a specific account:

```bash
terraform import ahasend_domain.sending mail.example.com
terraform import ahasend_domain.child 00000000-0000-0000-0000-000000000004/mail.customer.example.com
```

After import, run `terraform plan` and align configuration with remote settings (`tracking_subdomain`, `dkim_selector`, and so on).

## Child domains

`ImportState` does not store `api_key`. The first Read after import uses the **provider** credentials. For a sub-account domain that returns HTTP 401 with the parent key, import while authenticating as the child (provider alias), then migrate to the resource-level `api_key`:

```hcl
provider "ahasend" {
  alias      = "child"
  api_key    = var.child_api_key
  account_id = var.child_account_id
}

resource "ahasend_domain" "child_sending" {
  provider   = ahasend.child
  account_id = var.child_account_id
  domain     = "mail.customer.example.com"
}
```

```bash
terraform import ahasend_domain.child_sending \
  00000000-0000-0000-0000-000000000004/mail.customer.example.com
```

Then switch the resource to the parent provider, set `api_key` (for example from `ahasend_sub_account_api_key`), and run `terraform apply -refresh=false` so the child key is written into state before the next refresh.
