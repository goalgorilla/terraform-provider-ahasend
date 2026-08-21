package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// apiKeyModelV0 is the version-0 state shape when scopes was a List.
type apiKeyModelV0 struct {
	ID          types.String `tfsdk:"id"`
	Label       types.String `tfsdk:"label"`
	Scopes      types.List   `tfsdk:"scopes"`
	IPAllowList types.List   `tfsdk:"ip_allow_list"`
	PublicKey   types.String `tfsdk:"public_key"`
	SecretKey   types.String `tfsdk:"secret_key"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

// subAccountAPIKeyModelV0 is the version-0 state shape when scopes was a List.
type subAccountAPIKeyModelV0 struct {
	ID           types.String `tfsdk:"id"`
	SubAccountID types.String `tfsdk:"sub_account_id"`
	Label        types.String `tfsdk:"label"`
	Scopes       types.List   `tfsdk:"scopes"`
	IPAllowList  types.List   `tfsdk:"ip_allow_list"`
	PublicKey    types.String `tfsdk:"public_key"`
	SecretKey    types.String `tfsdk:"secret_key"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

// apiKeySchemaV0 is the prior schema for UpgradeState (scopes as List).
func apiKeySchemaV0() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true},
			"label":         schema.StringAttribute{Required: true},
			"scopes":        schema.ListAttribute{Required: true, ElementType: types.StringType},
			"ip_allow_list": schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			"public_key":    schema.StringAttribute{Computed: true},
			"secret_key":    schema.StringAttribute{Computed: true, Sensitive: true},
			"created_at":    schema.StringAttribute{Computed: true},
			"updated_at":    schema.StringAttribute{Computed: true},
		},
	}
}

// subAccountAPIKeySchemaV0 is the prior schema for UpgradeState (scopes as List).
func subAccountAPIKeySchemaV0() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true},
			"sub_account_id": schema.StringAttribute{Required: true},
			"label":          schema.StringAttribute{Required: true},
			"scopes":         schema.ListAttribute{Required: true, ElementType: types.StringType},
			"ip_allow_list":  schema.ListAttribute{Optional: true, Computed: true, ElementType: types.StringType},
			"public_key":     schema.StringAttribute{Computed: true},
			"secret_key":     schema.StringAttribute{Computed: true, Sensitive: true},
			"created_at":     schema.StringAttribute{Computed: true},
			"updated_at":     schema.StringAttribute{Computed: true},
		},
	}
}

// scopesListToSet converts a prior scopes List into a Set for schema v1.
func scopesListToSet(ctx context.Context, list types.List) (types.Set, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() {
		return types.SetNull(types.StringType), diags
	}
	if list.IsUnknown() {
		return types.SetUnknown(types.StringType), diags
	}
	var scopes []string
	diags.Append(list.ElementsAs(ctx, &scopes, false)...)
	if diags.HasError() {
		return types.SetNull(types.StringType), diags
	}
	set, setDiags := types.SetValueFrom(ctx, types.StringType, scopes)
	diags.Append(setDiags...)
	return set, diags
}

// upgradeAPIKeyStateV0toV1 migrates scopes from List to Set.
func upgradeAPIKeyStateV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior apiKeyModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes, diags := scopesListToSet(ctx, prior.Scopes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	upgraded := APIKeyResourceModel{
		ID:          prior.ID,
		Label:       prior.Label,
		Scopes:      scopes,
		IPAllowList: prior.IPAllowList,
		PublicKey:   prior.PublicKey,
		SecretKey:   prior.SecretKey,
		CreatedAt:   prior.CreatedAt,
		UpdatedAt:   prior.UpdatedAt,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
}

// upgradeSubAccountAPIKeyStateV0toV1 migrates scopes from List to Set.
func upgradeSubAccountAPIKeyStateV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var prior subAccountAPIKeyModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes, diags := scopesListToSet(ctx, prior.Scopes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	upgraded := SubAccountAPIKeyResourceModel{
		ID:           prior.ID,
		SubAccountID: prior.SubAccountID,
		Label:        prior.Label,
		Scopes:       scopes,
		IPAllowList:  prior.IPAllowList,
		PublicKey:    prior.PublicKey,
		SecretKey:    prior.SecretKey,
		CreatedAt:    prior.CreatedAt,
		UpdatedAt:    prior.UpdatedAt,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &upgraded)...)
}
