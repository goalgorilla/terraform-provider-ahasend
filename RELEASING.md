# Releasing (maintainers)

Internal checklist for GitHub Releases + Terraform Registry. Not published under `docs/` (Registry docs).

First public tag: **`v0.1.0`**.

## One-time setup

1. Public repo `goalgorilla/terraform-provider-ahasend`
2. GPG key (RSA/DSA): register **public** key on [registry.terraform.io](https://registry.terraform.io/); store private key + passphrase as GitHub secrets `GPG_PRIVATE_KEY` and `PASSPHRASE`
3. Green CI on `main`

Automation: `.github/workflows/release.yml` + `.goreleaser.yml` (first release is created as a **draft** so you can review assets before publishing).

## Cut a release

```bash
git checkout main && git pull
# no branch named like the tag
git tag -a v0.1.0 -m "v0.1.0"
git push origin v0.1.0
```

1. Watch the Release workflow; review the **draft** GitHub Release assets; publish the draft.
2. First time only: [Publish → Provider](https://registry.terraform.io/publish/provider) for this repo.

Consumers:

```hcl
terraform {
  required_providers {
    ahasend = {
      source  = "goalgorilla/ahasend"
      version = "~> 0.1.0"
    }
  }
}
```

Later tags: `v0.1.1`, `v0.2.0`, …
