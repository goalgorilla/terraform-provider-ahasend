package provider

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// privateIdempotencyKey is the provider private-state key for create Idempotency-Key tokens.
const privateIdempotencyKey = "idempotency_key"

// privateData is the subset of privatestate.ProviderData used for idempotency tokens.
type privateData interface {
	GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics)
	SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics
}

// ensureIdempotencyToken returns a stable UUID for this create attempt.
// Values are JSON-encoded in private state (SetKey requires valid JSON).
// CreateRequest has no Private in the Plugin Framework; the token is stored on
// CreateResponse.Private so it is persisted after a successful create.
func ensureIdempotencyToken(ctx context.Context, private privateData) (string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if private == nil {
		diags.AddError(
			"Missing Private State",
			"Provider private state is unavailable; cannot create an idempotency key.",
		)
		return "", diags
	}

	existing, d := private.GetKey(ctx, privateIdempotencyKey)
	diags.Append(d...)
	if diags.HasError() {
		return "", diags
	}
	if len(existing) > 0 {
		var token string
		if err := json.Unmarshal(existing, &token); err != nil {
			diags.AddError(
				"Invalid Idempotency Key",
				"Private state idempotency_key is not valid JSON: "+err.Error(),
			)
			return "", diags
		}
		if token != "" {
			return token, diags
		}
	}

	token := uuid.NewString()
	encoded, err := json.Marshal(token)
	if err != nil {
		diags.AddError("Idempotency Key Encode Error", err.Error())
		return "", diags
	}
	diags.Append(private.SetKey(ctx, privateIdempotencyKey, encoded)...)
	return token, diags
}

// domainIdempotencyKey builds the HTTP Idempotency-Key for ahasend_domain creates.
func domainIdempotencyKey(token string) string {
	return "terraform-ahasend-domain-" + token
}

// apiKeyIdempotencyKey builds the HTTP Idempotency-Key for ahasend_api_key creates.
func apiKeyIdempotencyKey(token string) string {
	return "terraform-ahasend-api-key-" + token
}

// webhookIdempotencyKey builds the HTTP Idempotency-Key for ahasend_webhook creates.
func webhookIdempotencyKey(token string) string {
	return "terraform-ahasend-webhook-" + token
}

// smtpIdempotencyKey builds the HTTP Idempotency-Key for ahasend_smtp_credential creates.
func smtpIdempotencyKey(token string) string {
	return "terraform-ahasend-smtp-" + token
}

// subAccountAPIKeyIdempotencyKey builds the HTTP Idempotency-Key for ahasend_sub_account_api_key creates.
func subAccountAPIKeyIdempotencyKey(token string) string {
	return "terraform-ahasend-sub-account-api-key-" + token
}

// subAccountIdempotencyKey builds the HTTP Idempotency-Key for ahasend_sub_account creates.
func subAccountIdempotencyKey(token string) string {
	return "terraform-ahasend-sub-account-" + token
}
