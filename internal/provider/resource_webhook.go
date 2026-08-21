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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = &WebhookResource{}
	_ resource.ResourceWithConfigure      = &WebhookResource{}
	_ resource.ResourceWithImportState    = &WebhookResource{}
	_ resource.ResourceWithValidateConfig = &WebhookResource{}
)

// WebhookResource manages an AhaSend webhook.
type WebhookResource struct {
	client *ahasendClient
}

// WebhookResourceModel is the Terraform state model for ahasend_webhook.
type WebhookResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	URL                  types.String `tfsdk:"url"`
	Enabled              types.Bool   `tfsdk:"enabled"`
	Secret               types.String `tfsdk:"secret"`
	Scope                types.String `tfsdk:"scope"`
	Domains              types.List   `tfsdk:"domains"`
	OnReception          types.Bool   `tfsdk:"on_reception"`
	OnDelivered          types.Bool   `tfsdk:"on_delivered"`
	OnTransientError     types.Bool   `tfsdk:"on_transient_error"`
	OnFailed             types.Bool   `tfsdk:"on_failed"`
	OnBounced            types.Bool   `tfsdk:"on_bounced"`
	OnSuppressed         types.Bool   `tfsdk:"on_suppressed"`
	OnOpened             types.Bool   `tfsdk:"on_opened"`
	OnClicked            types.Bool   `tfsdk:"on_clicked"`
	OnSuppressionCreated types.Bool   `tfsdk:"on_suppression_created"`
	OnDNSError           types.Bool   `tfsdk:"on_dns_error"`
	CreatedAt            types.String `tfsdk:"created_at"`
	UpdatedAt            types.String `tfsdk:"updated_at"`
}

// NewWebhookResource returns a new ahasend_webhook resource.
func NewWebhookResource() resource.Resource {
	return &WebhookResource{}
}

// Metadata sets the resource type name to ahasend_webhook.
func (r *WebhookResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

// Schema defines the ahasend_webhook resource attributes.
func (r *WebhookResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	eventAttr := func(desc string) schema.Attribute {
		return schema.BoolAttribute{
			MarkdownDescription: desc,
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		}
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an AhaSend webhook for delivery and engagement events. " +
			"Use `scope = \"global\"` for account-wide webhooks or `scope = \"scoped\"` with `domains`. " +
			"The signing `secret` is returned on create and preserved in state (not updatable).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Webhook UUID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable webhook name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 255),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "HTTPS endpoint that receives webhook events.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the webhook is enabled. Defaults to true.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Webhook signing secret returned on create. Preserved in state; not returned reliably on later reads and not updatable.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"scope": schema.StringAttribute{
				MarkdownDescription: "Webhook scope: `global` (account-wide) or `scoped` (requires `domains`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf("global", "scoped"),
				},
			},
			"domains": schema.ListAttribute{
				MarkdownDescription: "Domain names for a `scoped` webhook. Required when `scope` is `scoped`; must be omitted or empty when `scope` is `global`. " +
					"Omit on update to leave associations unchanged; set `domains = []` only when intentionally clearing them (global scope).",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1)),
				},
			},
			"on_reception":           eventAttr("Trigger on message reception."),
			"on_delivered":           eventAttr("Trigger on successful delivery."),
			"on_transient_error":     eventAttr("Trigger on transient delivery errors."),
			"on_failed":              eventAttr("Trigger on permanent delivery failure."),
			"on_bounced":             eventAttr("Trigger on bounce."),
			"on_suppressed":          eventAttr("Trigger when a recipient is suppressed."),
			"on_opened":              eventAttr("Trigger on open tracking."),
			"on_clicked":             eventAttr("Trigger on click tracking."),
			"on_suppression_created": eventAttr("Trigger when a suppression is created."),
			"on_dns_error":           eventAttr("Trigger on DNS validation errors."),
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
		},
	}
}

// ValidateConfig enforces the scope/domains contract before plan/apply.
func (r *WebhookResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data WebhookResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateScopeDomains(ctx, data.Scope, data.Domains)...)
}

// Configure stores the provider API client on the resource.
func (r *WebhookResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates a webhook with an idempotent request and stores the signing secret.
func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := webhookPlanToCreateRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, tokenDiags := ensureIdempotencyToken(ctx, resp.Private)
	resp.Diagnostics.Append(tokenDiags...)
	if resp.Diagnostics.HasError() {
		return
	}
	idempotencyKey := webhookIdempotencyKey(token)
	created, _, err := r.client.api.WebhooksAPI.CreateWebhook(
		ctx,
		r.client.accountID,
		createReq,
		api.WithIdempotencyKey(idempotencyKey),
	)
	if err != nil {
		resp.Diagnostics.AddError("Error creating AhaSend webhook", formatAPIError(err))
		return
	}

	state := WebhookResourceModel{}
	resp.Diagnostics.Append(flattenWebhook(ctx, created, types.StringNull(), &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.Secret.IsNull() || state.Secret.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Missing webhook secret on create",
			"AhaSend did not return a signing secret for the new webhook. The resource may exist remotely; import or delete it before retrying.",
		)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Read refreshes webhook state and preserves secret from prior state when omitted by the API.
func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := parseUUIDAttr(state.ID, "id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hook, _, err := r.client.api.WebhooksAPI.GetWebhook(ctx, r.client.accountID, id)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading AhaSend webhook", formatAPIError(err))
		return
	}

	newState := WebhookResourceModel{}
	resp.Diagnostics.Append(flattenWebhook(ctx, hook, state.Secret, &newState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Update applies webhook configuration changes; secret is not updatable.
func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookResourceModel
	var state WebhookResourceModel
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

	updateReq, diags := webhookPlanToUpdateRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updated, _, err := r.client.api.WebhooksAPI.UpdateWebhook(ctx, r.client.accountID, id, updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating AhaSend webhook", formatAPIError(err))
		return
	}

	newState := WebhookResourceModel{}
	resp.Diagnostics.Append(flattenWebhook(ctx, updated, state.Secret, &newState)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

// Delete removes the webhook; missing resources are treated as success.
func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, diags := parseUUIDAttr(state.ID, "id")
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, _, err := r.client.api.WebhooksAPI.DeleteWebhook(ctx, r.client.accountID, id)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting AhaSend webhook", formatAPIError(err))
		return
	}
}

// ImportState imports by webhook UUID.
func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	id := strings.TrimSpace(req.ID)
	if _, err := uuid.Parse(id); err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected webhook UUID, got: %q", req.ID),
		)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// webhookPlanToCreateRequest maps Terraform plan attributes to the SDK create request.
func webhookPlanToCreateRequest(ctx context.Context, plan WebhookResourceModel) (requests.CreateWebhookRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := requests.CreateWebhookRequest{
		Name:                 plan.Name.ValueString(),
		URL:                  plan.URL.ValueString(),
		Enabled:              boolPtr(plan.Enabled.ValueBool()),
		OnReception:          plan.OnReception.ValueBool(),
		OnDelivered:          plan.OnDelivered.ValueBool(),
		OnTransientError:     plan.OnTransientError.ValueBool(),
		OnFailed:             plan.OnFailed.ValueBool(),
		OnBounced:            plan.OnBounced.ValueBool(),
		OnSuppressed:         plan.OnSuppressed.ValueBool(),
		OnOpened:             plan.OnOpened.ValueBool(),
		OnClicked:            plan.OnClicked.ValueBool(),
		OnSuppressionCreated: plan.OnSuppressionCreated.ValueBool(),
		OnDnsError:           plan.OnDNSError.ValueBool(),
		Scope:                plan.Scope.ValueString(),
	}
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		domains, d := listToStringSlice(ctx, plan.Domains)
		diags.Append(d...)
		req.Domains = &domains
	}
	return req, diags
}

// webhookPlanToUpdateRequest maps Terraform plan attributes to the SDK update request.
// Domains are only sent when known: omit (nil) leaves associations unchanged; a known
// empty list clears them. Never coerce Unknown/null to [] — that would wipe scoped bindings.
func webhookPlanToUpdateRequest(ctx context.Context, plan WebhookResourceModel) (requests.UpdateWebhookRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	req := requests.UpdateWebhookRequest{
		Name:                 stringPtr(plan.Name.ValueString()),
		URL:                  stringPtr(plan.URL.ValueString()),
		Enabled:              boolPtr(plan.Enabled.ValueBool()),
		OnReception:          boolPtr(plan.OnReception.ValueBool()),
		OnDelivered:          boolPtr(plan.OnDelivered.ValueBool()),
		OnTransientError:     boolPtr(plan.OnTransientError.ValueBool()),
		OnFailed:             boolPtr(plan.OnFailed.ValueBool()),
		OnBounced:            boolPtr(plan.OnBounced.ValueBool()),
		OnSuppressed:         boolPtr(plan.OnSuppressed.ValueBool()),
		OnOpened:             boolPtr(plan.OnOpened.ValueBool()),
		OnClicked:            boolPtr(plan.OnClicked.ValueBool()),
		OnSuppressionCreated: boolPtr(plan.OnSuppressionCreated.ValueBool()),
		OnDnsError:           boolPtr(plan.OnDNSError.ValueBool()),
		Scope:                stringPtr(plan.Scope.ValueString()),
	}
	if !plan.Domains.IsNull() && !plan.Domains.IsUnknown() {
		domains, d := listToStringSlice(ctx, plan.Domains)
		diags.Append(d...)
		if domains == nil {
			domains = []string{}
		}
		req.Domains = &domains
	}
	return req, diags
}

// flattenWebhook copies webhook fields into state, preserving priorSecret when the API omits it.
func flattenWebhook(ctx context.Context, hook *responses.Webhook, priorSecret types.String, out *WebhookResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	out.ID = types.StringValue(hook.ID.String())
	out.Name = types.StringValue(hook.Name)
	out.URL = types.StringValue(hook.URL)
	out.Enabled = types.BoolValue(hook.Enabled)
	out.Scope = types.StringValue(hook.Scope)
	out.OnReception = types.BoolValue(hook.OnReception)
	out.OnDelivered = types.BoolValue(hook.OnDelivered)
	out.OnTransientError = types.BoolValue(hook.OnTransientError)
	out.OnFailed = types.BoolValue(hook.OnFailed)
	out.OnBounced = types.BoolValue(hook.OnBounced)
	out.OnSuppressed = types.BoolValue(hook.OnSuppressed)
	out.OnOpened = types.BoolValue(hook.OnOpened)
	out.OnClicked = types.BoolValue(hook.OnClicked)
	out.OnSuppressionCreated = types.BoolValue(hook.OnSuppressionCreated)
	out.OnDNSError = types.BoolValue(hook.OnDNSError)
	out.CreatedAt = types.StringValue(hook.CreatedAt.UTC().Format(time.RFC3339))
	out.UpdatedAt = types.StringValue(hook.UpdatedAt.UTC().Format(time.RFC3339))

	if hook.Secret != "" {
		out.Secret = types.StringValue(hook.Secret)
	} else if !priorSecret.IsNull() && !priorSecret.IsUnknown() {
		out.Secret = priorSecret
	} else {
		out.Secret = types.StringNull()
	}

	domainNames := hook.Domains
	if domainNames == nil {
		domainNames = []string{}
	}
	domains, domainDiags := types.ListValueFrom(ctx, types.StringType, domainNames)
	diags.Append(domainDiags...)
	out.Domains = domains

	return diags
}
