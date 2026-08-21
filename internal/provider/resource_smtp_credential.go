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
			"`name`, `scope`, `sandbox`, or `domains` forces a new resource. " +
			"The one-time `password` is returned on create and preserved in state.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "SMTP credential UUID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

	createReq := requests.CreateSMTPCredentialRequest{
		Name:    plan.Name.ValueString(),
		Scope:   plan.Scope.ValueString(),
		Sandbox: plan.Sandbox.ValueBool(),
	}
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		domains, diags := listToStringSlice(ctx, plan.Domains)
		resp.Diagnostics.Append(diags...)
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
	created, _, err := r.client.api.SMTPCredentialsAPI.CreateSMTPCredential(
		ctx,
		r.client.accountID,
		createReq,
		api.WithIdempotencyKey(idempotencyKey),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AhaSend SMTP credential", formatAPIError(err))
		return
	}

	state := SMTPCredentialResourceModel{}
	resp.Diagnostics.Append(flattenSMTPCredential(ctx, created, types.StringNull(), &state)...)
	if resp.Diagnostics.HasError() {
		return
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

	cred, _, err := r.client.api.SMTPCredentialsAPI.GetSMTPCredential(ctx, r.client.accountID, id)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading AhaSend SMTP credential", formatAPIError(err))
		return
	}

	newState := SMTPCredentialResourceModel{}
	resp.Diagnostics.Append(flattenSMTPCredential(ctx, cred, state.Password, &newState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update always errors; configuration changes force replacement.
func (r *SMTPCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// sandbox may be the only attribute that can change without RequiresReplace if we forget
	// a plan modifier; all config fields ForceNew. Treat unexpected Update as replace-only.
	resp.Diagnostics.AddError(
		"Unexpected Update",
		"ahasend_smtp_credential does not support in-place updates. Changing configuration forces replacement.",
	)
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

	_, _, err := r.client.api.SMTPCredentialsAPI.DeleteSMTPCredential(ctx, r.client.accountID, id)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting AhaSend SMTP credential", formatAPIError(err))
		return
	}
}

// ImportState imports by SMTP credential UUID.
func (r *SMTPCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	if _, err := uuid.Parse(id); err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected SMTP credential UUID, got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// flattenSMTPCredential copies credential fields into state, preserving priorPassword when the API omits it.
func flattenSMTPCredential(ctx context.Context, cred *responses.SMTPCredential, priorPassword types.String, out *SMTPCredentialResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	out.ID = types.StringValue(cred.ID.String())
	out.Name = types.StringValue(cred.Name)
	out.Username = types.StringValue(cred.Username)
	out.Sandbox = types.BoolValue(cred.Sandbox)
	out.Scope = types.StringValue(cred.Scope)
	out.CreatedAt = types.StringValue(cred.CreatedAt.UTC().Format(time.RFC3339))
	out.UpdatedAt = types.StringValue(cred.UpdatedAt.UTC().Format(time.RFC3339))

	if cred.Password != "" {
		out.Password = types.StringValue(cred.Password)
	} else if !priorPassword.IsNull() && !priorPassword.IsUnknown() {
		out.Password = priorPassword
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
