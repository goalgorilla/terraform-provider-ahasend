# Accounts and sub accounts

The provider authenticates as **one** AhaSend account (`api_key` + `account_id`). That is usually your Platform Partner / parent account.

- Creating a domain with the default provider uses `POST /v2/accounts/{account_id}/domains` on that account. Domains are not “sub account-only.”
- Sub accounts are an optional layer: create a child with `ahasend_sub_account`, bootstrap a key with `ahasend_sub_account_api_key`, then point a **provider alias** at the child credentials for child-owned domains.

Official AhaSend docs use a child bearer token for `/v2/accounts/{child_id}/domains`. Prefer the alias pattern over putting `api_key` on every resource.
