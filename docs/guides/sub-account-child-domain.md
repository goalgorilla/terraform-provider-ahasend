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

## Sub-account SMTP credential

Child SMTP credentials use the same pattern on [`ahasend_smtp_credential`](../resources/smtp_credential): set `account_id` to the sub-account and `api_key` to a child key with `smtp-credentials:*` scopes.

```hcl
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
```

Rotating the child `api_key` is an in-place update (state only); it does not recreate the SMTP credential or its password.

## Import

See [Import an existing domain](import-domain). For child domains, import with a child provider alias first (so Read succeeds), then move to the resource `api_key` pattern above.

Child SMTP credentials are the same: `ImportState` does not store `api_key`, so the first Read needs child credentials. Before applying after import, set `name`, `scope`, `domains`, `sandbox`, and `account_id` to match the existing credential — those attributes force replacement (and a new one-time SMTP password) when they differ. Confirm with `terraform plan -refresh=false` that the plan shows no replacement.

```hcl
provider "ahasend" {
  alias      = "child"
  api_key    = var.child_api_key
  account_id = var.child_account_id
}

resource "ahasend_smtp_credential" "child" {
  provider   = ahasend.child
  account_id = var.child_account_id
  name       = "opensocial-smtp"
  scope      = "scoped"
  domains    = ["mail.customer.example.com"]
  sandbox    = false
}
```

```bash
terraform import ahasend_smtp_credential.child \
  00000000-0000-0000-0000-0000000000aa/00000000-0000-0000-0000-000000000030
```

Then switch the resource to the parent provider, set `api_key` (for example from `ahasend_sub_account_api_key`), and run `terraform apply -refresh=false` so the child key is written into state before the next refresh.

## Revoked child API key

Refresh uses the `api_key` already in state. If that key was revoked, Read fails before Terraform can apply a replacement key from configuration. Recover with refresh disabled, then resume normal applies:

```bash
terraform apply -refresh=false
```

After the new `api_key` is in state, subsequent plans and refreshes use it again.
