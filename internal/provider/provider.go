package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
	"github.com/hookbase/terraform-provider-hookbase/internal/resources"
)

var _ provider.Provider = &HookbaseProvider{}

type HookbaseProvider struct {
	version string
}

type HookbaseProviderModel struct {
	APIKey         types.String `tfsdk:"api_key"`
	APIURL         types.String `tfsdk:"api_url"`
	OrganizationID types.String `tfsdk:"organization_id"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &HookbaseProvider{version: version}
	}
}

func (p *HookbaseProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "hookbase"
	resp.Version = p.version
}

func (p *HookbaseProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage Hookbase webhook infrastructure — both inbound and outbound — as code.",
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Description: "Hookbase API key (whr_... prefix). Can also be set via HOOKBASE_API_KEY environment variable.",
				Optional:    true,
				Sensitive:   true,
			},
			"api_url": schema.StringAttribute{
				Description: "Hookbase API URL. Defaults to https://api.hookbase.app. Can also be set via HOOKBASE_API_URL environment variable.",
				Optional:    true,
			},
			"organization_id": schema.StringAttribute{
				Description: "Hookbase organization ID. Can also be set via HOOKBASE_ORG_ID environment variable.",
				Optional:    true,
			},
		},
	}
}

func (p *HookbaseProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config HookbaseProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiKey := os.Getenv("HOOKBASE_API_KEY")
	if !config.APIKey.IsNull() {
		apiKey = config.APIKey.ValueString()
	}
	if apiKey == "" {
		resp.Diagnostics.AddError("Missing API Key", "api_key must be set in the provider configuration or via the HOOKBASE_API_KEY environment variable.")
		return
	}

	apiURL := "https://api.hookbase.app"
	if envURL := os.Getenv("HOOKBASE_API_URL"); envURL != "" {
		apiURL = envURL
	}
	if !config.APIURL.IsNull() {
		apiURL = config.APIURL.ValueString()
	}

	orgID := os.Getenv("HOOKBASE_ORG_ID")
	if !config.OrganizationID.IsNull() {
		orgID = config.OrganizationID.ValueString()
	}
	// orgID is optional — the API infers it from the API key when not provided

	c := client.New(apiURL, orgID, apiKey)
	resp.DataSourceData = c
	resp.ResourceData = c
}

func (p *HookbaseProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewSourceResource,
		resources.NewDestinationResource,
		resources.NewRouteResource,
		resources.NewTransformResource,
		resources.NewFilterResource,
		resources.NewWebhookApplicationResource,
		resources.NewWebhookEndpointResource,
		resources.NewWebhookSubscriptionResource,
		resources.NewEventTypeResource,
	}
}

func (p *HookbaseProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		resources.NewSourceDataSource,
		resources.NewDestinationDataSource,
		resources.NewRouteDataSource,
		resources.NewTransformDataSource,
		resources.NewFilterDataSource,
		resources.NewWebhookApplicationDataSource,
		resources.NewWebhookEndpointDataSource,
		resources.NewWebhookSubscriptionDataSource,
		resources.NewEventTypeDataSource,
	}
}
