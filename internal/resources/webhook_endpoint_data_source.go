package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var _ datasource.DataSource = &WebhookEndpointDataSource{}

type WebhookEndpointDataSource struct {
	client *client.Client
}

type WebhookEndpointDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	ApplicationID   types.String `tfsdk:"application_id"`
	URL             types.String `tfsdk:"url"`
	Description     types.String `tfsdk:"description"`
	TimeoutSeconds  types.Int64  `tfsdk:"timeout_seconds"`
	IsDisabled      types.Bool   `tfsdk:"is_disabled"`
	CircuitState    types.String `tfsdk:"circuit_state"`
	SecretVersion   types.Int64  `tfsdk:"secret_version"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

func NewWebhookEndpointDataSource() datasource.DataSource {
	return &WebhookEndpointDataSource{}
}

func (d *WebhookEndpointDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_endpoint"
}

func (d *WebhookEndpointDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Hookbase webhook endpoint by ID.",
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Required: true},
			"application_id":  schema.StringAttribute{Computed: true},
			"url":             schema.StringAttribute{Computed: true},
			"description":     schema.StringAttribute{Computed: true},
			"timeout_seconds": schema.Int64Attribute{Computed: true},
			"is_disabled":     schema.BoolAttribute{Computed: true},
			"circuit_state":   schema.StringAttribute{Computed: true},
			"secret_version":  schema.Int64Attribute{Computed: true},
			"created_at":      schema.StringAttribute{Computed: true},
			"updated_at":      schema.StringAttribute{Computed: true},
		},
	}
}

func (d *WebhookEndpointDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected DataSource Configure Type", "Expected *client.Client")
		return
	}
	d.client = c
}

func (d *WebhookEndpointDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WebhookEndpointDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ep, err := d.client.GetWebhookEndpoint(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook endpoint", err.Error())
		return
	}

	config.ID = types.StringValue(ep.ID)
	config.ApplicationID = types.StringValue(ep.ApplicationID)
	config.URL = types.StringValue(ep.URL)
	config.Description = stringPtrToValue(ep.Description)
	config.TimeoutSeconds = types.Int64Value(int64(ep.TimeoutSeconds))
	config.IsDisabled = types.BoolValue(ep.IsDisabled)
	config.CircuitState = types.StringValue(ep.CircuitState)
	config.SecretVersion = types.Int64Value(int64(ep.SecretVersion))
	config.CreatedAt = types.StringValue(ep.CreatedAt)
	config.UpdatedAt = types.StringValue(ep.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
