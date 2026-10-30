package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AhaSend/ahasend-go/api"
	"github.com/AhaSend/ahasend-go/models/requests"
	"github.com/AhaSend/ahasend-go/models/responses"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &SMTPCredentialResource{}
	_ resource.ResourceWithConfigure      = &SMTPCredentialResource{}
	_ resource.ResourceWithImportState    = &SMTPCredentialResource{}
	_ resource.ResourceWithValidateConfig = &SMTPCredentialResource{}
)

// SMTPCredentialResource manages an AhaSend SMTP credential.
type SMTPCredentialResource struct {
	client *ahasendClient
}

// SMTPCredentialResourceModel is the Terraform state model for ahasend_smtp_credential.
type SMTPCredentialResourceModel struct {
	ID        types.String `tfsdk:"id"`
	AccountID types.String `tfsdk:"account_id"`
	APIKey    types.String `tfsdk:"api_key"`
	Name      types.String `tfsdk:"name"`
	Scope     types.String `tfsdk:"scope"`
	Sandbox   types.Bool   `tfsdk:"sandbox"`
	Domains   types.List   `tfsdk:"domains"`
	Username  types.String `tfsdk:"username"`
	Password  types.String `tfsdk:"password"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// NewSMTPCredentialResource returns a new ahasend_smtp_credential resource.
func NewSMTPCredentialResource() resource.Resource {
	return &SMTPCredentialResource{}
}

// Metadata sets the resource type name to ahasend_smtp_credential.
func (r *SMTPCredentialResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_smtp_credential"
}

// Schema defines the ahasend_smtp_credential resource attributes.
func (r *SMTPCredentialResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AhaSend SMTP credential. There is no API update endpoint; changing " +
			"`name`, `scope`, `sandbox`, `domains`, or `account_id` forces a new resource. " +
			"The one-time `password` is returned on create and preserved in state. " +
			"For Platform Partner sub-account credentials in the same root module, set `account_id` to the child " +
			"account and `api_key` to that child's API key (for example from `ahasend_sub_account_api_key`).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "SMTP credential UUID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_id": schema.StringAttribute{
				MarkdownDescription: "Account UUID that owns the SMTP credential. Defaults to the provider `account_id`. " +
					"Set this to a sub-account UUID when managing a child credential (with a matching child `api_key`).",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "Optional Bearer token for this resource's API calls. " +
					"When set, overrides the provider `api_key` so a child sub-account key can manage " +
					"that child's SMTP credential in the same apply as `ahasend_sub_account_api_key`. " +
					"Stored sensitive in state (required for Read/Delete; write-only is not available on Read). " +
					"Omit to use the provider credentials.",
				Optional:  true,
				Sensitive: true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Unique credential name within the account. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "Credential scope: `global` or `scoped` (requires `domains`). Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("global", "scoped"),
				},
			},
			"sandbox": schema.BoolAttribute{
				MarkdownDescription: "When true, the credential operates in sandbox mode. Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"domains": schema.ListAttribute{
				MarkdownDescription: "Domain names for a `scoped` credential. Required when `scope` is `scoped`; must be omitted or empty when `scope` is `global`. Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
					listplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "SMTP username.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "One-time SMTP password returned only on create. Preserved in state; omitted from later reads.",
				Computed:            true,
				Sensitive:           true,
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

// ValidateConfig enforces the scope/domains contract before plan/apply.
func (r *SMTPCredentialResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data SMTPCredentialResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateScopeDomains(ctx, data.Scope, data.Domains)...)
}

// Configure stores the provider API client on the resource.
func (r *SMTPCredentialResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates an SMTP credential with an idempotent request and stores the one-time password.
func (r *SMTPCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SMTPCredentialResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, diags := r.resolveAccountID(plan.AccountID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := requests.CreateSMTPCredentialRequest{
		Name:    plan.Name.ValueString(),
		Scope:   plan.Scope.ValueString(),
		Sandbox: plan.Sandbox.ValueBool(),
	}
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		domains, domainDiags := listToStringSlice(ctx, plan.Domains)
		resp.Diagnostics.Append(domainDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Domains = domains
	}

	token, tokenDiags := ensureIdempotencyToken(ctx, resp.Private)
	resp.Diagnostics.Append(tokenDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	idempotencyKey := smtpIdempotencyKey(token)
	opts := append([]api.RequestOption{api.WithIdempotencyKey(idempotencyKey)}, resourceAPIKeyOpts(plan.APIKey)...)
	created, _, err := r.client.api.SMTPCredentialsAPI.CreateSMTPCredential(
		ctx,
		accountID,
		createReq,
		opts...,
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AhaSend SMTP credential", formatAPIError(err))
		return
	}

	state := SMTPCredentialResourceModel{}
	resp.Diagnostics.Append(flattenSMTPCredential(ctx, created, plan, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.AccountID.IsNull() || state.AccountID.IsUnknown() || state.AccountID.ValueString() == "" {
		state.AccountID = types.StringValue(accountID.String())
	}
	if state.Password.IsNull() || state.Password.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing SMTP password on create",
			"AhaSend did not return a password for the new SMTP credential. The credential may exist remotely; import or delete it before retrying.",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes credential state and preserves password from prior state when omitted by the API.
func (r *SMTPCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SMTPCredentialResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := parseUUIDAttr(state.ID, "id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, accountDiags := r.resolveAccountID(state.AccountID)
	resp.Diagnostics.Append(accountDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	cred, _, err := r.client.api.SMTPCredentialsAPI.GetSMTPCredential(
		ctx,
		accountID,
		id,
		resourceAPIKeyOpts(state.APIKey)...,
	)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading AhaSend SMTP credential", formatAPIError(err))
		return
	}

	newState := SMTPCredentialResourceModel{}
	resp.Diagnostics.Append(flattenSMTPCredential(ctx, cred, state, &newState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if newState.AccountID.IsNull() || newState.AccountID.IsUnknown() || newState.AccountID.ValueString() == "" {
		newState.AccountID = types.StringValue(accountID.String())
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update allows a state-only api_key rotation; all other attributes ForceNew.
func (r *SMTPCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SMTPCredentialResourceModel
	var state SMTPCredentialResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Name.Equal(state.Name) ||
		!plan.Scope.Equal(state.Scope) ||
		!plan.Sandbox.Equal(state.Sandbox) ||
		!plan.Domains.Equal(state.Domains) ||
		!plan.AccountID.Equal(state.AccountID) ||
		!plan.ID.Equal(state.ID) {
		resp.Diagnostics.AddError(
			"Unexpected Update",
			"ahasend_smtp_credential only supports in-place updates of api_key. "+
				"Changing name, scope, sandbox, domains, or account_id forces replacement.",
		)
		return
	}

	// api_key is not an AhaSend SMTP field; persist the planned Bearer for Read/Delete.
	newState := state
	newState.APIKey = plan.APIKey
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Delete removes the SMTP credential; missing resources are treated as success.
func (r *SMTPCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SMTPCredentialResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := parseUUIDAttr(state.ID, "id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, accountDiags := r.resolveAccountID(state.AccountID)
	resp.Diagnostics.Append(accountDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.api.SMTPCredentialsAPI.DeleteSMTPCredential(
		ctx,
		accountID,
		id,
		resourceAPIKeyOpts(state.APIKey)...,
	)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting AhaSend SMTP credential", formatAPIError(err))
		return
	}
}

// ImportState imports by SMTP credential UUID or ACCOUNT_ID/CREDENTIAL_UUID.
func (r *SMTPCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 1:
		if _, err := uuid.Parse(parts[0]); err != nil {
			resp.Diagnostics.AddError(
				"Unexpected Import Identifier",
				fmt.Sprintf("Expected SMTP credential UUID, got: %q", req.ID),
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_id"), r.client.accountID.String())...)
	case 2:
		if _, err := uuid.Parse(parts[0]); err != nil {
			resp.Diagnostics.AddError(
				"Unexpected Import Identifier",
				fmt.Sprintf("Expected ACCOUNT_ID to be a UUID in ACCOUNT_ID/CREDENTIAL_UUID, got: %q", parts[0]),
			)
			return
		}
		if _, err := uuid.Parse(parts[1]); err != nil {
			resp.Diagnostics.AddError(
				"Unexpected Import Identifier",
				fmt.Sprintf("Expected CREDENTIAL_UUID to be a UUID in ACCOUNT_ID/CREDENTIAL_UUID, got: %q", parts[1]),
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
	default:
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import ID of the form CREDENTIAL_UUID or ACCOUNT_ID/CREDENTIAL_UUID, got: %q", req.ID),
		)
	}
}

// resolveAccountID returns the resource account override or the provider default account.
func (r *SMTPCredentialResource) resolveAccountID(configured types.String) (uuid.UUID, diag.Diagnostics) {
	var diags diag.Diagnostics
	if configured.IsNull() || configured.IsUnknown() || configured.ValueString() == "" {
		return r.client.accountID, diags
	}
	id, err := uuid.Parse(configured.ValueString())
	if err != nil {
		diags.AddAttributeError(
			path.Root("account_id"),
			"Invalid account_id",
			err.Error(),
		)
		return uuid.Nil, diags
	}
	return id, diags
}

// flattenSMTPCredential copies credential fields into state, preserving prior password and api_key.
func flattenSMTPCredential(ctx context.Context, cred *responses.SMTPCredential, prior SMTPCredentialResourceModel, out *SMTPCredentialResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	out.ID = types.StringValue(cred.ID.String())
	out.Name = types.StringValue(cred.Name)
	out.Username = types.StringValue(cred.Username)
	out.Sandbox = types.BoolValue(cred.Sandbox)
	out.Scope = types.StringValue(cred.Scope)
	out.CreatedAt = types.StringValue(cred.CreatedAt.UTC().Format(time.RFC3339))
	out.UpdatedAt = types.StringValue(cred.UpdatedAt.UTC().Format(time.RFC3339))

	// Resource api_key is not returned by the API; preserve from prior state/plan.
	out.APIKey = prior.APIKey

	// Prefer configured/prior account_id; API may not echo it on SMTP payloads.
	if !prior.AccountID.IsNull() && !prior.AccountID.IsUnknown() && prior.AccountID.ValueString() != "" {
		out.AccountID = prior.AccountID
	} else {
		out.AccountID = types.StringNull()
	}

	if cred.Password != "" {
		out.Password = types.StringValue(cred.Password)
	} else if !prior.Password.IsNull() && !prior.Password.IsUnknown() {
		out.Password = prior.Password
	} else {
		out.Password = types.StringNull()
	}

	domainNames := cred.Domains
	if domainNames == nil {
		domainNames = []string{}
	}
	domains, domainDiags := types.ListValueFrom(ctx, types.StringType, domainNames)
	diags.Append(domainDiags...)
	out.Domains = domains

	return diags
}
