package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/AhaSend/ahasend-go/api"
	"github.com/AhaSend/ahasend-go/models/responses"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// dkimSelectorPattern validates Partner custom DKIM selector labels.
var dkimSelectorPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,60}[a-z0-9])?$`)

var (
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithConfigure   = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
	_ resource.ResourceWithModifyPlan  = &DomainResource{}
)

// DomainResource manages an AhaSend sending domain.
type DomainResource struct {
	client *ahasendClient
}

// domainDNSRecordModel is a nested DNS record object.
type domainDNSRecordModel struct {
	Type       types.String `tfsdk:"type"`
	Label      types.String `tfsdk:"label"`
	Host       types.String `tfsdk:"host"`
	Content    types.String `tfsdk:"content"`
	Required   types.Bool   `tfsdk:"required"`
	Propagated types.Bool   `tfsdk:"propagated"`
}

// DomainResourceModel is the Terraform state model for ahasend_domain.
type DomainResourceModel struct {
	ID                       types.String `tfsdk:"id"`
	Domain                   types.String `tfsdk:"domain"`
	AccountID                types.String `tfsdk:"account_id"`
	TrackingSubdomain        types.String `tfsdk:"tracking_subdomain"`
	ReturnPathSubdomain      types.String `tfsdk:"return_path_subdomain"`
	SubscriptionSubdomain    types.String `tfsdk:"subscription_subdomain"`
	MediaSubdomain           types.String `tfsdk:"media_subdomain"`
	DKIMSelector             types.String `tfsdk:"dkim_selector"`
	DKIMPrivateKey           types.String `tfsdk:"dkim_private_key"`
	DKIMPrivateKeyVersion    types.String `tfsdk:"dkim_private_key_version"`
	DKIMRotationIntervalDays types.Int64  `tfsdk:"dkim_rotation_interval_days"`
	CheckDNS                 types.Bool   `tfsdk:"check_dns"`
	DNSValid                 types.Bool   `tfsdk:"dns_valid"`
	LastDNSCheckAt           types.String `tfsdk:"last_dns_check_at"`
	RotationReady            types.Bool   `tfsdk:"rotation_ready"`
	DSNRecipient             types.String `tfsdk:"dsn_recipient"`
	DNSRecords               types.List   `tfsdk:"dns_records"`
	CreatedAt                types.String `tfsdk:"created_at"`
	UpdatedAt                types.String `tfsdk:"updated_at"`
}

// createDomainBody extends the SDK create request with Partner-only dkim_selector,
// which is present in the OpenAPI spec but not yet in ahasend-go request models.
type createDomainBody struct {
	Domain                   string  `json:"domain"`
	DKIMPrivateKey           *string `json:"dkim_private_key,omitempty"`
	TrackingSubdomain        *string `json:"tracking_subdomain,omitempty"`
	ReturnPathSubdomain      *string `json:"return_path_subdomain,omitempty"`
	SubscriptionSubdomain    *string `json:"subscription_subdomain,omitempty"`
	MediaSubdomain           *string `json:"media_subdomain,omitempty"`
	DKIMRotationIntervalDays *int    `json:"dkim_rotation_interval_days,omitempty"`
	DKIMSelector             *string `json:"dkim_selector,omitempty"`
}

// updateDomainBody extends the SDK update request with optional dkim_selector
// and Partner dkim_private_key (create-shaped field used for key rotation).
type updateDomainBody struct {
	TrackingSubdomain        *string `json:"tracking_subdomain,omitempty"`
	ReturnPathSubdomain      *string `json:"return_path_subdomain,omitempty"`
	SubscriptionSubdomain    *string `json:"subscription_subdomain,omitempty"`
	MediaSubdomain           *string `json:"media_subdomain,omitempty"`
	DKIMRotationIntervalDays *int    `json:"dkim_rotation_interval_days,omitempty"`
	DKIMSelector             *string `json:"dkim_selector,omitempty"`
	DKIMPrivateKey           *string `json:"dkim_private_key,omitempty"`
}

// domainAPIResponse mirrors responses.Domain and optionally captures dkim_selector
// when the API returns it.
type domainAPIResponse struct {
	responses.Domain
	DKIMSelector *string `json:"dkim_selector"`
}

// NewDomainResource returns a new ahasend_domain resource.
func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

// Metadata sets the resource type name to ahasend_domain.
func (r *DomainResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

// Schema defines the ahasend_domain resource attributes.
func (r *DomainResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AhaSend sending domain on the configured account (parent by default). " +
			"Optional custom subdomains and Partner-only DKIM settings are supported. " +
			"DNS verification is checked when `check_dns` is true, but apply does **not** fail when `dns_valid` is false.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "AhaSend domain UUID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"domain": schema.StringAttribute{
				MarkdownDescription: "Fully qualified domain name. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"account_id": schema.StringAttribute{
				MarkdownDescription: "Account UUID that owns the domain. Defaults to the provider `account_id`. " +
					"Set this (with a matching provider alias API key) when managing a sub account domain.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"tracking_subdomain": schema.StringAttribute{
				MarkdownDescription: "Optional custom tracking subdomain. Omit on create to use the account/product default; " +
					"after apply Terraform shows the effective value returned by AhaSend. " +
					"Omitting after create keeps the current effective value.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"return_path_subdomain": schema.StringAttribute{
				MarkdownDescription: "Optional custom return-path subdomain. Omit on create to use the account/product default; " +
					"after apply Terraform shows the effective value returned by AhaSend. " +
					"Omitting after create keeps the current effective value.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subscription_subdomain": schema.StringAttribute{
				MarkdownDescription: "Optional custom subscription management subdomain. Omit on create to use the account/product default; " +
					"after apply Terraform shows the effective value returned by AhaSend. " +
					"Omitting after create keeps the current effective value.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"media_subdomain": schema.StringAttribute{
				MarkdownDescription: "Optional custom media subdomain. Omit on create to use the account/product default; " +
					"after apply Terraform shows the effective value returned by AhaSend. " +
					"Omitting after create keeps the current effective value.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dkim_selector": schema.StringAttribute{
				MarkdownDescription: "Optional custom DKIM selector (Platform Partner). Must be a single DNS label: 1-62 lowercase alphanumeric characters and hyphens, without leading/trailing hyphen. " +
					"Omit on create for no per-domain override; after apply Terraform shows the value returned by AhaSend (null when unset). " +
					"Omitting after create keeps the current override. Clear by setting an empty string (Update sends empty to revert to the default selector).",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.Any(
						stringvalidator.LengthAtMost(0),
						stringvalidator.RegexMatches(
							dkimSelectorPattern,
							"must be a single DNS label: 1-62 lowercase letters, numbers, and hyphens, without leading or trailing hyphen",
						),
					),
				},
			},
			"dkim_private_key": schema.StringAttribute{
				MarkdownDescription: "Optional DKIM RSA private key (Platform Partner, write-only). Minimum 1024 bits; 2048 recommended. " +
					"Not returned by the API and not persisted in Terraform state (requires Terraform 1.11+). Do not generate keys in the provider. " +
					"To rotate on update, change `dkim_private_key_version` and supply a new `dkim_private_key` in the same apply.",
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
			},
			"dkim_private_key_version": schema.StringAttribute{
				MarkdownDescription: "Optional opaque version/trigger for DKIM private key rotation. Changing this value causes Update to " +
					"read write-only `dkim_private_key` from configuration and send it to AhaSend. The key itself is never stored in state.",
				Optional: true,
			},
			"dkim_rotation_interval_days": schema.Int64Attribute{
				MarkdownDescription: "Optional custom DKIM rotation interval in days. Only supported for managed DNS domains on eligible plans. " +
					"Omit on create to use the account default; after apply Terraform shows the effective value from AhaSend. " +
					"Omitting after create keeps the current effective value (does not clear back to account default).",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"check_dns": schema.BoolAttribute{
				MarkdownDescription: "When true (default), trigger AhaSend DNS validation after create/update and on refresh. " +
					"Read skips a new check-dns POST when `last_dns_check_at` is within the last 60 seconds (API cache window). " +
					"Does not fail apply when DNS is invalid.",
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
			},
			"dns_valid": schema.BoolAttribute{
				MarkdownDescription: "Whether AhaSend considers required DNS records valid.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"last_dns_check_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 timestamp of the last DNS check, if any.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"rotation_ready": schema.BoolAttribute{
				MarkdownDescription: "Whether the standby DKIM slot is ready for rotation.",
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"dsn_recipient": schema.StringAttribute{
				MarkdownDescription: "Optional recipient address for delivery status notifications.",
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
			"updated_at": schema.StringAttribute{
				MarkdownDescription: "RFC3339 last update timestamp.",
				Computed:            true,
			},
			"dns_records": schema.ListNestedAttribute{
				MarkdownDescription: "DNS records required for domain verification.",
				Computed:            true,
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							MarkdownDescription: "DNS record type (for example CNAME, TXT, MX).",
							Computed:            true,
						},
						"label": schema.StringAttribute{
							MarkdownDescription: "Human-readable DNS record label.",
							Computed:            true,
						},
						"host": schema.StringAttribute{
							MarkdownDescription: "DNS record host/name.",
							Computed:            true,
						},
						"content": schema.StringAttribute{
							MarkdownDescription: "DNS record content/value.",
							Computed:            true,
						},
						"required": schema.BoolAttribute{
							MarkdownDescription: "Whether this record is required for verification.",
							Computed:            true,
						},
						"propagated": schema.BoolAttribute{
							MarkdownDescription: "Whether AhaSend has observed this record as propagated.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

// Configure stores the provider API client on the resource.
func (r *DomainResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ModifyPlan marks DNS-related computed attributes unknown when subdomain or
// dkim_selector inputs change, so UseStateForUnknown does not keep stale
// dns_records hosts in the plan (inconsistent result after apply).
func (r *DomainResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dnsInputsChanged := !plan.TrackingSubdomain.Equal(state.TrackingSubdomain) ||
		!plan.ReturnPathSubdomain.Equal(state.ReturnPathSubdomain) ||
		!plan.SubscriptionSubdomain.Equal(state.SubscriptionSubdomain) ||
		!plan.MediaSubdomain.Equal(state.MediaSubdomain) ||
		!plan.DKIMSelector.Equal(state.DKIMSelector)

	if !dnsInputsChanged {
		return
	}

	plan.DNSRecords = types.ListUnknown(dnsRecordObjectType())
	plan.DNSValid = types.BoolUnknown()
	plan.LastDNSCheckAt = types.StringUnknown()
	plan.UpdatedAt = types.StringUnknown()

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// Create registers a domain, optionally checks DNS, and stores computed DNS records.
func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel
	var config DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, diags := r.resolveAccountID(plan.AccountID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := createDomainBody{
		Domain: plan.Domain.ValueString(),
	}
	if !plan.TrackingSubdomain.IsNull() && !plan.TrackingSubdomain.IsUnknown() {
		body.TrackingSubdomain = stringPtr(plan.TrackingSubdomain.ValueString())
	}
	if !plan.ReturnPathSubdomain.IsNull() && !plan.ReturnPathSubdomain.IsUnknown() {
		body.ReturnPathSubdomain = stringPtr(plan.ReturnPathSubdomain.ValueString())
	}
	if !plan.SubscriptionSubdomain.IsNull() && !plan.SubscriptionSubdomain.IsUnknown() {
		body.SubscriptionSubdomain = stringPtr(plan.SubscriptionSubdomain.ValueString())
	}
	if !plan.MediaSubdomain.IsNull() && !plan.MediaSubdomain.IsUnknown() {
		body.MediaSubdomain = stringPtr(plan.MediaSubdomain.ValueString())
	}
	if !plan.DKIMSelector.IsNull() && !plan.DKIMSelector.IsUnknown() {
		body.DKIMSelector = stringPtr(plan.DKIMSelector.ValueString())
	}
	// Write-only: read from config (not persisted in plan/state).
	if !config.DKIMPrivateKey.IsNull() && !config.DKIMPrivateKey.IsUnknown() && config.DKIMPrivateKey.ValueString() != "" {
		body.DKIMPrivateKey = stringPtr(config.DKIMPrivateKey.ValueString())
	}
	if !plan.DKIMRotationIntervalDays.IsNull() && !plan.DKIMRotationIntervalDays.IsUnknown() {
		body.DKIMRotationIntervalDays = intPtr(int(plan.DKIMRotationIntervalDays.ValueInt64()))
	}

	var result domainAPIResponse
	token, tokenDiags := ensureIdempotencyToken(ctx, resp.Private)
	resp.Diagnostics.Append(tokenDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	idempotencyKey := domainIdempotencyKey(token)
	reqConfig := api.RequestConfig{
		Method:       http.MethodPost,
		PathTemplate: "/v2/accounts/{account_id}/domains",
		PathParams: map[string]string{
			"account_id": accountID.String(),
		},
		Body:   body,
		Result: &result,
	}
	api.WithIdempotencyKey(idempotencyKey)(&reqConfig)
	_, err := r.client.api.Execute(ctx, reqConfig)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating AhaSend domain",
			formatAPIError(err),
		)
		return
	}

	domain := &result
	if plan.CheckDNS.ValueBool() {
		checked, checkErr := r.checkDNS(ctx, accountID, plan.Domain.ValueString())
		if checkErr != nil {
			resp.Diagnostics.AddWarning(
				"Domain created but DNS check failed",
				formatAPIError(checkErr),
			)
		} else {
			domain = checked
		}
	}

	state := DomainResourceModel{}
	resp.Diagnostics.Append(r.flattenDomain(ctx, domain, plan, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes domain state; when check_dns is true it may POST check-dns unless last check was recent.
func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, diags := r.resolveAccountID(state.AccountID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var domain *domainAPIResponse
	var err error
	if state.CheckDNS.ValueBool() && shouldCheckDNS(state.LastDNSCheckAt) {
		domain, err = r.checkDNS(ctx, accountID, state.Domain.ValueString())
	} else {
		domain, err = r.getDomain(ctx, accountID, state.Domain.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading AhaSend domain",
			formatAPIError(err),
		)
		return
	}

	newState := DomainResourceModel{}
	resp.Diagnostics.Append(r.flattenDomain(ctx, domain, state, &newState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update applies subdomain/DKIM settings; dkim_private_key_version triggers write-only key rotation.
func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DomainResourceModel
	var state DomainResourceModel
	var config DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, diags := r.resolveAccountID(plan.AccountID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := updateDomainBody{}
	hasUpdate := false

	if !plan.TrackingSubdomain.Equal(state.TrackingSubdomain) {
		hasUpdate = true
		if plan.TrackingSubdomain.IsNull() {
			body.TrackingSubdomain = stringPtr("")
		} else {
			body.TrackingSubdomain = stringPtr(plan.TrackingSubdomain.ValueString())
		}
	}
	if !plan.ReturnPathSubdomain.Equal(state.ReturnPathSubdomain) {
		hasUpdate = true
		if plan.ReturnPathSubdomain.IsNull() {
			body.ReturnPathSubdomain = stringPtr("")
		} else {
			body.ReturnPathSubdomain = stringPtr(plan.ReturnPathSubdomain.ValueString())
		}
	}
	if !plan.SubscriptionSubdomain.Equal(state.SubscriptionSubdomain) {
		hasUpdate = true
		if plan.SubscriptionSubdomain.IsNull() {
			body.SubscriptionSubdomain = stringPtr("")
		} else {
			body.SubscriptionSubdomain = stringPtr(plan.SubscriptionSubdomain.ValueString())
		}
	}
	if !plan.MediaSubdomain.Equal(state.MediaSubdomain) {
		hasUpdate = true
		if plan.MediaSubdomain.IsNull() {
			body.MediaSubdomain = stringPtr("")
		} else {
			body.MediaSubdomain = stringPtr(plan.MediaSubdomain.ValueString())
		}
	}
	if !plan.DKIMRotationIntervalDays.Equal(state.DKIMRotationIntervalDays) {
		// Optional+Computed: omit-from-config keeps state via UseStateForUnknown.
		// Null plan must not trigger a no-op "clear" (API has no clear path here).
		if !plan.DKIMRotationIntervalDays.IsNull() && !plan.DKIMRotationIntervalDays.IsUnknown() {
			hasUpdate = true
			body.DKIMRotationIntervalDays = intPtr(int(plan.DKIMRotationIntervalDays.ValueInt64()))
		}
	}
	if !plan.DKIMSelector.Equal(state.DKIMSelector) {
		hasUpdate = true
		if plan.DKIMSelector.IsNull() {
			body.DKIMSelector = stringPtr("")
		} else {
			body.DKIMSelector = stringPtr(plan.DKIMSelector.ValueString())
		}
	}
	if !plan.DKIMPrivateKeyVersion.Equal(state.DKIMPrivateKeyVersion) {
		hasUpdate = true
		if !config.DKIMPrivateKey.IsNull() && !config.DKIMPrivateKey.IsUnknown() && config.DKIMPrivateKey.ValueString() != "" {
			body.DKIMPrivateKey = stringPtr(config.DKIMPrivateKey.ValueString())
		} else {
			resp.Diagnostics.AddAttributeError(
				path.Root("dkim_private_key"),
				"Missing DKIM Private Key",
				"Changing dkim_private_key_version requires a new write-only dkim_private_key in the same configuration.",
			)
			return
		}
	}

	var domain *domainAPIResponse
	if hasUpdate {
		var result domainAPIResponse
		_, err := r.client.api.Execute(ctx, api.RequestConfig{
			Method:       http.MethodPut,
			PathTemplate: "/v2/accounts/{account_id}/domains/{domain}",
			PathParams: map[string]string{
				"account_id": accountID.String(),
				"domain":     plan.Domain.ValueString(),
			},
			Body:   body,
			Result: &result,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				"Error updating AhaSend domain",
				formatAPIError(err),
			)
			return
		}
		domain = &result
	} else {
		var err error
		domain, err = r.getDomain(ctx, accountID, plan.Domain.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Error reading AhaSend domain during update",
				formatAPIError(err),
			)
			return
		}
	}

	if plan.CheckDNS.ValueBool() {
		checked, checkErr := r.checkDNS(ctx, accountID, plan.Domain.ValueString())
		if checkErr != nil {
			resp.Diagnostics.AddWarning(
				"Domain updated but DNS check failed",
				formatAPIError(checkErr),
			)
		} else {
			domain = checked
		}
	}

	newState := DomainResourceModel{}
	resp.Diagnostics.Append(r.flattenDomain(ctx, domain, plan, &newState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Delete removes the domain; missing resources are treated as success.
func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accountID, diags := r.resolveAccountID(state.AccountID)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.api.DomainsAPI.DeleteDomain(ctx, accountID, state.Domain.ValueString())
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError(
			"Error deleting AhaSend domain",
			formatAPIError(err),
		)
		return
	}
}

// ImportState imports DOMAIN or ACCOUNT_ID/DOMAIN.
func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	parts := strings.Split(id, "/")
	switch len(parts) {
	case 1:
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_id"), r.client.accountID.String())...)
	case 2:
		if _, err := uuid.Parse(parts[0]); err != nil {
			resp.Diagnostics.AddError(
				"Unexpected Import Identifier",
				fmt.Sprintf("Expected ACCOUNT_ID to be a UUID in ACCOUNT_ID/DOMAIN, got: %q", parts[0]),
			)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("account_id"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("domain"), parts[1])...)
	default:
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import ID of the form DOMAIN or ACCOUNT_ID/DOMAIN, got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("check_dns"), true)...)
}

// resolveAccountID returns the resource account override or the provider default account.
func (r *DomainResource) resolveAccountID(configured types.String) (uuid.UUID, diag.Diagnostics) {
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

// getDomain fetches a domain without triggering DNS validation.
func (r *DomainResource) getDomain(ctx context.Context, accountID uuid.UUID, domain string) (*domainAPIResponse, error) {
	var result domainAPIResponse
	_, err := r.client.api.Execute(ctx, api.RequestConfig{
		Method:       http.MethodGet,
		PathTemplate: "/v2/accounts/{account_id}/domains/{domain}",
		PathParams: map[string]string{
			"account_id": accountID.String(),
			"domain":     domain,
		},
		Result: &result,
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// checkDNS POSTs AhaSend check-dns and returns the updated domain payload.
func (r *DomainResource) checkDNS(ctx context.Context, accountID uuid.UUID, domain string) (*domainAPIResponse, error) {
	var result domainAPIResponse
	_, err := r.client.api.Execute(ctx, api.RequestConfig{
		Method:       http.MethodPost,
		PathTemplate: "/v2/accounts/{account_id}/domains/{domain}/check-dns",
		PathParams: map[string]string{
			"account_id": accountID.String(),
			"domain":     domain,
		},
		Result: &result,
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// flattenDomain copies API domain fields into state; write-only DKIM key stays null.
func (r *DomainResource) flattenDomain(ctx context.Context, domain *domainAPIResponse, prior DomainResourceModel, out *DomainResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	// domain.Domain is the embedded responses.Domain (field name collides with Domain.Domain string).
	apiDomain := domain.Domain

	out.ID = types.StringValue(apiDomain.ID.String())
	out.Domain = types.StringValue(apiDomain.Domain)
	out.AccountID = types.StringValue(apiDomain.AccountID.String())
	out.DNSValid = types.BoolValue(apiDomain.DNSValid)
	out.RotationReady = types.BoolValue(apiDomain.RotationReady)
	out.CreatedAt = types.StringValue(apiDomain.CreatedAt.UTC().Format(time.RFC3339))
	out.UpdatedAt = types.StringValue(apiDomain.UpdatedAt.UTC().Format(time.RFC3339))
	out.CheckDNS = prior.CheckDNS
	if out.CheckDNS.IsNull() || out.CheckDNS.IsUnknown() {
		out.CheckDNS = types.BoolValue(true)
	}

	out.TrackingSubdomain = optionalString(apiDomain.TrackingSubdomain, prior.TrackingSubdomain)
	out.ReturnPathSubdomain = optionalString(apiDomain.ReturnPathSubdomain, prior.ReturnPathSubdomain)
	out.SubscriptionSubdomain = optionalString(apiDomain.SubscriptionSubdomain, prior.SubscriptionSubdomain)
	out.MediaSubdomain = optionalString(apiDomain.MediaSubdomain, prior.MediaSubdomain)
	out.DKIMSelector = optionalString(domain.DKIMSelector, prior.DKIMSelector)
	out.DSNRecipient = nullableString(apiDomain.DSNRecipient)
	out.DKIMPrivateKeyVersion = prior.DKIMPrivateKeyVersion

	if apiDomain.DKIMRotationIntervalDays != nil {
		out.DKIMRotationIntervalDays = types.Int64Value(int64(*apiDomain.DKIMRotationIntervalDays))
	} else if !prior.DKIMRotationIntervalDays.IsNull() && !prior.DKIMRotationIntervalDays.IsUnknown() {
		out.DKIMRotationIntervalDays = prior.DKIMRotationIntervalDays
	} else {
		// API omitted/null and config omitted → known null (not unknown).
		out.DKIMRotationIntervalDays = types.Int64Null()
	}

	// Write-only attributes are never stored in state.
	out.DKIMPrivateKey = types.StringNull()

	if apiDomain.LastDNSCheckAt != nil {
		out.LastDNSCheckAt = types.StringValue(apiDomain.LastDNSCheckAt.UTC().Format(time.RFC3339))
	} else {
		out.LastDNSCheckAt = types.StringNull()
	}

	records := make([]domainDNSRecordModel, 0, len(apiDomain.DNSRecords))
	for _, rec := range apiDomain.DNSRecords {
		item := domainDNSRecordModel{
			Type:       types.StringValue(rec.Type),
			Host:       types.StringValue(rec.Host),
			Content:    types.StringValue(rec.Content),
			Required:   types.BoolValue(rec.Required),
			Propagated: types.BoolValue(rec.Propagated),
		}
		if rec.Label != nil {
			item.Label = types.StringValue(*rec.Label)
		} else {
			item.Label = types.StringNull()
		}
		records = append(records, item)
	}

	list, listDiags := types.ListValueFrom(ctx, dnsRecordObjectType(), records)
	diags.Append(listDiags...)
	out.DNSRecords = list

	return diags
}

// optionalString prefers a non-empty API value, otherwise keeps prior state (omitempty-safe).
func optionalString(apiValue *string, prior types.String) types.String {
	if apiValue != nil && *apiValue != "" {
		return types.StringValue(*apiValue)
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		return prior
	}
	if apiValue != nil {
		return types.StringValue(*apiValue)
	}
	return types.StringNull()
}

// nullableString maps a nil *string to types.StringNull.
func nullableString(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// dnsRecordObjectType is the Terraform object type for nested dns_records elements.
func dnsRecordObjectType() types.ObjectType {
	return types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type":       types.StringType,
			"label":      types.StringType,
			"host":       types.StringType,
			"content":    types.StringType,
			"required":   types.BoolType,
			"propagated": types.BoolType,
		},
	}
}

// shouldCheckDNS returns true when last_dns_check_at is missing or older than
// the AhaSend ~60s check-dns cache window.
func shouldCheckDNS(lastCheck types.String) bool {
	if lastCheck.IsNull() || lastCheck.IsUnknown() || lastCheck.ValueString() == "" {
		return true
	}
	ts, err := time.Parse(time.RFC3339, lastCheck.ValueString())
	if err != nil {
		return true
	}
	return time.Since(ts) > 60*time.Second
}
