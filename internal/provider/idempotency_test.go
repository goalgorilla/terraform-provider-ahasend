package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func TestAPIKeyIdempotencyKey(t *testing.T) {
	t.Parallel()
	token := "11111111-1111-1111-1111-111111111111"
	got := apiKeyIdempotencyKey(token)
	want := "terraform-ahasend-api-key-11111111-1111-1111-1111-111111111111"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestWebhookIdempotencyKey(t *testing.T) {
	t.Parallel()
	token := "22222222-2222-2222-2222-222222222222"
	got := webhookIdempotencyKey(token)
	want := "terraform-ahasend-webhook-22222222-2222-2222-2222-222222222222"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSMTPIdempotencyKey(t *testing.T) {
	t.Parallel()
	token := "33333333-3333-3333-3333-333333333333"
	got := smtpIdempotencyKey(token)
	want := "terraform-ahasend-smtp-33333333-3333-3333-3333-333333333333"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDomainIdempotencyKey(t *testing.T) {
	t.Parallel()
	token := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	got := domainIdempotencyKey(token)
	want := "terraform-ahasend-domain-aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSubAccountAPIKeyIdempotencyKey(t *testing.T) {
	t.Parallel()
	token := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	got := subAccountAPIKeyIdempotencyKey(token)
	want := "terraform-ahasend-sub-account-api-key-bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSubAccountIdempotencyKey(t *testing.T) {
	t.Parallel()
	token := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	got := subAccountIdempotencyKey(token)
	want := "terraform-ahasend-sub-account-cccccccc-cccc-cccc-cccc-cccccccccccc"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEnsureIdempotencyTokenJSONRoundTrip(t *testing.T) {
	t.Parallel()

	store := &memPrivate{data: map[string][]byte{}}
	token1, diags := ensureIdempotencyToken(context.Background(), store)
	if diags.HasError() {
		t.Fatalf("first ensure: %v", diags)
	}
	if token1 == "" {
		t.Fatal("empty token")
	}
	if !json.Valid(store.data[privateIdempotencyKey]) {
		t.Fatalf("stored value is not valid JSON: %q", store.data[privateIdempotencyKey])
	}

	token2, diags := ensureIdempotencyToken(context.Background(), store)
	if diags.HasError() {
		t.Fatalf("second ensure: %v", diags)
	}
	if token1 != token2 {
		t.Fatalf("expected reuse, got %q then %q", token1, token2)
	}
}

type memPrivate struct {
	data map[string][]byte
}

func (m *memPrivate) GetKey(ctx context.Context, key string) ([]byte, diag.Diagnostics) {
	return m.data[key], nil
}

func (m *memPrivate) SetKey(ctx context.Context, key string, value []byte) diag.Diagnostics {
	m.data[key] = append([]byte(nil), value...)
	return nil
}
