package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var _ datasource.DataSource = &WebhookApplicationDataSource{}

type WebhookApplicationDataSource struct {
	client *client.Client
}

type WebhookApplicationDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ExternalID         types.String `tfsdk:"external_id"`
	Name               types.String `tfsdk:"name"`
	RateLimitPerSecond types.Int64  `tfsdk:"rate_limit_per_second"`
	RateLimitPerMinute types.Int64  `tfsdk:"rate_limit_per_minute"`
	RateLimitPerHour   types.Int64  `tfsdk:"rate_limit_per_hour"`
	IsDisabled         types.Bool   `tfsdk:"is_disabled"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func NewWebhookApplicationDataSource() datasource.DataSource {
	return &WebhookApplicationDataSource{}
}

func (d *WebhookApplicationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_application"
}

func (d *WebhookApplicationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Hookbase webhook application by ID or external ID.",
		Attributes: map[string]schema.Attribute{
			"id":                   schema.StringAttribute{Optional: true, Computed: true},
			"external_id":          schema.StringAttribute{Optional: true, Computed: true, Description: "External ID. Provide either id or external_id."},
			"name":                 schema.StringAttribute{Computed: true},
			"rate_limit_per_second": schema.Int64Attribute{Computed: true},
			"rate_limit_per_minute": schema.Int64Attribute{Computed: true},
			"rate_limit_per_hour":   schema.Int64Attribute{Computed: true},
			"is_disabled":          schema.BoolAttribute{Computed: true},
			"created_at":           schema.StringAttribute{Computed: true},
			"updated_at":           schema.StringAttribute{Computed: true},
		},
	}
}

func (d *WebhookApplicationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WebhookApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WebhookApplicationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var app *client.WebhookApplication
	var err error

	if !config.ID.IsNull() {
		app, err = d.client.GetWebhookApplication(ctx, config.ID.ValueString())
	} else if !config.ExternalID.IsNull() {
		app, err = d.client.GetWebhookApplicationByExternalID(ctx, config.ExternalID.ValueString())
	} else {
		resp.Diagnostics.AddError("Missing Identifier", "Either id or external_id must be provided.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook application", err.Error())
		return
	}

	config.ID = types.StringValue(app.ID)
	config.ExternalID = stringPtrToValue(app.ExternalID)
	config.Name = types.StringValue(app.Name)
	config.RateLimitPerSecond = types.Int64Value(int64(app.RateLimitPerSecond))
	config.RateLimitPerMinute = types.Int64Value(int64(app.RateLimitPerMinute))
	config.RateLimitPerHour = types.Int64Value(int64(app.RateLimitPerHour))
	config.IsDisabled = types.BoolValue(app.IsDisabled)
	config.CreatedAt = types.StringValue(app.CreatedAt)
	config.UpdatedAt = types.StringValue(app.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
