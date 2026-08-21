package provider

import (
	"context"
	"testing"
	"time"

	"github.com/AhaSend/ahasend-go/models/responses"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestWebhookPlanToCreateRequest(t *testing.T) {
	t.Parallel()

	t.Run("global null domains", func(t *testing.T) {
		t.Parallel()
		plan := WebhookResourceModel{
			Name:                 types.StringValue("delivery-events"),
			URL:                  types.StringValue("https://example.com/hook"),
			Enabled:              types.BoolValue(true),
			Scope:                types.StringValue("global"),
			OnDelivered:          types.BoolValue(true),
			OnBounced:            types.BoolValue(true),
			OnFailed:             types.BoolValue(true),
			OnReception:          types.BoolValue(false),
			OnTransientError:     types.BoolValue(false),
			OnSuppressed:         types.BoolValue(false),
			OnOpened:             types.BoolValue(false),
			OnClicked:            types.BoolValue(false),
			OnSuppressionCreated: types.BoolValue(false),
			OnDNSError:           types.BoolValue(false),
			Domains:              types.ListNull(types.StringType),
		}

		req, diags := webhookPlanToCreateRequest(context.Background(), plan)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if req.Name != "delivery-events" || req.URL != "https://example.com/hook" {
			t.Fatalf("unexpected name/url: %+v", req)
		}
		if req.Enabled == nil || !*req.Enabled {
			t.Fatal("expected Enabled true")
		}
		if !req.OnDelivered || !req.OnBounced || !req.OnFailed {
			t.Fatalf("expected event flags set, got %+v", req)
		}
		if req.Domains != nil {
			t.Fatalf("expected nil domains for global without list, got %v", req.Domains)
		}
	})

	t.Run("scoped domains", func(t *testing.T) {
		t.Parallel()
		domains, d := types.ListValueFrom(context.Background(), types.StringType, []string{"mail.example.com"})
		if d.HasError() {
			t.Fatalf("list: %v", d)
		}
		plan := WebhookResourceModel{
			Name:                 types.StringValue("scoped-hook"),
			URL:                  types.StringValue("https://example.com/scoped"),
			Enabled:              types.BoolValue(true),
			Scope:                types.StringValue("scoped"),
			Domains:              domains,
			OnDelivered:          types.BoolValue(true),
			OnBounced:            types.BoolValue(false),
			OnFailed:             types.BoolValue(false),
			OnReception:          types.BoolValue(false),
			OnTransientError:     types.BoolValue(false),
			OnSuppressed:         types.BoolValue(false),
			OnOpened:             types.BoolValue(false),
			OnClicked:            types.BoolValue(false),
			OnSuppressionCreated: types.BoolValue(false),
			OnDNSError:           types.BoolValue(false),
		}

		req, diags := webhookPlanToCreateRequest(context.Background(), plan)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		if req.Scope != "scoped" {
			t.Fatalf("scope = %q", req.Scope)
		}
		if req.Domains == nil || len(*req.Domains) != 1 || (*req.Domains)[0] != "mail.example.com" {
			t.Fatalf("domains = %v", req.Domains)
		}
	})
}

func TestWebhookPlanToUpdateRequestOmitsUnknownDomains(t *testing.T) {
	t.Parallel()

	plan := WebhookResourceModel{
		Name:                 types.StringValue("delivery-events"),
		URL:                  types.StringValue("https://example.com/hook"),
		Enabled:              types.BoolValue(true),
		Scope:                types.StringValue("scoped"),
		Domains:              types.ListUnknown(types.StringType),
		OnDelivered:          types.BoolValue(true),
		OnBounced:            types.BoolValue(false),
		OnFailed:             types.BoolValue(false),
		OnReception:          types.BoolValue(false),
		OnTransientError:     types.BoolValue(false),
		OnSuppressed:         types.BoolValue(false),
		OnOpened:             types.BoolValue(false),
		OnClicked:            types.BoolValue(false),
		OnSuppressionCreated: types.BoolValue(false),
		OnDNSError:           types.BoolValue(false),
	}

	req, diags := webhookPlanToUpdateRequest(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if req.Domains != nil {
		t.Fatalf("expected Domains omitted (nil) when plan unknown, got %v", req.Domains)
	}
}

func TestWebhookPlanToUpdateRequestSendsKnownEmptyDomains(t *testing.T) {
	t.Parallel()

	empty, d := types.ListValueFrom(context.Background(), types.StringType, []string{})
	if d.HasError() {
		t.Fatalf("list: %v", d)
	}
	plan := WebhookResourceModel{
		Name:                 types.StringValue("delivery-events"),
		URL:                  types.StringValue("https://example.com/hook"),
		Enabled:              types.BoolValue(true),
		Scope:                types.StringValue("global"),
		Domains:              empty,
		OnDelivered:          types.BoolValue(true),
		OnBounced:            types.BoolValue(false),
		OnFailed:             types.BoolValue(false),
		OnReception:          types.BoolValue(false),
		OnTransientError:     types.BoolValue(false),
		OnSuppressed:         types.BoolValue(false),
		OnOpened:             types.BoolValue(false),
		OnClicked:            types.BoolValue(false),
		OnSuppressionCreated: types.BoolValue(false),
		OnDNSError:           types.BoolValue(false),
	}

	req, diags := webhookPlanToUpdateRequest(context.Background(), plan)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if req.Domains == nil {
		t.Fatal("expected Domains pointer for known empty list")
	}
	if len(*req.Domains) != 0 {
		t.Fatalf("expected empty domains slice, got %v", *req.Domains)
	}
}

func TestFlattenWebhookPreservesSecret(t *testing.T) {
	t.Parallel()

	hookID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	hook := &responses.Webhook{
		ID:        hookID,
		Name:      "delivery-events",
		URL:       "https://example.com/hook",
		Enabled:   true,
		Secret:    "",
		Scope:     "global",
		Domains:   []string{},
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}

	prior := types.StringValue("whsec-preserved")
	var out WebhookResourceModel
	diags := flattenWebhook(context.Background(), hook, prior, &out)
	if diags.HasError() {
		t.Fatalf("flattenWebhook diagnostics: %v", diags)
	}
	if out.Secret.ValueString() != "whsec-preserved" {
		t.Fatalf("secret = %q, want preserved", out.Secret.ValueString())
	}
	if out.Name.ValueString() != "delivery-events" {
		t.Fatalf("name = %q", out.Name.ValueString())
	}
}

func TestFlattenWebhookCreateSecretFromAPI(t *testing.T) {
	t.Parallel()

	hookID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	hook := &responses.Webhook{
		ID:        hookID,
		Name:      "delivery-events",
		URL:       "https://example.com/hook",
		Enabled:   true,
		Secret:    "whsec-from-api",
		Scope:     "global",
		Domains:   nil,
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}

	var out WebhookResourceModel
	diags := flattenWebhook(context.Background(), hook, types.StringNull(), &out)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out.Secret.ValueString() != "whsec-from-api" {
		t.Fatalf("secret = %q", out.Secret.ValueString())
	}
	if out.Domains.IsNull() || out.Domains.IsUnknown() {
		t.Fatal("domains must be known empty list when API returns nil")
	}
	if len(out.Domains.Elements()) != 0 {
		t.Fatalf("expected empty domains, got %d", len(out.Domains.Elements()))
	}
}
