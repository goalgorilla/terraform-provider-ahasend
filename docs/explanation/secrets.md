# Secrets and one-time keys

- Provider `api_key` is sensitive and may come from `AHASEND_API_KEY`.
- `ahasend_domain.dkim_private_key` is **write-only** (Terraform 1.11+): sent on create, never stored in state. Do not generate keys in the provider.
- `ahasend_api_key.secret_key`, `ahasend_sub_account_api_key.secret_key`, `ahasend_webhook.secret`, and `ahasend_smtp_credential.password` are returned **once** on create (idempotent create retries may replay within a short window). The provider stores them in state so other resources can reference them. GET responses typically omit these fields; Read preserves the prior state value. Import cannot recover secrets.

Terraform `sensitive` only redacts values in UI output. State and saved plan files may still contain credential values in cleartext. Use an encrypted, access-controlled remote backend, and never commit state or plan files to version control.
