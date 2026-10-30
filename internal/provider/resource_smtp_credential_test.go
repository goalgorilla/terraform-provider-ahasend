package provider

import (
	"context"
	"testing"
	"time"

	"github.com/AhaSend/ahasend-go/models/responses"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenSMTPCredentialPreservesPassword(t *testing.T) {
	t.Parallel()

	credID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	cred := &responses.SMTPCredential{
		ID:        credID,
		Name:      "app-smtp",
		Username:  "smtp-user",
		Password:  "",
		Sandbox:   false,
		Scope:     "global",
		Domains:   []string{},
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}

	prior := SMTPCredentialResourceModel{
		AccountID: types.StringValue("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"),
		APIKey:    types.StringValue("aha-sk-child"),
		Password:  types.StringValue("smtp-password-once"),
	}
	var out SMTPCredentialResourceModel
	diags := flattenSMTPCredential(context.Background(), cred, prior, &out)
	if diags.HasError() {
		t.Fatalf("flattenSMTPCredential diagnostics: %v", diags)
	}
	if out.Password.ValueString() != "smtp-password-once" {
		t.Fatalf("password = %q, want preserved", out.Password.ValueString())
	}
	if out.Username.ValueString() != "smtp-user" {
		t.Fatalf("username = %q", out.Username.ValueString())
	}
	if out.APIKey.ValueString() != "aha-sk-child" {
		t.Fatalf("api_key = %q, want preserved", out.APIKey.ValueString())
	}
	if out.AccountID.ValueString() != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" {
		t.Fatalf("account_id = %q, want preserved", out.AccountID.ValueString())
	}
}

func TestFlattenSMTPCreatePasswordFromAPI(t *testing.T) {
	t.Parallel()

	credID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	cred := &responses.SMTPCredential{
		ID:        credID,
		Name:      "app-smtp",
		Username:  "smtp-user",
		Password:  "smtp-password-once",
		Sandbox:   false,
		Scope:     "global",
		Domains:   nil,
		CreatedAt: time.Unix(0, 0).UTC(),
		UpdatedAt: time.Unix(0, 0).UTC(),
	}

	prior := SMTPCredentialResourceModel{
		AccountID: types.StringValue("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"),
		APIKey:    types.StringValue("aha-sk-child"),
		Password:  types.StringNull(),
	}
	var out SMTPCredentialResourceModel
	diags := flattenSMTPCredential(context.Background(), cred, prior, &out)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out.Password.ValueString() != "smtp-password-once" {
		t.Fatalf("password = %q", out.Password.ValueString())
	}
	if out.Domains.IsNull() || out.Domains.IsUnknown() {
		t.Fatal("domains must be known empty list when API returns nil")
	}
	if out.APIKey.ValueString() != "aha-sk-child" {
		t.Fatalf("api_key = %q", out.APIKey.ValueString())
	}
}
