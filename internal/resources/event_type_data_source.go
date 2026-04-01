package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var _ datasource.DataSource = &EventTypeDataSource{}

type EventTypeDataSource struct {
	client *client.Client
}

type EventTypeDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	DisplayName      types.String `tfsdk:"display_name"`
	Description      types.String `tfsdk:"description"`
	Category         types.String `tfsdk:"category"`
	Schema           types.String `tfsdk:"schema"`
	ExamplePayload   types.String `tfsdk:"example_payload"`
	DocumentationURL types.String `tfsdk:"documentation_url"`
	IsEnabled        types.Bool   `tfsdk:"is_enabled"`
	IsDeprecated     types.Bool   `tfsdk:"is_deprecated"`
	DefaultPriority  types.Int64  `tfsdk:"default_priority"`
	SchemaVersion    types.Int64  `tfsdk:"schema_version"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func NewEventTypeDataSource() datasource.DataSource {
	return &EventTypeDataSource{}
}

func (d *EventTypeDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event_type"
}

func (d *EventTypeDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Hookbase event type by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Optional: true, Computed: true},
			"name":              schema.StringAttribute{Optional: true, Computed: true, Description: "Event type name (e.g. order.created)."},
			"display_name":      schema.StringAttribute{Computed: true},
			"description":       schema.StringAttribute{Computed: true},
			"category":          schema.StringAttribute{Computed: true},
			"schema":            schema.StringAttribute{Computed: true},
			"example_payload":   schema.StringAttribute{Computed: true},
			"documentation_url": schema.StringAttribute{Computed: true},
			"is_enabled":        schema.BoolAttribute{Computed: true},
			"is_deprecated":     schema.BoolAttribute{Computed: true},
			"default_priority":  schema.Int64Attribute{Computed: true},
			"schema_version":    schema.Int64Attribute{Computed: true},
			"created_at":        schema.StringAttribute{Computed: true},
			"updated_at":        schema.StringAttribute{Computed: true},
		},
	}
}

func (d *EventTypeDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *EventTypeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config EventTypeDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var et *client.EventType
	var err error

	if !config.ID.IsNull() {
		et, err = d.client.GetEventType(ctx, config.ID.ValueString())
	} else if !config.Name.IsNull() {
		et, err = d.client.GetEventTypeByName(ctx, config.Name.ValueString())
	} else {
		resp.Diagnostics.AddError("Missing Identifier", "Either id or name must be provided.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading event type", err.Error())
		return
	}

	config.ID = types.StringValue(et.ID)
	config.Name = types.StringValue(et.Name)
	config.DisplayName = types.StringValue(et.DisplayName)
	config.Description = stringPtrToValue(et.Description)
	config.Category = stringPtrToValue(et.Category)
	config.Schema = stringPtrToValue(et.Schema)
	config.ExamplePayload = stringPtrToValue(et.ExamplePayload)
	config.DocumentationURL = stringPtrToValue(et.DocumentationURL)
	config.IsEnabled = types.BoolValue(et.IsEnabled)
	config.IsDeprecated = types.BoolValue(et.IsDeprecated)
	config.DefaultPriority = types.Int64Value(int64(et.DefaultPriority))
	config.SchemaVersion = types.Int64Value(int64(et.SchemaVersion))
	config.CreatedAt = types.StringValue(et.CreatedAt)
	config.UpdatedAt = types.StringValue(et.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
