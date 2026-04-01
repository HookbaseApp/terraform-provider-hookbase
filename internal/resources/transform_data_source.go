package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var _ datasource.DataSource = &TransformDataSource{}

type TransformDataSource struct {
	client *client.Client
}

type TransformDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Slug          types.String `tfsdk:"slug"`
	Description   types.String `tfsdk:"description"`
	Code          types.String `tfsdk:"code"`
	TransformType types.String `tfsdk:"transform_type"`
	InputFormat   types.String `tfsdk:"input_format"`
	OutputFormat  types.String `tfsdk:"output_format"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func NewTransformDataSource() datasource.DataSource {
	return &TransformDataSource{}
}

func (d *TransformDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transform"
}

func (d *TransformDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Hookbase transform by ID or slug.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Optional: true, Computed: true},
			"name":           schema.StringAttribute{Computed: true},
			"slug":           schema.StringAttribute{Optional: true, Computed: true},
			"description":    schema.StringAttribute{Computed: true},
			"code":           schema.StringAttribute{Computed: true},
			"transform_type": schema.StringAttribute{Computed: true},
			"input_format":   schema.StringAttribute{Computed: true},
			"output_format":  schema.StringAttribute{Computed: true},
			"created_at":     schema.StringAttribute{Computed: true},
			"updated_at":     schema.StringAttribute{Computed: true},
		},
	}
}

func (d *TransformDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TransformDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config TransformDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var t *client.Transform
	var err error

	if !config.ID.IsNull() {
		t, err = d.client.GetTransform(ctx, config.ID.ValueString())
	} else if !config.Slug.IsNull() {
		t, err = d.client.GetTransformBySlug(ctx, config.Slug.ValueString())
	} else {
		resp.Diagnostics.AddError("Missing Identifier", "Either id or slug must be provided.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading transform", err.Error())
		return
	}

	config.ID = types.StringValue(t.ID)
	config.Name = types.StringValue(t.Name)
	config.Slug = types.StringValue(t.Slug)
	config.Description = stringPtrToValue(t.Description)
	config.Code = types.StringValue(t.Code)
	config.TransformType = types.StringValue(t.GetTransformType())
	config.InputFormat = types.StringValue(t.GetInputFormat())
	config.OutputFormat = types.StringValue(t.GetOutputFormat())
	config.CreatedAt = types.StringValue(t.CreatedAt)
	config.UpdatedAt = types.StringValue(t.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
