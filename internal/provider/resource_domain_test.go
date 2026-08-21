package provider

import (
	"context"
	"testing"
	"time"

	"github.com/AhaSend/ahasend-go/models/responses"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestResolveAccountID(t *testing.T) {
	t.Parallel()

	providerAccount := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	r := &DomainResource{
		client: &ahasendClient{accountID: providerAccount},
	}

	t.Run("defaults to provider", func(t *testing.T) {
		t.Parallel()
		id, diags := r.resolveAccountID(types.StringNull())
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if id != providerAccount {
			t.Fatalf("got %s, want %s", id, providerAccount)
		}
	})

	t.Run("override", func(t *testing.T) {
		t.Parallel()
		override := "22222222-2222-2222-2222-222222222222"
		id, diags := r.resolveAccountID(types.StringValue(override))
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if id.String() != override {
			t.Fatalf("got %s, want %s", id, override)
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		_, diags := r.resolveAccountID(types.StringValue("not-a-uuid"))
		if !diags.HasError() {
			t.Fatal("expected error diagnostics")
		}
	})
}

func TestFlattenDomainDNSRecords(t *testing.T) {
	t.Parallel()

	label := "SPF"
	domainID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	accountID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	apiDomain := &domainAPIResponse{
		Domain: responses.Domain{
			ID:            domainID,
			AccountID:     accountID,
			Domain:        "mail.example.com",
			DNSValid:      false,
			RotationReady: false,
			CreatedAt:     time.Unix(0, 0).UTC(),
			UpdatedAt:     time.Unix(0, 0).UTC(),
			DNSRecords: []responses.DNSRecord{
				{
					Type:       "TXT",
					Host:       "@",
					Content:    "v=spf1 include:spf.ahasend.com ~all",
					Required:   true,
					Propagated: false,
					Label:      &label,
				},
			},
		},
	}

	r := &DomainResource{}
	prior := DomainResourceModel{
		CheckDNS: types.BoolValue(true),
	}
	var out DomainResourceModel
	diags := r.flattenDomain(context.Background(), apiDomain, prior, &out)
	if diags.HasError() {
		t.Fatalf("flattenDomain diagnostics: %v", diags)
	}
	if out.DNSValid.ValueBool() {
		t.Fatal("expected dns_valid false")
	}
	if out.Domain.ValueString() != "mail.example.com" {
		t.Fatalf("domain = %q", out.Domain.ValueString())
	}
	if out.DNSRecords.IsNull() || out.DNSRecords.IsUnknown() {
		t.Fatal("expected dns_records list")
	}
	if len(out.DNSRecords.Elements()) != 1 {
		t.Fatalf("expected 1 dns record, got %d", len(out.DNSRecords.Elements()))
	}
	if !out.DKIMPrivateKey.IsNull() {
		t.Fatal("write-only dkim_private_key must be null in state")
	}
}

// TestFlattenDomainAPIDefaultsOmitsConfig covers create with only `domain` set:
// omitted subdomain knobs are unknown/null in plan; AhaSend returns product defaults
// that must land in state (Optional+Computed), not stay null.
func TestFlattenDomainAPIDefaultsOmitsConfig(t *testing.T) {
	t.Parallel()

	domainID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	accountID := uuid.MustParse("66666666-6666-6666-6666-666666666666")
	tracking := "track"
	returnPath := "rp"
	subscription := "subs"
	media := "media"
	rotationDays := 45

	apiDomain := &domainAPIResponse{
		Domain: responses.Domain{
			ID:                       domainID,
			AccountID:                accountID,
			Domain:                   "mail.smoke.example.com",
			DNSValid:                 false,
			RotationReady:            false,
			TrackingSubdomain:        &tracking,
			ReturnPathSubdomain:      &returnPath,
			SubscriptionSubdomain:    &subscription,
			MediaSubdomain:           &media,
			DKIMRotationIntervalDays: &rotationDays,
			CreatedAt:                time.Unix(0, 0).UTC(),
			UpdatedAt:                time.Unix(0, 0).UTC(),
		},
	}

	r := &DomainResource{}
	// Simulate create plan: only domain configured; optional overrides unknown/null.
	prior := DomainResourceModel{
		Domain:                   types.StringValue("mail.smoke.example.com"),
		TrackingSubdomain:        types.StringUnknown(),
		ReturnPathSubdomain:      types.StringUnknown(),
		SubscriptionSubdomain:    types.StringUnknown(),
		MediaSubdomain:           types.StringUnknown(),
		DKIMSelector:             types.StringUnknown(),
		DKIMRotationIntervalDays: types.Int64Unknown(),
		CheckDNS:                 types.BoolValue(true),
	}
	var out DomainResourceModel
	diags := r.flattenDomain(context.Background(), apiDomain, prior, &out)
	if diags.HasError() {
		t.Fatalf("flattenDomain diagnostics: %v", diags)
	}

	assertStringAttr := func(name string, got types.String, want string) {
		t.Helper()
		if got.IsNull() || got.IsUnknown() {
			t.Fatalf("%s: expected %q, got null/unknown", name, want)
		}
		if got.ValueString() != want {
			t.Fatalf("%s = %q, want %q", name, got.ValueString(), want)
		}
	}
	assertStringAttr("tracking_subdomain", out.TrackingSubdomain, tracking)
	assertStringAttr("return_path_subdomain", out.ReturnPathSubdomain, returnPath)
	assertStringAttr("subscription_subdomain", out.SubscriptionSubdomain, subscription)
	assertStringAttr("media_subdomain", out.MediaSubdomain, media)

	if out.DKIMRotationIntervalDays.IsNull() || out.DKIMRotationIntervalDays.IsUnknown() {
		t.Fatal("dkim_rotation_interval_days: expected API default, got null/unknown")
	}
	if out.DKIMRotationIntervalDays.ValueInt64() != int64(rotationDays) {
		t.Fatalf("dkim_rotation_interval_days = %d, want %d", out.DKIMRotationIntervalDays.ValueInt64(), rotationDays)
	}
}

// TestFlattenDomainRotationDaysNullWhenAPIOmits ensures omitted config + nil API
// becomes known null, not unknown (Terraform rejects unknown after apply).
func TestFlattenDomainRotationDaysNullWhenAPIOmits(t *testing.T) {
	t.Parallel()

	domainID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	accountID := uuid.MustParse("88888888-8888-8888-8888-888888888888")
	media := "media"
	subscription := "subs"

	apiDomain := &domainAPIResponse{
		Domain: responses.Domain{
			ID:                    domainID,
			AccountID:             accountID,
			Domain:                "mail.smoke.example.com",
			DNSValid:              false,
			RotationReady:         false,
			SubscriptionSubdomain: &subscription,
			MediaSubdomain:        &media,
			// DKIMRotationIntervalDays intentionally nil (account default / not returned).
			CreatedAt: time.Unix(0, 0).UTC(),
			UpdatedAt: time.Unix(0, 0).UTC(),
		},
	}

	r := &DomainResource{}
	prior := DomainResourceModel{
		Domain:                   types.StringValue("mail.smoke.example.com"),
		MediaSubdomain:           types.StringUnknown(),
		SubscriptionSubdomain:    types.StringUnknown(),
		DKIMRotationIntervalDays: types.Int64Unknown(),
		CheckDNS:                 types.BoolValue(true),
	}
	var out DomainResourceModel
	diags := r.flattenDomain(context.Background(), apiDomain, prior, &out)
	if diags.HasError() {
		t.Fatalf("flattenDomain diagnostics: %v", diags)
	}
	if out.DKIMRotationIntervalDays.IsUnknown() {
		t.Fatal("dkim_rotation_interval_days must not remain unknown after flatten")
	}
	if !out.DKIMRotationIntervalDays.IsNull() {
		t.Fatalf("dkim_rotation_interval_days = %v, want null", out.DKIMRotationIntervalDays)
	}
	if out.MediaSubdomain.ValueString() != media {
		t.Fatalf("media_subdomain = %q", out.MediaSubdomain.ValueString())
	}
}

func TestParseUUIDAttr(t *testing.T) {
	t.Parallel()

	id := "55555555-5555-5555-5555-555555555555"
	got, diags := parseUUIDAttr(types.StringValue(id), "id")
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if got.String() != id {
		t.Fatalf("got %s", got)
	}

	_, diags = parseUUIDAttr(types.StringNull(), "id")
	if !diags.HasError() {
		t.Fatal("expected error for null")
	}
}

func TestShouldCheckDNS(t *testing.T) {
	t.Parallel()
	if !shouldCheckDNS(types.StringNull()) {
		t.Fatal("null should check")
	}
	if !shouldCheckDNS(types.StringValue(time.Now().UTC().Add(-2 * time.Minute).Format(time.RFC3339))) {
		t.Fatal("old timestamp should check")
	}
	if shouldCheckDNS(types.StringValue(time.Now().UTC().Format(time.RFC3339))) {
		t.Fatal("recent timestamp should skip check")
	}
}

func TestOptionalStringNilAPIUnknownPrior(t *testing.T) {
	t.Parallel()

	got := optionalString(nil, types.StringUnknown())
	if !got.IsNull() {
		t.Fatalf("expected null, got %#v", got)
	}

	apiVal := "media"
	got = optionalString(&apiVal, types.StringUnknown())
	if got.ValueString() != "media" {
		t.Fatalf("got %q, want media", got.ValueString())
	}
}
