package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestScopesListToSet(t *testing.T) {
	t.Parallel()

	list, diags := types.ListValueFrom(context.Background(), types.StringType, []string{"domains:write", "domains:read"})
	if diags.HasError() {
		t.Fatalf("list: %v", diags)
	}
	set, diags := scopesListToSet(context.Background(), list)
	if diags.HasError() {
		t.Fatalf("scopesListToSet: %v", diags)
	}
	if set.IsNull() || set.IsUnknown() {
		t.Fatal("expected known set")
	}
	if len(set.Elements()) != 2 {
		t.Fatalf("expected 2 scopes, got %d", len(set.Elements()))
	}

	nullSet, diags := scopesListToSet(context.Background(), types.ListNull(types.StringType))
	if diags.HasError() {
		t.Fatalf("null: %v", diags)
	}
	if !nullSet.IsNull() {
		t.Fatal("expected null set from null list")
	}
}
