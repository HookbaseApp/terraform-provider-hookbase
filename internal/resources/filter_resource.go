package resources

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var (
	_ resource.Resource                = &FilterResource{}
	_ resource.ResourceWithImportState = &FilterResource{}
)

type FilterResource struct {
	client *client.Client
}

type FilterResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Slug        types.String `tfsdk:"slug"`
	Description types.String `tfsdk:"description"`
	Conditions  types.String `tfsdk:"conditions"`
	Logic       types.String `tfsdk:"logic"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func NewFilterResource() resource.Resource {
	return &FilterResource{}
}

func (r *FilterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_filter"
}

func (r *FilterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hookbase filter.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Filter ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Filter name.",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "URL-safe identifier. Auto-generated from name if not provided. Changing this forces a new resource.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				Description: "Filter description.",
				Optional:    true,
			},
			"conditions": schema.StringAttribute{
				Description: "JSON string of conditions array. Each condition has field, operator, and value.",
				Required:    true,
			},
			"logic": schema.StringAttribute{
				Description: "Logic operator for combining conditions: AND or OR.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("AND"),
			},
			"created_at": schema.StringAttribute{
				Description: "Creation timestamp.",
				Computed:    true,
			},
			"updated_at": schema.StringAttribute{
				Description: "Last update timestamp.",
				Computed:    true,
			},
		},
	}
}

func (r *FilterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", "Expected *client.Client")
		return
	}
	r.client = c
}

func (r *FilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan FilterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateFilterRequest{
		Name:       plan.Name.ValueString(),
		Conditions: json.RawMessage(plan.Conditions.ValueString()),
	}

	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		createReq.Slug = plan.Slug.ValueString()
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		createReq.Description = &v
	}
	if !plan.Logic.IsNull() {
		v := plan.Logic.ValueString()
		createReq.Logic = &v
	}

	filter, err := r.client.CreateFilter(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating filter", err.Error())
		return
	}

	// Re-read to get full object with timestamps
	filter, err = r.client.GetFilter(ctx, filter.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading filter after create", err.Error())
		return
	}

	mapFilterToState(filter, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state FilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	filter, err := r.client.GetFilter(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading filter", err.Error())
		return
	}

	mapFilterToState(filter, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *FilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan FilterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateFilterRequest{}

	if !plan.Name.IsNull() {
		v := plan.Name.ValueString()
		updateReq.Name = &v
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		updateReq.Description = &v
	}
	if !plan.Conditions.IsNull() {
		updateReq.Conditions = json.RawMessage(plan.Conditions.ValueString())
	}
	if !plan.Logic.IsNull() {
		v := plan.Logic.ValueString()
		updateReq.Logic = &v
	}

	filter, err := r.client.UpdateFilter(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating filter", err.Error())
		return
	}

	mapFilterToState(filter, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state FilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteFilter(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting filter", err.Error())
	}
}

func (r *FilterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapFilterToState(filter *client.Filter, state *FilterResourceModel, _ *diag.Diagnostics) {
	state.ID = types.StringValue(filter.ID)
	state.Name = types.StringValue(filter.Name)
	state.Slug = types.StringValue(filter.Slug)
	state.Description = stringPtrToValue(filter.Description)

	// Normalize conditions JSON: unwrap double-encoding and re-serialize for consistent ordering
	conditions := filter.Conditions
	// Unwrap if double-encoded (DB returns string-within-JSON)
	if len(conditions) > 0 && conditions[0] == '"' {
		var unwrapped string
		if json.Unmarshal(conditions, &unwrapped) == nil {
			conditions = json.RawMessage(unwrapped)
		}
	}
	// Re-parse and re-marshal to normalize key order and spacing
	var parsed interface{}
	if json.Unmarshal(conditions, &parsed) == nil {
		if normalized, err := json.Marshal(parsed); err == nil {
			conditions = normalized
		}
	}
	state.Conditions = types.StringValue(string(conditions))

	state.Logic = types.StringValue(filter.Logic)
	if filter.CreatedAt != "" {
		state.CreatedAt = types.StringValue(filter.CreatedAt)
	}
	if filter.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(filter.UpdatedAt)
	}
}
