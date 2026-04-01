package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var _ datasource.DataSource = &FilterDataSource{}

type FilterDataSource struct {
	client *client.Client
}

type FilterDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Slug       types.String `tfsdk:"slug"`
	Description types.String `tfsdk:"description"`
	Conditions types.String `tfsdk:"conditions"`
	Logic      types.String `tfsdk:"logic"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func NewFilterDataSource() datasource.DataSource {
	return &FilterDataSource{}
}

func (d *FilterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_filter"
}

func (d *FilterDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Hookbase filter by ID or slug.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Optional: true, Computed: true},
			"name":        schema.StringAttribute{Computed: true},
			"slug":        schema.StringAttribute{Optional: true, Computed: true},
			"description": schema.StringAttribute{Computed: true},
			"conditions":  schema.StringAttribute{Computed: true, Description: "JSON array of filter conditions."},
			"logic":       schema.StringAttribute{Computed: true},
			"created_at":  schema.StringAttribute{Computed: true},
			"updated_at":  schema.StringAttribute{Computed: true},
		},
	}
}

func (d *FilterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FilterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config FilterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var f *client.Filter
	var err error

	if !config.ID.IsNull() {
		f, err = d.client.GetFilter(ctx, config.ID.ValueString())
	} else if !config.Slug.IsNull() {
		f, err = d.client.GetFilterBySlug(ctx, config.Slug.ValueString())
	} else {
		resp.Diagnostics.AddError("Missing Identifier", "Either id or slug must be provided.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading filter", err.Error())
		return
	}

	config.ID = types.StringValue(f.ID)
	config.Name = types.StringValue(f.Name)
	config.Slug = types.StringValue(f.Slug)
	config.Description = stringPtrToValue(f.Description)
	config.Logic = types.StringValue(f.Logic)
	config.CreatedAt = types.StringValue(f.CreatedAt)
	config.UpdatedAt = types.StringValue(f.UpdatedAt)

	if f.Conditions != nil {
		config.Conditions = types.StringValue(string(f.Conditions))
	} else {
		config.Conditions = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
