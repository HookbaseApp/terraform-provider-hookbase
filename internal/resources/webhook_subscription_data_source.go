package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var _ datasource.DataSource = &WebhookSubscriptionDataSource{}

type WebhookSubscriptionDataSource struct {
	client *client.Client
}

type WebhookSubscriptionDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	EndpointID       types.String `tfsdk:"endpoint_id"`
	EventTypeID      types.String `tfsdk:"event_type_id"`
	FilterExpression types.String `tfsdk:"filter_expression"`
	LabelFilterMode  types.String `tfsdk:"label_filter_mode"`
	TransformID      types.String `tfsdk:"transform_id"`
	IsEnabled        types.Bool   `tfsdk:"is_enabled"`
	PriorityOverride types.Int64  `tfsdk:"priority_override"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

func NewWebhookSubscriptionDataSource() datasource.DataSource {
	return &WebhookSubscriptionDataSource{}
}

func (d *WebhookSubscriptionDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_subscription"
}

func (d *WebhookSubscriptionDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Hookbase webhook subscription by ID.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Required: true},
			"endpoint_id":      schema.StringAttribute{Computed: true},
			"event_type_id":    schema.StringAttribute{Computed: true},
			"filter_expression": schema.StringAttribute{Computed: true},
			"label_filter_mode": schema.StringAttribute{Computed: true},
			"transform_id":     schema.StringAttribute{Computed: true},
			"is_enabled":       schema.BoolAttribute{Computed: true},
			"priority_override": schema.Int64Attribute{Computed: true},
			"created_at":       schema.StringAttribute{Computed: true},
		},
	}
}

func (d *WebhookSubscriptionDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WebhookSubscriptionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config WebhookSubscriptionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sub, err := d.client.GetWebhookSubscription(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook subscription", err.Error())
		return
	}

	config.ID = types.StringValue(sub.ID)
	config.EndpointID = types.StringValue(sub.EndpointID)
	config.EventTypeID = types.StringValue(sub.EventTypeID)
	config.FilterExpression = stringPtrToValue(sub.FilterExpression)
	config.LabelFilterMode = types.StringValue(sub.LabelFilterMode)
	config.TransformID = stringPtrToValue(sub.TransformID)
	config.IsEnabled = types.BoolValue(sub.IsEnabled)
	config.CreatedAt = types.StringValue(sub.CreatedAt)

	if sub.PriorityOverride != nil {
		config.PriorityOverride = types.Int64Value(int64(*sub.PriorityOverride))
	} else {
		config.PriorityOverride = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
