package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// validateScopeDomains enforces the AhaSend scope/domains contract:
// scope "scoped" requires a non-empty domains list; scope "global" rejects configured domains.
// Unknown values (including unknown list *elements* from interpolated references) are skipped
// so validation can run again once values are known.
func validateScopeDomains(ctx context.Context, scope types.String, domains types.List) diag.Diagnostics {
	var diags diag.Diagnostics
	if scope.IsUnknown() || scope.IsNull() || domains.IsUnknown() {
		return diags
	}
	if listContainsUnknownElements(domains) {
		return diags
	}

	domainList, d := listToStringSlice(ctx, domains)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	switch scope.ValueString() {
	case "scoped":
		if len(domainList) == 0 {
			diags.AddAttributeError(
				path.Root("domains"),
				"Missing Domains for Scoped Resource",
				"When scope is \"scoped\", domains must be a non-empty list of domain names.",
			)
		}
	case "global":
		if len(domainList) > 0 {
			diags.AddAttributeError(
				path.Root("domains"),
				"Domains Not Allowed for Global Scope",
				"When scope is \"global\", omit domains or set domains to an empty list.",
			)
		}
	}
	return diags
}

// listContainsUnknownElements reports whether a known list still has unknown elements
// (for example domains = [ahasend_domain.x.domain] before the domain is applied).
func listContainsUnknownElements(list types.List) bool {
	if list.IsNull() || list.IsUnknown() {
		return false
	}
	for _, elem := range list.Elements() {
		if elem.IsUnknown() {
			return true
		}
	}
	return false
}
