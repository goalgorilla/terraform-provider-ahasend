package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure AhaSendProvider satisfies provider.Provider.
var _ provider.Provider = &AhaSendProvider{}

// AhaSendProvider defines the provider implementation.
type AhaSendProvider struct {
	version string
}

// AhaSendProviderModel describes the provider data model.
type AhaSendProviderModel struct {
	APIKey    types.String `tfsdk:"api_key"`
	AccountID types.String `tfsdk:"account_id"`
	Endpoint  types.String `tfsdk:"endpoint"`
}

// Metadata sets the provider type name and version reported to Terraform.
func (p *AhaSendProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "ahasend"
	resp.Version = p.version
}

// Schema defines provider configuration attributes (api_key, account_id, endpoint).
func (p *AhaSendProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The AhaSend provider manages AhaSend account resources such as sending domains, API keys, webhooks, SMTP credentials, and (optionally) Platform Partner sub accounts. " +
			"Configure it with an API key and account ID for the AhaSend account you want to manage. " +
			"Authenticate as a sub account by pointing a provider alias at that child's credentials.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "AhaSend API key (Bearer token). May also be set via the `AHASEND_API_KEY` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"account_id": schema.StringAttribute{
				MarkdownDescription: "AhaSend account UUID used as the default account for resources. May also be set via the `AHASEND_ACCOUNT_ID` environment variable.",
				Optional:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Optional API base URL. Defaults to `https://api.ahasend.com`. May also be set via the `AHASEND_ENDPOINT` environment variable.",
				Optional:            true,
			},
		},
	}
}

// Configure builds the shared AhaSend API client from config and environment variables.
// Unknown configured values produce attribute errors instead of falling through to env defaults.
func (p *AhaSendProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data AhaSendProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.APIKey.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Unknown AhaSend API Key",
			"The provider cannot create the AhaSend API client because the configured api_key value is unknown. "+
				"Set a known api_key value, or omit the api_key attribute and use the AHASEND_API_KEY environment variable.",
		)
	}
	if data.AccountID.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Unknown AhaSend Account ID",
			"The provider cannot create the AhaSend API client because the configured account_id value is unknown. "+
				"Set a known account_id value, or omit the account_id attribute and use the AHASEND_ACCOUNT_ID environment variable.",
		)
	}
	if data.Endpoint.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("endpoint"),
			"Unknown AhaSend Endpoint",
			"The provider cannot create the AhaSend API client because the configured endpoint value is unknown. "+
				"Set a known endpoint value, omit the endpoint attribute to use the default (https://api.ahasend.com), "+
				"or omit endpoint and use the AHASEND_ENDPOINT environment variable.",
		)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("AHASEND_API_KEY")
	if !data.APIKey.IsNull() {
		apiKey = data.APIKey.ValueString()
	}

	accountID := os.Getenv("AHASEND_ACCOUNT_ID")
	if !data.AccountID.IsNull() {
		accountID = data.AccountID.ValueString()
	}

	endpoint := os.Getenv("AHASEND_ENDPOINT")
	if !data.Endpoint.IsNull() {
		endpoint = data.Endpoint.ValueString()
	}
	if endpoint == "" {
		endpoint = "https://api.ahasend.com"
	}

	if apiKey == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing AhaSend API Key",
			"The provider cannot create the AhaSend API client because there is an empty value for the API key. "+
				"Set the api_key value in the provider configuration or use the AHASEND_API_KEY environment variable.",
		)
	}

	if accountID == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("account_id"),
			"Missing AhaSend Account ID",
			"The provider cannot create the AhaSend API client because there is an empty value for the account ID. "+
				"Set the account_id value in the provider configuration or use the AHASEND_ACCOUNT_ID environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	client, err := newAhaSendClient(apiKey, accountID, endpoint)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create AhaSend API client",
			err.Error(),
		)
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

// Resources registers all managed resource factories for this provider.
func (p *AhaSendProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewDomainResource,
		NewAPIKeyResource,
		NewWebhookResource,
		NewSMTPCredentialResource,
		NewSubAccountResource,
		NewSubAccountAPIKeyResource,
	}
}

// DataSources registers data sources; none are implemented yet.
func (p *AhaSendProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}

// New returns a function that creates a new AhaSend provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &AhaSendProvider{
			version: version,
		}
	}
}
