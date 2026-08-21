# Getting started

Create one sending domain on your **parent** AhaSend account. OpenTofu works the same with `tofu` instead of `terraform`.

## Prerequisites

- Terraform CLI 1.11+ (required if you use write-only `dkim_private_key`; otherwise 1.x is fine)
- An AhaSend API key with at least `domains:read`, `domains:write`, and `domains:delete:mail.example.com`
- Your AhaSend account UUID

## 1. Install the provider

### From the Terraform Registry (preferred after publish)

```hcl
terraform {
  required_providers {
    ahasend = {
      source  = "goalgorilla/ahasend"
      version = "~> 0.1.0"
    }
  }
}

provider "ahasend" {}
```

```bash
terraform init
```

### Local build (`dev_overrides`)

For unreleased commits: Go 1.25+, then:

```bash
git clone https://github.com/goalgorilla/terraform-provider-ahasend.git
cd terraform-provider-ahasend
mkdir -p bin
go build -o bin/terraform-provider-ahasend .
# if go is not on PATH and .tools/go exists:
# .tools/go/bin/go build -o bin/terraform-provider-ahasend .
```

Create or edit `~/.terraformrc` with the **absolute** path to this repo’s `bin/`:

```hcl
provider_installation {
  dev_overrides {
    "goalgorilla/ahasend" = "/Users/YOU/Sites/terraform-provider-ahasend/bin"
  }
  direct {}
}
```

With provider-only configs and `dev_overrides`, you may skip `terraform init` and run `plan` / `apply` directly.

## 2. Authenticate

```bash
export AHASEND_API_KEY="aha-sk-..."
export AHASEND_ACCOUNT_ID="00000000-0000-0000-0000-000000000000"
```

## 3. Create a domain

```hcl
resource "ahasend_domain" "sending" {
  domain                = "mail.example.com"
  tracking_subdomain    = "t"
  return_path_subdomain = "rp"
}
```

```bash
terraform plan
terraform apply
```

Inspect `dns_records` and `dns_valid` in state. Apply does **not** fail when DNS is not yet valid — publish the returned records at your DNS provider, then refresh or re-apply with `check_dns = true` (the default).

## OpenTofu

Use the same `source` / version constraint. For local overrides, put the `provider_installation` block in the OpenTofu CLI config (`~/.tofurc` or `$TOFU_CLI_CONFIG_FILE`).
