# Rate limits and idempotency

AhaSend general APIs allow about **100 req/s** with a **200** burst per account (statistics are 1 req/s; this provider does not call those in v1). Exceeding limits returns HTTP 429.

The provider builds the official [`ahasend-go`](https://github.com/AhaSend/ahasend-go) client with:

- client-side rate limiting enabled (`WithRateLimit(true)`)
- default retry configuration (`WithRetryConfig(DefaultRetryConfig())`) for 429 and typical 5xx

Create operations send an `Idempotency-Key` for:

| Resource | Key strategy |
| --- | --- |
| `ahasend_domain` | Private-state UUID token |
| `ahasend_api_key` | Private-state UUID token |
| `ahasend_webhook` | Private-state UUID token |
| `ahasend_smtp_credential` | Private-state UUID token |
| `ahasend_sub_account_api_key` | Private-state UUID token |
| `ahasend_sub_account` | Private-state UUID token |

If retries are exhausted, the provider surfaces a clear API diagnostic. A failed create after the server already succeeded may still create a duplicate when the private-state token was never persisted (UUID-token resources).
