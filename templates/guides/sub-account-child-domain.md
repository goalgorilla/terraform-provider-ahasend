# Sub account and child domain

Platform Partner only. Prefer a **provider alias** for child credentials.

Terraform cannot initialize a provider configuration from attributes of resources managed in the same configuration (for example `api_key = ahasend_sub_account_api_key.bootstrap.secret_key`). Use a two-step apply: bootstrap the sub account and key first, then manage child-owned resources with credentials supplied via variables or environment.

## Step 1 — Bootstrap (parent provider)

```hcl
provider "ahasend" {
  # parent credentials via env or config
}

resource "ahasend_sub_account" "project" {
  name    = "Customer X"
  website = "customer.example.com"
}

resource "ahasend_sub_account_api_key" "bootstrap" {
  sub_account_id = ahasend_sub_account.project.id
  label          = "terraform"
  scopes = [
    "domains:read",
    "domains:write",
    "domains:delete:all",
  ]
}

output "child_account_id" {
  value = ahasend_sub_account.project.id
}

output "child_api_key" {
  value     = ahasend_sub_account_api_key.bootstrap.secret_key
  sensitive = true
}
```

Apply, then copy the outputs into environment variables or a secrets store (the `secret_key` is returned once on create; import cannot recover it).

## Step 2 — Child resources (separate apply / workspace)

```hcl
variable "child_api_key" {
  type      = string
  sensitive = true
}

variable "child_account_id" {
  type = string
}

provider "ahasend" {
  alias      = "child"
  api_key    = var.child_api_key
  account_id = var.child_account_id
}

resource "ahasend_domain" "child_sending" {
  provider = ahasend.child
  domain   = "mail.customer.example.com"
}
```

Alternatively, keep using the parent provider and set optional `account_id` on `ahasend_domain` while authenticating with a key that can act on that child account.
