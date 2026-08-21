package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AhaSend/ahasend-go"
	"github.com/AhaSend/ahasend-go/api"
	"github.com/AhaSend/ahasend-go/models/requests"
	"github.com/AhaSend/ahasend-go/models/responses"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &SubAccountResource{}
	_ resource.ResourceWithConfigure   = &SubAccountResource{}
	_ resource.ResourceWithImportState = &SubAccountResource{}
)

// SubAccountResource manages an AhaSend Platform Partner sub account.
type SubAccountResource struct {
	client *ahasendClient
}

// SubAccountResourceModel is the Terraform state model for ahasend_sub_account.
type SubAccountResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Website          types.String `tfsdk:"website"`
	MonthlyCredit    types.Int64  `tfsdk:"monthly_credit"`
	Suspended        types.Bool   `tfsdk:"suspended"`
	SuspensionReason types.String `tfsdk:"suspension_reason"`
	Status           types.String `tfsdk:"status"`
	ParentAccountID  types.String `tfsdk:"parent_account_id"`
	CreatedAt        types.String `tfsdk:"created_at"`
	DomainCount      types.Int64  `tfsdk:"domain_count"`
	MemberCount      types.Int64  `tfsdk:"member_count"`
	LastActivityAt   types.String `tfsdk:"last_activity_at"`
}

// NewSubAccountResource returns a new ahasend_sub_account resource.
func NewSubAccountResource() resource.Resource {
	return &SubAccountResource{}
}

// Metadata sets the resource type name to ahasend_sub_account.
func (r *SubAccountResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sub_account"
}

// Schema defines the ahasend_sub_account resource attributes.
func (r *SubAccountResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AhaSend Platform Partner sub account under the configured parent account. " +
			"Requires parent API key scopes `sub-accounts:read|write|delete` and `sub-accounts:suspend` when using `suspended`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Sub account UUID (child account ID).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name for the sub account.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"website": schema.StringAttribute{
				MarkdownDescription: "Account website domain (FQDN).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"monthly_credit": schema.Int64Attribute{
				MarkdownDescription: "Optional monthly sending cap. `0` means no cap.",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(0),
				Validators: []validator.Int64{
					int64validator.Between(0, 1000000000),
				},
			},
			"suspended": schema.BoolAttribute{
				MarkdownDescription: "When true, the sub account is suspended via the suspend endpoint. When false, unsuspend is called if previously suspended.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"suspension_reason": schema.StringAttribute{
				MarkdownDescription: "Reason sent to the suspend endpoint when `suspended` is true. Defaults to `Suspended via Terraform`.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("Suspended via Terraform"),
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "Current sub account status (`active`, `suspended`, `parent-suspended`, or `deleted`).",
				Computed:            true,
			},
			"parent_account_id": schema.StringAttribute{
				MarkdownDescription: "Parent account UUID.",
				Computed:            true,
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
			"domain_count": schema.Int64Attribute{
				MarkdownDescription: "Number of domains owned by the sub account.",
				Computed:            true,
			},
			"member_count": schema.Int64Attribute{
				MarkdownDescription: "Number of direct members on the sub account.",
				Computed:            true,
			},
			"last_activity_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 timestamp of last recorded email activity, if any.",
				Computed:            true,
			},
		},
	}
}

// Configure stores the provider API client on the resource.
func (r *SubAccountResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates a sub account and optionally suspends it; state is saved before suspend so destroy remains possible.
func (r *SubAccountResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SubAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := requests.CreateSubAccountRequest{
		Name:          plan.Name.ValueString(),
		Website:       plan.Website.ValueString(),
		MonthlyCredit: ahasend.Int64(plan.MonthlyCredit.ValueInt64()),
	}

	token, tokenDiags := ensureIdempotencyToken(ctx, resp.Private)
	resp.Diagnostics.Append(tokenDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	idempotencyKey := subAccountIdempotencyKey(token)
	created, _, err := r.client.api.SubAccountsAPI.CreateSubAccount(
		ctx,
		r.client.accountID,
		createReq,
		api.WithIdempotencyKey(idempotencyKey),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AhaSend sub account", formatAPIError(err))
		return
	}

	if created.Status == "deleted" {
		resp.Diagnostics.AddError(
			"AhaSend returned a deleted sub account",
			fmt.Sprintf(
				"Create returned sub account %s with status deleted (likely an idempotent replay of a soft-deleted resource). "+
					"Do not reuse a deterministic Idempotency-Key across destroy/recreate. Retry apply with a fresh create attempt.",
				created.ID.String(),
			),
		)
		return
	}

	// Persist create result before optional suspend so destroy/update remain possible if suspend fails.
	flattenSubAccount(created, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.Suspended.ValueBool() {
		reason := plan.SuspensionReason.ValueString()
		if reason == "" {
			reason = "Suspended via Terraform"
		}
		updated, _, suspendErr := r.client.api.SubAccountsAPI.SuspendSubAccount(
			ctx,
			r.client.accountID,
			created.ID,
			requests.SuspendSubAccountRequest{Reason: reason},
		)
		if suspendErr != nil {
			resp.Diagnostics.AddError("Error suspending AhaSend sub account after create", formatAPIError(suspendErr))
			return
		}
		plan.SuspensionReason = types.StringValue(reason)
		flattenSubAccount(updated, &plan)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

// Read refreshes sub account state; deleted statuses remove the resource from state.
func (r *SubAccountResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SubAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := parseUUIDAttr(state.ID, "id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	sub, _, err := r.client.api.SubAccountsAPI.GetSubAccount(ctx, r.client.accountID, id)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading AhaSend sub account", formatAPIError(err))
		return
	}

	if sub.Status == "deleted" {
		resp.State.RemoveResource(ctx)
		return
	}

	flattenSubAccount(sub, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update applies metadata changes and reconciles suspension (parent-suspended is read-only).
func (r *SubAccountResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SubAccountResourceModel
	var state SubAccountResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := parseUUIDAttr(state.ID, "id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	current, _, err := r.client.api.SubAccountsAPI.GetSubAccount(ctx, r.client.accountID, id)
	if err != nil {
		resp.Diagnostics.AddError("Error reading AhaSend sub account before update", formatAPIError(err))
		return
	}

	updateReq := requests.UpdateSubAccountRequest{}
	needsUpdate := false
	if !plan.Name.Equal(state.Name) {
		needsUpdate = true
		updateReq.Name = stringPtr(plan.Name.ValueString())
	}
	if !plan.Website.Equal(state.Website) {
		needsUpdate = true
		updateReq.Website = stringPtr(plan.Website.ValueString())
	}
	if !plan.MonthlyCredit.Equal(state.MonthlyCredit) {
		needsUpdate = true
		updateReq.MonthlyCredit = ahasend.Int64(plan.MonthlyCredit.ValueInt64())
	}

	if needsUpdate {
		updated, _, updateErr := r.client.api.SubAccountsAPI.UpdateSubAccount(ctx, r.client.accountID, id, updateReq)
		if updateErr != nil {
			resp.Diagnostics.AddError("Error updating AhaSend sub account", formatAPIError(updateErr))
			return
		}
		current = updated
	}

	wantSuspended := plan.Suspended.ValueBool()
	isSuspended := subAccountIsSuspended(current.Status)
	parentSuspended := current.Status == "parent-suspended"

	if parentSuspended {
		// Parent-driven suspension cannot be toggled from the child resource.
		if !wantSuspended {
			resp.Diagnostics.AddWarning(
				"Sub account is parent-suspended",
				"AhaSend reports status parent-suspended. Terraform will not call unsuspend; suspended remains true until the parent restores the account.",
			)
		}
	} else if wantSuspended && !isSuspended {
		reason := plan.SuspensionReason.ValueString()
		if reason == "" {
			reason = "Suspended via Terraform"
		}
		updated, _, suspendErr := r.client.api.SubAccountsAPI.SuspendSubAccount(
			ctx,
			r.client.accountID,
			id,
			requests.SuspendSubAccountRequest{Reason: reason},
		)
		if suspendErr != nil {
			resp.Diagnostics.AddError("Error suspending AhaSend sub account", formatAPIError(suspendErr))
			return
		}
		current = updated
		plan.SuspensionReason = types.StringValue(reason)
	} else if !wantSuspended && isSuspended {
		updated, _, unsuspendErr := r.client.api.SubAccountsAPI.UnsuspendSubAccount(ctx, r.client.accountID, id)
		if unsuspendErr != nil {
			resp.Diagnostics.AddError("Error unsuspending AhaSend sub account", formatAPIError(unsuspendErr))
			return
		}
		current = updated
	}

	flattenSubAccount(current, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete removes the sub account; missing resources are treated as success.
func (r *SubAccountResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SubAccountResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := parseUUIDAttr(state.ID, "id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.api.SubAccountsAPI.DeleteSubAccount(ctx, r.client.accountID, id)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting AhaSend sub account", formatAPIError(err))
		return
	}
}

// ImportState imports by sub account UUID.
func (r *SubAccountResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	if _, err := uuid.Parse(id); err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected sub account UUID, got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("suspended"), false)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("monthly_credit"), int64(0))...)
}

// flattenSubAccount copies sub account fields into state; suspended includes parent-suspended.
func flattenSubAccount(sub *responses.SubAccount, out *SubAccountResourceModel) {
	out.ID = types.StringValue(sub.ID.String())
	out.Name = types.StringValue(sub.Name)
	out.Website = types.StringValue(sub.Website)
	out.Status = types.StringValue(sub.Status)
	out.ParentAccountID = types.StringValue(sub.ParentAccountID.String())
	out.CreatedAt = types.StringValue(sub.CreatedAt.UTC().Format(time.RFC3339))
	out.MonthlyCredit = types.Int64Value(sub.MonthlyCredit)
	out.DomainCount = types.Int64Value(sub.DomainCount)
	out.MemberCount = types.Int64Value(sub.MemberCount)
	out.Suspended = types.BoolValue(subAccountIsSuspended(sub.Status))
	if sub.LastActivityAt != nil {
		out.LastActivityAt = types.StringValue(sub.LastActivityAt.UTC().Format(time.RFC3339))
	} else {
		out.LastActivityAt = types.StringNull()
	}
}

// subAccountIsSuspended reports whether status is suspended or parent-suspended.
func subAccountIsSuspended(status string) bool {
	return status == "suspended" || status == "parent-suspended"
}
