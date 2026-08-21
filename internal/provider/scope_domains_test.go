package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateScopeDomains(t *testing.T) {
	t.Parallel()

	t.Run("scoped requires domains", func(t *testing.T) {
		t.Parallel()
		diags := validateScopeDomains(context.Background(), types.StringValue("scoped"), types.ListNull(types.StringType))
		if !diags.HasError() {
			t.Fatal("expected error")
		}
	})

	t.Run("global rejects domains", func(t *testing.T) {
		t.Parallel()
		domains, d := types.ListValueFrom(context.Background(), types.StringType, []string{"mail.example.com"})
		if d.HasError() {
			t.Fatalf("%v", d)
		}
		diags := validateScopeDomains(context.Background(), types.StringValue("global"), domains)
		if !diags.HasError() {
			t.Fatal("expected error")
		}
	})

	t.Run("scoped with domains ok", func(t *testing.T) {
		t.Parallel()
		domains, d := types.ListValueFrom(context.Background(), types.StringType, []string{"mail.example.com"})
		if d.HasError() {
			t.Fatalf("%v", d)
		}
		diags := validateScopeDomains(context.Background(), types.StringValue("scoped"), domains)
		if diags.HasError() {
			t.Fatalf("unexpected: %v", diags)
		}
	})

	t.Run("skips when domains list element is unknown", func(t *testing.T) {
		t.Parallel()
		// Config like: domains = [ahasend_domain.sending.domain] before apply.
		domains, d := types.ListValue(types.StringType, []attr.Value{types.StringUnknown()})
		if d.HasError() {
			t.Fatalf("%v", d)
		}
		if domains.IsUnknown() {
			t.Fatal("list itself should be known; only the element is unknown")
		}
		diags := validateScopeDomains(context.Background(), types.StringValue("scoped"), domains)
		if diags.HasError() {
			t.Fatalf("ValidateConfig must tolerate unknown domain elements, got: %v", diags)
		}
	})
}
