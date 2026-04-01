package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var _ datasource.DataSource = &RouteDataSource{}

type RouteDataSource struct {
	client *client.Client
}

type RouteDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	SourceID        types.String `tfsdk:"source_id"`
	DestinationID   types.String `tfsdk:"destination_id"`
	FilterID        types.String `tfsdk:"filter_id"`
	TransformID     types.String `tfsdk:"transform_id"`
	SchemaID        types.String `tfsdk:"schema_id"`
	Priority        types.Int64  `tfsdk:"priority"`
	IsActive        types.Bool   `tfsdk:"is_active"`
	CreatedAt       types.String `tfsdk:"created_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

func NewRouteDataSource() datasource.DataSource {
	return &RouteDataSource{}
}

func (d *RouteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

func (d *RouteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Hookbase route by ID.",
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Required: true, Description: "Route ID."},
			"name":            schema.StringAttribute{Computed: true},
			"source_id":       schema.StringAttribute{Computed: true},
			"destination_id":  schema.StringAttribute{Computed: true},
			"filter_id":       schema.StringAttribute{Computed: true},
			"transform_id":    schema.StringAttribute{Computed: true},
			"schema_id":       schema.StringAttribute{Computed: true},
			"priority":        schema.Int64Attribute{Computed: true},
			"is_active":       schema.BoolAttribute{Computed: true},
			"created_at":      schema.StringAttribute{Computed: true},
			"updated_at":      schema.StringAttribute{Computed: true},
		},
	}
}

func (d *RouteDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RouteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config RouteDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	route, err := d.client.GetRoute(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading route", err.Error())
		return
	}

	config.ID = types.StringValue(route.ID)
	config.Name = types.StringValue(route.Name)
	config.SourceID = types.StringValue(route.SourceID)
	config.DestinationID = types.StringValue(route.DestinationID)
	config.FilterID = stringPtrToValue(route.FilterID)
	config.TransformID = stringPtrToValue(route.TransformID)
	config.SchemaID = stringPtrToValue(route.SchemaID)
	config.Priority = types.Int64Value(int64(route.Priority))
	config.IsActive = types.BoolValue(route.IsActive)
	config.CreatedAt = types.StringValue(route.CreatedAt)
	config.UpdatedAt = types.StringValue(route.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
