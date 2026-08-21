package provider

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// parseUUIDAttr parses a Terraform string attribute as a UUID with attribute-path diagnostics.
func parseUUIDAttr(v types.String, attrName string) (uuid.UUID, diag.Diagnostics) {
	var diags diag.Diagnostics
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		diags.AddAttributeError(
			path.Root(attrName),
			"Missing "+attrName,
			fmt.Sprintf("%s must be a valid UUID.", attrName),
		)
		return uuid.Nil, diags
	}
	id, err := uuid.Parse(v.ValueString())
	if err != nil {
		diags.AddAttributeError(
			path.Root(attrName),
			"Invalid "+attrName,
			err.Error(),
		)
		return uuid.Nil, diags
	}
	return id, diags
}

// listToStringSlice converts a Terraform list of strings to []string; null/unknown yields nil.
// Unknown or non-string elements produce diagnostics (callers that run at plan time must
// skip via list.IsUnknown / listContainsUnknownElements first).
func listToStringSlice(ctx context.Context, list types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	elems := list.Elements()
	out := make([]string, 0, len(elems))
	for i, elem := range elems {
		sv, ok := elem.(types.String)
		if !ok {
			diags.AddError(
				"Unexpected List Element Type",
				fmt.Sprintf("expected string at index %d, got %T", i, elem),
			)
			return nil, diags
		}
		if sv.IsNull() || sv.IsUnknown() {
			diags.AddError(
				"Unknown List Element",
				fmt.Sprintf("list element at index %d is unknown; cannot convert to string until it is known", i),
			)
			return nil, diags
		}
		out = append(out, sv.ValueString())
	}
	return out, diags
}

// setToStringSlice converts a Terraform set of strings to []string; null/unknown yields nil.
func setToStringSlice(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	if set.IsNull() || set.IsUnknown() {
		return nil, diags
	}
	elems := set.Elements()
	out := make([]string, 0, len(elems))
	for i, elem := range elems {
		sv, ok := elem.(types.String)
		if !ok {
			diags.AddError(
				"Unexpected Set Element Type",
				fmt.Sprintf("expected string at index %d, got %T", i, elem),
			)
			return nil, diags
		}
		if sv.IsNull() || sv.IsUnknown() {
			diags.AddError(
				"Unknown Set Element",
				fmt.Sprintf("set element at index %d is unknown; cannot convert to string until it is known", i),
			)
			return nil, diags
		}
		out = append(out, sv.ValueString())
	}
	return out, diags
}
