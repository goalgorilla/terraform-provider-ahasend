package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/AhaSend/ahasend-go/api"
	"github.com/AhaSend/ahasend-go/models/requests"
	"github.com/AhaSend/ahasend-go/models/responses"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// uuidPattern validates UUID strings for plan-time attribute validation.
var uuidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

var (
	_ resource.Resource                 = &SubAccountAPIKeyResource{}
	_ resource.ResourceWithConfigure    = &SubAccountAPIKeyResource{}
	_ resource.ResourceWithImportState  = &SubAccountAPIKeyResource{}
	_ resource.ResourceWithUpgradeState = &SubAccountAPIKeyResource{}
)

// SubAccountAPIKeyResource manages an API key owned by an AhaSend sub account.
type SubAccountAPIKeyResource struct {
	client *ahasendClient
}

// SubAccountAPIKeyResourceModel is the Terraform state model for ahasend_sub_account_api_key.
type SubAccountAPIKeyResourceModel struct {
	ID           types.String `tfsdk:"id"`
	SubAccountID types.String `tfsdk:"sub_account_id"`
	Label        types.String `tfsdk:"label"`
	Scopes       types.Set    `tfsdk:"scopes"`
	IPAllowList  types.List   `tfsdk:"ip_allow_list"`
	PublicKey    types.String `tfsdk:"public_key"`
	SecretKey    types.String `tfsdk:"secret_key"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

// NewSubAccountAPIKeyResource returns a new ahasend_sub_account_api_key resource.
func NewSubAccountAPIKeyResource() resource.Resource {
	return &SubAccountAPIKeyResource{}
}

// Metadata sets the resource type name to ahasend_sub_account_api_key.
func (r *SubAccountAPIKeyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sub_account_api_key"
}

// Schema defines the ahasend_sub_account_api_key resource attributes.
func (r *SubAccountAPIKeyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		MarkdownDescription: "Manages an API key owned by an AhaSend Platform Partner sub account. " +
			"Created with the **parent** account credentials (`sub-account-api-keys:write`). " +
			"The one-time `secret_key` is returned only on create (and exact idempotent replay within five minutes); " +
			"use it with a provider alias to manage child-account resources such as domains.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "API key UUID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"sub_account_id": schema.StringAttribute{
				MarkdownDescription: "Sub account UUID that owns this API key. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					stringvalidator.RegexMatches(
						uuidPattern,
						"must be a valid UUID",
					),
				},
			},
			"label": schema.StringAttribute{
				MarkdownDescription: "Human-readable label for the API key. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"scopes": schema.SetAttribute{
				MarkdownDescription: "Scopes granted to this API key (for example `domains:read`, `domains:write`). Order is insignificant. Changing this forces a new resource.",
				Required:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
					setvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"ip_allow_list": schema.ListAttribute{
				MarkdownDescription: "Optional source IPs (CIDR or bare address) allowed to authenticate with this key. " +
					"Bare IPv4/IPv6 addresses are stored as `/32` or `/128` (AhaSend canonical form); Terraform normalizes config to match. " +
					"Omit or leave empty to allow any IP. Changing this forces a new resource.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
					listplanmodifier.UseStateForUnknown(),
					normalizeIPAllowListPlanModifier(),
				},
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "Public portion of the API key (safe to display).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "One-time secret key returned only on create. Store immediately; not returned by subsequent reads. " +
					"Exact idempotent create retries within five minutes may return the same secret.",
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 creation timestamp.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 last update timestamp.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// UpgradeState migrates version-0 state (scopes as List) to version 1 (scopes as Set).
func (r *SubAccountAPIKeyResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	prior := subAccountAPIKeySchemaV0()
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &prior,
			StateUpgrader: upgradeSubAccountAPIKeyStateV0toV1,
		},
	}
}

// Configure stores the provider API client on the resource.
func (r *SubAccountAPIKeyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*ahasendClient)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *ahasendClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = client
}

// Create creates a sub-account API key with an idempotent request and stores the one-time secret_key.
func (r *SubAccountAPIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SubAccountAPIKeyResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	subAccountID, diags := parseUUIDAttr(plan.SubAccountID, "sub_account_id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopes, diags := setToStringSlice(ctx, plan.Scopes)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := requests.CreateAPIKeyRequest{
		Label:  plan.Label.ValueString(),
		Scopes: scopes,
	}

	if !plan.IPAllowList.IsNull() && !plan.IPAllowList.IsUnknown() {
		ips, ipDiags := listToStringSlice(ctx, plan.IPAllowList)
		resp.Diagnostics.Append(ipDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.IPAllowList = normalizeIPAllowList(ips)
	}

	token, tokenDiags := ensureIdempotencyToken(ctx, resp.Private)
	resp.Diagnostics.Append(tokenDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	idempotencyKey := subAccountAPIKeyIdempotencyKey(token)

	created, _, err := r.client.api.SubAccountsAPI.CreateSubAccountAPIKey(
		ctx,
		r.client.accountID,
		subAccountID,
		createReq,
		api.WithIdempotencyKey(idempotencyKey),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AhaSend sub account API key", formatAPIError(err))
		return
	}

	if created.SecretKey == nil || *created.SecretKey == "" {
		resp.Diagnostics.AddError(
			"Missing API key secret on create",
			"AhaSend did not return secret_key for the new sub-account API key. The key may exist remotely; import or delete it before retrying.",
		)
		return
	}

	state := SubAccountAPIKeyResourceModel{
		SubAccountID: plan.SubAccountID,
		SecretKey:    types.StringValue(*created.SecretKey),
	}

	resp.Diagnostics.Append(flattenSubAccountAPIKey(ctx, created, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes key state and preserves secret_key from prior state.
func (r *SubAccountAPIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SubAccountAPIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keyID, diags := parseUUIDAttr(state.ID, "id")
	resp.Diagnostics.Append(diags...)
	subAccountID, diags := parseUUIDAttr(state.SubAccountID, "sub_account_id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	key, _, err := r.client.api.SubAccountsAPI.GetSubAccountAPIKey(
		ctx,
		r.client.accountID,
		subAccountID,
		keyID,
	)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading AhaSend sub account API key", formatAPIError(err))
		return
	}

	// GET omits secret_key; preserve any value already in state.
	priorSecret := state.SecretKey
	newState := SubAccountAPIKeyResourceModel{
		SubAccountID: state.SubAccountID,
		SecretKey:    priorSecret,
	}
	resp.Diagnostics.Append(flattenSubAccountAPIKey(ctx, key, &newState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update always errors; configuration changes force replacement.
func (r *SubAccountAPIKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable attributes ForceNew via RequiresReplace; Update should not run.
	resp.Diagnostics.AddError(
		"Unexpected Update",
		"ahasend_sub_account_api_key does not support in-place updates. All configuration changes force replacement.",
	)
}

// Delete removes the sub-account API key; missing resources are treated as success.
func (r *SubAccountAPIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SubAccountAPIKeyResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keyID, diags := parseUUIDAttr(state.ID, "id")
	resp.Diagnostics.Append(diags...)
	subAccountID, diags := parseUUIDAttr(state.SubAccountID, "sub_account_id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.api.SubAccountsAPI.DeleteSubAccountAPIKey(
		ctx,
		r.client.accountID,
		subAccountID,
		keyID,
	)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting AhaSend sub account API key", formatAPIError(err))
		return
	}
}

// ImportState imports using SUB_ACCOUNT_ID/KEY_ID.
func (r *SubAccountAPIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	parts := strings.Split(id, "/")
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import ID of the form SUB_ACCOUNT_ID/KEY_ID, got: %q", req.ID),
		)
		return
	}
	if _, err := uuid.Parse(parts[0]); err != nil {
		resp.Diagnostics.AddError("Invalid sub_account_id in import ID", err.Error())
		return
	}
	if _, err := uuid.Parse(parts[1]); err != nil {
		resp.Diagnostics.AddError("Invalid key id in import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("sub_account_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}

// flattenSubAccountAPIKey copies API key fields into state; callers must set SubAccountID and SecretKey.
func flattenSubAccountAPIKey(ctx context.Context, key *responses.APIKey, out *SubAccountAPIKeyResourceModel) diag.Diagnostics {
	common, diags := flattenAPIKeyCommon(ctx, key)
	out.ID = common.ID
	out.Label = common.Label
	out.PublicKey = common.PublicKey
	out.CreatedAt = common.CreatedAt
	out.UpdatedAt = common.UpdatedAt
	out.Scopes = common.Scopes
	out.IPAllowList = common.IPAllowList
	return diags
}
