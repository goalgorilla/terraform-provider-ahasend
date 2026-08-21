# AhaSend Terraform Provider

Unofficial [Terraform](https://www.terraform.io/) / [OpenTofu](https://opentofu.org/) provider for [AhaSend](https://ahasend.com/).

[AhaSend](https://ahasend.com/) is a transactional email platform: HTTP and SMTP sending, domain authentication (SPF/DKIM/DMARC), and delivery webhooks. This provider manages account setup as code — sending domains, API keys, webhooks, SMTP credentials, and (optionally) Platform Partner sub accounts.

**Not affiliated with AhaSend.** Maintained by [GoalGorilla / Open Social](https://www.getopensocial.com/). Latest release: [registry.terraform.io/providers/goalgorilla/ahasend](https://registry.terraform.io/providers/goalgorilla/ahasend).

## Install

```hcl
terraform {
  required_providers {
    ahasend = {
      source  = "goalgorilla/ahasend"
      version = "~> 0.1.0"
    }
  }
}

provider "ahasend" {
  # Or set AHASEND_API_KEY / AHASEND_ACCOUNT_ID
}
```

```bash
terraform init
terraform plan
```

## AhaSend links

| Link | What |
| --- | --- |
| [ahasend.com](https://ahasend.com/) | Product home |
| [Documentation](https://ahasend.com/docs) | Official product & API docs |
| [Dashboard](https://dash.ahasend.com/) | Account, domains, and API keys |
| [OpenAPI](https://ahasend.com/docs/openapi.yaml) | API contract (also pinned in [`openapi/`](openapi/openapi.yaml)) |
| [ahasend-go](https://github.com/AhaSend/ahasend-go) | Official Go SDK used at runtime |

## Documentation map

| Type | Where |
| --- | --- |
| **Tutorial** | [Getting started](docs/guides/getting-started.md) — Registry install or local `dev_overrides` |
| **How-to** | [Guides](docs/guides/) — custom DKIM/DNS, sub accounts, import |
| **Explanation** | [Explanation](docs/explanation/) — accounts, scopes, secrets, rate limits, DNS validation |
| **Reference** | [Registry docs](https://registry.terraform.io/providers/goalgorilla/ahasend/latest/docs) — also generated under [`docs/`](docs/) |

## Local development (`dev_overrides`)

For unreleased commits, build into this repo’s `bin/` (gitignored):

```bash
mkdir -p bin
go build -o bin/terraform-provider-ahasend .
# equivalent: make build
# if go is not on PATH: .tools/go/bin/go build -o bin/terraform-provider-ahasend .
```

```hcl
# ~/.terraformrc — absolute path to the directory that contains the binary
provider_installation {
  dev_overrides {
    "goalgorilla/ahasend" = "/Users/YOU/Sites/terraform-provider-ahasend/bin"
  }
  direct {}
}
```

With provider-only configs and `dev_overrides`, you can skip `terraform init` and run `plan` / `apply` directly. Prefer Registry install for normal use.

### OpenTofu

Same binary and `source`. Put the equivalent `provider_installation` block in `~/.tofurc` (or `$TOFU_CLI_CONFIG_FILE`) when using overrides.

## Authentication

| Argument | Environment variable |
| --- | --- |
| `api_key` | `AHASEND_API_KEY` |
| `account_id` | `AHASEND_ACCOUNT_ID` |
| `endpoint` (optional) | `AHASEND_ENDPOINT` |

## Development

```bash
make build         # compile
make test          # unit tests (no live AhaSend account)
make generate      # regenerate docs/ from schema + examples
make openapi-sync  # refresh openapi/openapi.yaml from https://ahasend.com/docs/openapi.yaml
```

The pinned [openapi/openapi.yaml](openapi/openapi.yaml) is a **contract snapshot** for docs and schema diffs. Runtime calls use [`ahasend-go`](https://github.com/AhaSend/ahasend-go). After `make openapi-sync`, review the diff, bump the SDK when needed, then fix resources if shapes changed.

Acceptance tests (`make testacc`) are not part of the default CI path yet.

## License

[MIT](LICENSE)
