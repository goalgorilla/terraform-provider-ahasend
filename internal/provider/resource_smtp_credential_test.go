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

	prior := types.StringValue("smtp-password-once")
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

	var out SMTPCredentialResourceModel
	diags := flattenSMTPCredential(context.Background(), cred, types.StringNull(), &out)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out.Password.ValueString() != "smtp-password-once" {
		t.Fatalf("password = %q", out.Password.ValueString())
	}
	if out.Domains.IsNull() || out.Domains.IsUnknown() {
		t.Fatal("domains must be known empty list when API returns nil")
	}
}
