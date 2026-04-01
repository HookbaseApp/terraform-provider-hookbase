package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var _ datasource.DataSource = &DestinationDataSource{}

type DestinationDataSource struct {
	client *client.Client
}

type DestinationDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Slug         types.String `tfsdk:"slug"`
	URL          types.String `tfsdk:"url"`
	Method       types.String `tfsdk:"method"`
	AuthType     types.String `tfsdk:"auth_type"`
	TimeoutMs    types.Int64  `tfsdk:"timeout_ms"`
	Type         types.String `tfsdk:"type"`
	UseStaticIP  types.Bool   `tfsdk:"use_static_ip"`
	IsActive     types.Bool   `tfsdk:"is_active"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func NewDestinationDataSource() datasource.DataSource {
	return &DestinationDataSource{}
}

func (d *DestinationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (d *DestinationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Hookbase destination by ID or slug.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Optional: true, Computed: true, Description: "Destination ID."},
			"name":          schema.StringAttribute{Computed: true},
			"slug":          schema.StringAttribute{Optional: true, Computed: true, Description: "Destination slug."},
			"url":           schema.StringAttribute{Computed: true},
			"method":        schema.StringAttribute{Computed: true},
			"auth_type":     schema.StringAttribute{Computed: true},
			"timeout_ms":    schema.Int64Attribute{Computed: true},
			"type":          schema.StringAttribute{Computed: true},
			"use_static_ip": schema.BoolAttribute{Computed: true},
			"is_active":     schema.BoolAttribute{Computed: true},
			"created_at":    schema.StringAttribute{Computed: true},
			"updated_at":    schema.StringAttribute{Computed: true},
		},
	}
}

func (d *DestinationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *DestinationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config DestinationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var dest *client.Destination
	var err error

	if !config.ID.IsNull() {
		dest, err = d.client.GetDestination(ctx, config.ID.ValueString())
	} else if !config.Slug.IsNull() {
		dest, err = d.client.GetDestinationBySlug(ctx, config.Slug.ValueString())
	} else {
		resp.Diagnostics.AddError("Missing Identifier", "Either id or slug must be provided.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading destination", err.Error())
		return
	}

	config.ID = types.StringValue(dest.ID)
	config.Name = types.StringValue(dest.Name)
	config.Slug = types.StringValue(dest.Slug)
	config.URL = types.StringValue(dest.URL)
	config.Method = types.StringValue(dest.Method)
	config.AuthType = stringPtrToValue(dest.AuthType)
	if dest.TimeoutMs != nil {
		config.TimeoutMs = types.Int64Value(int64(*dest.TimeoutMs))
	}
	config.Type = types.StringValue(dest.Type)
	config.UseStaticIP = types.BoolValue(dest.UseStaticIP)
	config.IsActive = types.BoolValue(dest.IsActive)
	config.CreatedAt = types.StringValue(dest.CreatedAt)
	config.UpdatedAt = types.StringValue(dest.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
