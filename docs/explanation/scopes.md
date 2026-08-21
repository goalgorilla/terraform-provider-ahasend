# Scopes

Required scopes depend on what you manage.

**Parent-only domains**

- `domains:read`
- `domains:write`
- `domains:delete:all` — delete any domain
- `domains:delete:{domain}` — delete a specific domain

**API keys, webhooks, SMTP credentials** (same account as the provider)

- API keys: `api-keys:read`, `api-keys:write`, `api-keys:delete`
- Webhooks:
  - `webhooks:read:all`, `webhooks:write:all`, `webhooks:delete:all`
  - `webhooks:read:{domain}`, `webhooks:write:{domain}`, `webhooks:delete:{domain}`
- SMTP credentials:
  - `smtp-credentials:read:all`, `smtp-credentials:write:all`, `smtp-credentials:delete:all`
  - `smtp-credentials:read:{domain}`, `smtp-credentials:write:{domain}`, `smtp-credentials:delete:{domain}`

**Sub account management** (parent key)

- `sub-accounts:read`, `sub-accounts:write`, `sub-accounts:delete`
- `sub-accounts:suspend` when using `suspended`
- `sub-account-api-keys:read`, `sub-account-api-keys:write`, `sub-account-api-keys:delete`

**Child domain keys** typically need the same domain scopes on the child key itself.

Partner early-access must be enabled for sub account operations. Parent-domain workflows should not require it.
