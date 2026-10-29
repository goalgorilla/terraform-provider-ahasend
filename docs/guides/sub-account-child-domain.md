# Sub account and child domain

Platform Partner only. AhaSend rejects the **parent** API key on child domain routes (`/v2/accounts/{child_id}/domains`). Use a child API key for `ahasend_domain`.

Terraform cannot put a resource attribute on a **provider** block in the same configuration. Prefer the resource-level `api_key` on `ahasend_domain` so one apply can create the child key and the domain.

## Single apply (parent provider + resource `api_key`)

```hcl
provider "ahasend" {
  # parent credentials via env or config
}

resource "ahasend_sub_account" "project" {
  name    = "Customer X"
  website = "customer.example.com"
}

resource "ahasend_sub_account_api_key" "domains" {
  sub_account_id = ahasend_sub_account.project.id
  label          = "terraform-domains"
  scopes = [
    "domains:read",
    "domains:write",
    "domains:delete:all",
  ]
}

resource "ahasend_domain" "child_sending" {
  account_id = ahasend_sub_account.project.id
  api_key    = ahasend_sub_account_api_key.domains.secret_key
  domain     = "mail.customer.example.com"
}
```

The domain resource stores `api_key` sensitive in state so Read/Delete can authenticate. Provider aliases remain valid when credentials are supplied from variables or a prior apply.

## Import

See [Import an existing domain](import-domain). For child domains, import with a child provider alias first (so Read succeeds), then move to the resource `api_key` pattern above.

## Revoked child API key

Refresh uses the `api_key` already in state. If that key was revoked, Read fails before Terraform can apply a replacement key from configuration. Recover with refresh disabled, then resume normal applies:

```bash
terraform apply -refresh=false
```

After the new `api_key` is in state, subsequent plans and refreshes use it again.
