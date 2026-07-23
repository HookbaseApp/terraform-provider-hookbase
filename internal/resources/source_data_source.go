package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var _ datasource.DataSource = &SourceDataSource{}

type SourceDataSource struct {
	client *client.Client
}

type SourceDataSourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Slug                    types.String `tfsdk:"slug"`
	ProviderType            types.String `tfsdk:"provider_type"`
	Description             types.String `tfsdk:"description"`
	RejectInvalidSignatures types.Bool   `tfsdk:"reject_invalid_signatures"`
	RateLimitPerMinute      types.Int64  `tfsdk:"rate_limit_per_minute"`
	IsActive                types.Bool   `tfsdk:"is_active"`
	IPFilterMode            types.String `tfsdk:"ip_filter_mode"`
	IPAllowlist             types.List   `tfsdk:"ip_allowlist"`
	IPDenylist              types.List   `tfsdk:"ip_denylist"`
	DedupEnabled            types.Bool   `tfsdk:"dedup_enabled"`
	DedupStrategy           types.String `tfsdk:"dedup_strategy"`
	DedupWindowHours        types.Int64  `tfsdk:"dedup_window_hours"`
	TransientMode           types.Bool   `tfsdk:"transient_mode"`
	AllowedMethods          types.List   `tfsdk:"allowed_methods"`
	CreatedAt               types.String `tfsdk:"created_at"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
}

func NewSourceDataSource() datasource.DataSource {
	return &SourceDataSource{}
}

func (d *SourceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source"
}

func (d *SourceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Look up an existing Hookbase source by ID or slug.",
		Attributes: map[string]schema.Attribute{
			"id":                        schema.StringAttribute{Optional: true, Computed: true, Description: "Source ID."},
			"name":                      schema.StringAttribute{Computed: true},
			"slug":                      schema.StringAttribute{Optional: true, Computed: true, Description: "Source slug. Provide either id or slug."},
			"provider_type":             schema.StringAttribute{Computed: true},
			"description":               schema.StringAttribute{Computed: true},
			"reject_invalid_signatures": schema.BoolAttribute{Computed: true},
			"rate_limit_per_minute":     schema.Int64Attribute{Computed: true},
			"is_active":                 schema.BoolAttribute{Computed: true},
			"ip_filter_mode":            schema.StringAttribute{Computed: true},
			"ip_allowlist":              schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"ip_denylist":               schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"dedup_enabled":             schema.BoolAttribute{Computed: true},
			"dedup_strategy":            schema.StringAttribute{Computed: true},
			"dedup_window_hours":        schema.Int64Attribute{Computed: true},
			"transient_mode":            schema.BoolAttribute{Computed: true},
			"allowed_methods":           schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"created_at":                schema.StringAttribute{Computed: true},
			"updated_at":                schema.StringAttribute{Computed: true},
		},
	}
}

func (d *SourceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config SourceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var source *client.Source
	var err error

	if !config.ID.IsNull() {
		source, err = d.client.GetSource(ctx, config.ID.ValueString())
	} else if !config.Slug.IsNull() {
		source, err = d.client.GetSourceBySlug(ctx, config.Slug.ValueString())
	} else {
		resp.Diagnostics.AddError("Missing Identifier", "Either id or slug must be provided.")
		return
	}

	if err != nil {
		resp.Diagnostics.AddError("Error reading source", err.Error())
		return
	}

	mapSourceToDataSourceState(ctx, source, &config, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}

func mapSourceToDataSourceState(ctx context.Context, source *client.Source, state *SourceDataSourceModel, diags *diag.Diagnostics) {
	state.ID = types.StringValue(source.ID)
	state.Name = types.StringValue(source.Name)
	state.Slug = types.StringValue(source.Slug)
	state.ProviderType = stringPtrToValue(source.Provider)
	state.Description = stringPtrToValue(source.Description)
	state.RejectInvalidSignatures = types.BoolValue(source.RejectInvalidSignatures)
	state.IsActive = types.BoolValue(source.IsActive)
	state.IPFilterMode = types.StringValue(source.IPFilterMode)
	state.DedupEnabled = types.BoolValue(source.DedupEnabled)
	state.DedupStrategy = types.StringValue(source.DedupStrategy)
	state.DedupWindowHours = types.Int64Value(int64(source.DedupWindowHours))
	state.TransientMode = types.BoolValue(source.TransientMode)
	state.CreatedAt = types.StringValue(source.CreatedAt)
	state.UpdatedAt = types.StringValue(source.UpdatedAt)

	if source.RateLimitPerMinute != nil {
		state.RateLimitPerMinute = types.Int64Value(int64(*source.RateLimitPerMinute))
	} else {
		state.RateLimitPerMinute = types.Int64Null()
	}

	state.IPAllowlist = sliceToStringList(ctx, source.IPAllowlist, diags)
	state.IPDenylist = sliceToStringList(ctx, source.IPDenylist, diags)
	state.AllowedMethods = sliceToStringList(ctx, source.AllowedMethods, diags)
}
