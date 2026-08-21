package provider

import (
	"context"
	"testing"
	"time"

	"github.com/AhaSend/ahasend-go/models/responses"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenSubAccountStatuses(t *testing.T) {
	t.Parallel()

	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	parent := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	base := responses.SubAccount{
		ID:              id,
		ParentAccountID: parent,
		Name:            "Customer",
		Website:         "customer.example.com",
		CreatedAt:       time.Unix(0, 0).UTC(),
		MonthlyCredit:   0,
	}

	tests := []struct {
		status    string
		suspended bool
	}{
		{status: "active", suspended: false},
		{status: "suspended", suspended: true},
		{status: "parent-suspended", suspended: true},
		{status: "deleted", suspended: false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			t.Parallel()
			sub := base
			sub.Status = tt.status
			var out SubAccountResourceModel
			flattenSubAccount(&sub, &out)
			if out.Suspended.ValueBool() != tt.suspended {
				t.Fatalf("suspended = %v, want %v for status %q", out.Suspended.ValueBool(), tt.suspended, tt.status)
			}
			if out.Status.ValueString() != tt.status {
				t.Fatalf("status = %q", out.Status.ValueString())
			}
		})
	}
}

func TestFlattenSubAccountNilLastActivity(t *testing.T) {
	t.Parallel()

	subID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	parentID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	sub := &responses.SubAccount{
		ID:              subID,
		ParentAccountID: parentID,
		Name:            "child",
		Website:         "https://example.com",
		Status:          "active",
		MonthlyCredit:   0,
		DomainCount:     1,
		MemberCount:     2,
		CreatedAt:       time.Unix(0, 0).UTC(),
		LastActivityAt:  nil,
	}

	var out SubAccountResourceModel
	flattenSubAccount(sub, &out)
	if !out.LastActivityAt.IsNull() {
		t.Fatalf("last_activity_at = %#v, want null", out.LastActivityAt)
	}
	if out.Suspended.ValueBool() {
		t.Fatal("expected suspended false for active")
	}
	if out.DomainCount.ValueInt64() != 1 || out.MemberCount.ValueInt64() != 2 {
		t.Fatalf("counts domain=%d member=%d", out.DomainCount.ValueInt64(), out.MemberCount.ValueInt64())
	}
}

func TestFlattenSubAccountAPIKey(t *testing.T) {
	t.Parallel()

	keyID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	key := &responses.APIKey{
		ID:          keyID,
		Label:       "bootstrap",
		PublicKey:   "aha-pk",
		IPAllowList: nil,
		Scopes:      []responses.APIKeyScope{{Scope: "domains:read"}},
		CreatedAt:   time.Unix(0, 0).UTC(),
		UpdatedAt:   time.Unix(0, 0).UTC(),
	}
	out := SubAccountAPIKeyResourceModel{
		SubAccountID: types.StringValue("ffffffff-ffff-ffff-ffff-ffffffffffff"),
	}
	diags := flattenSubAccountAPIKey(context.Background(), key, &out)
	if diags.HasError() {
		t.Fatalf("diagnostics: %v", diags)
	}
	if out.ID.ValueString() != keyID.String() {
		t.Fatalf("id = %q", out.ID.ValueString())
	}
	if out.SubAccountID.ValueString() != "ffffffff-ffff-ffff-ffff-ffffffffffff" {
		t.Fatal("sub_account_id should be preserved by caller")
	}
	if len(out.IPAllowList.Elements()) != 0 {
		t.Fatalf("expected empty ip allow list, got %d", len(out.IPAllowList.Elements()))
	}
}
