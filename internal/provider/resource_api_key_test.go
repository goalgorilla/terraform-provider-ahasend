package provider

import (
	"context"
	"testing"
	"time"

	"github.com/AhaSend/ahasend-go/models/responses"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenAPIKey(t *testing.T) {
	t.Parallel()

	keyID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	accountID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	secret := "aha-sk-secret"
	createdAt := time.Unix(0, 0).UTC()
	key := &responses.APIKey{
		ID:          keyID,
		AccountID:   accountID,
		Label:       "terraform",
		PublicKey:   "aha-pk-public",
		SecretKey:   &secret,
		IPAllowList: []string{"203.0.113.0/24"},
		Scopes: []responses.APIKeyScope{
			{Scope: "domains:read"},
			{Scope: "domains:write"},
		},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	out := APIKeyResourceModel{SecretKey: types.StringValue(secret)}
	diags := flattenAPIKey(context.Background(), key, &out)
	if diags.HasError() {
		t.Fatalf("flattenAPIKey diagnostics: %v", diags)
	}
	if out.ID.ValueString() != keyID.String() {
		t.Fatalf("id = %q", out.ID.ValueString())
	}
	if out.Label.ValueString() != "terraform" {
		t.Fatalf("label = %q", out.Label.ValueString())
	}
	if out.SecretKey.ValueString() != secret {
		t.Fatalf("secret_key should be preserved by caller, got %q", out.SecretKey.ValueString())
	}
	if len(out.Scopes.Elements()) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(out.Scopes.Elements()))
	}
	if len(out.IPAllowList.Elements()) != 1 {
		t.Fatalf("expected 1 ip, got %d", len(out.IPAllowList.Elements()))
	}
	if out.CreatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("created_at = %q", out.CreatedAt.ValueString())
	}
	if out.UpdatedAt.ValueString() != createdAt.Format(time.RFC3339) {
		t.Fatalf("updated_at = %q", out.UpdatedAt.ValueString())
	}

	t.Run("nil IPAllowList", func(t *testing.T) {
		t.Parallel()
		nilKey := *key
		nilKey.IPAllowList = nil
		var nilOut APIKeyResourceModel
		d := flattenAPIKey(context.Background(), &nilKey, &nilOut)
		if d.HasError() {
			t.Fatalf("diagnostics: %v", d)
		}
		if nilOut.IPAllowList.IsNull() || nilOut.IPAllowList.IsUnknown() {
			t.Fatal("expected empty list, not null/unknown")
		}
		if len(nilOut.IPAllowList.Elements()) != 0 {
			t.Fatalf("expected empty list, got %d", len(nilOut.IPAllowList.Elements()))
		}
	})
}

func TestFlattenAPIKeyNormalizesBareIP(t *testing.T) {
	t.Parallel()

	keyID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	accountID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	secret := "aha-sk-secret"
	createdAt := time.Unix(0, 0).UTC()
	key := &responses.APIKey{
		ID:          keyID,
		AccountID:   accountID,
		Label:       "terraform",
		PublicKey:   "aha-pk-public",
		SecretKey:   &secret,
		IPAllowList: []string{"198.51.100.7"},
		Scopes: []responses.APIKeyScope{
			{Scope: "domains:read"},
		},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}

	var out APIKeyResourceModel
	diags := flattenAPIKey(context.Background(), key, &out)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	var ips []string
	diags = out.IPAllowList.ElementsAs(context.Background(), &ips, false)
	if diags.HasError() {
		t.Fatalf("elements: %v", diags)
	}
	if len(ips) != 1 || ips[0] != "198.51.100.7/32" {
		t.Fatalf("ip_allow_list = %v, want [198.51.100.7/32]", ips)
	}
}
