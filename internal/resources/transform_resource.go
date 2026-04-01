package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var (
	_ resource.Resource                = &TransformResource{}
	_ resource.ResourceWithImportState = &TransformResource{}
)

type TransformResource struct {
	client *client.Client
}

type TransformResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Slug          types.String `tfsdk:"slug"`
	Description   types.String `tfsdk:"description"`
	Code          types.String `tfsdk:"code"`
	TransformType types.String `tfsdk:"transform_type"`
	InputFormat   types.String `tfsdk:"input_format"`
	OutputFormat  types.String `tfsdk:"output_format"`
	IsActive      types.Bool   `tfsdk:"is_active"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func NewTransformResource() resource.Resource {
	return &TransformResource{}
}

func (r *TransformResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transform"
}

func (r *TransformResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hookbase transform.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Transform ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Transform name.",
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
				Description: "Transform description.",
				Optional:    true,
			},
			"code": schema.StringAttribute{
				Description: "Transform code.",
				Required:    true,
			},
			"transform_type": schema.StringAttribute{
				Description: "Transform type: jsonata, xslt, liquid, or javascript.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("jsonata"),
			},
			"input_format": schema.StringAttribute{
				Description: "Input format: json, xml, or text.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("json"),
			},
			"output_format": schema.StringAttribute{
				Description: "Output format: json, xml, or text.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("json"),
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether the transform is active.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
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

func (r *TransformResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TransformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TransformResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateTransformRequest{
		Name: plan.Name.ValueString(),
		Code: plan.Code.ValueString(),
	}

	if !plan.Slug.IsNull() && !plan.Slug.IsUnknown() {
		createReq.Slug = plan.Slug.ValueString()
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		createReq.Description = &v
	}
	if !plan.TransformType.IsNull() {
		v := plan.TransformType.ValueString()
		createReq.TransformType = &v
	}
	if !plan.InputFormat.IsNull() {
		v := plan.InputFormat.ValueString()
		createReq.InputFormat = &v
	}
	if !plan.OutputFormat.IsNull() {
		v := plan.OutputFormat.ValueString()
		createReq.OutputFormat = &v
	}

	transform, err := r.client.CreateTransform(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating transform", err.Error())
		return
	}

	// Re-read to get full object with timestamps
	transform, err = r.client.GetTransform(ctx, transform.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading transform after create", err.Error())
		return
	}

	mapTransformToState(transform, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TransformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state TransformResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	transform, err := r.client.GetTransform(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading transform", err.Error())
		return
	}

	mapTransformToState(transform, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *TransformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan TransformResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateTransformRequest{}

	if !plan.Name.IsNull() {
		v := plan.Name.ValueString()
		updateReq.Name = &v
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		updateReq.Description = &v
	}
	if !plan.Code.IsNull() {
		v := plan.Code.ValueString()
		updateReq.Code = &v
	}
	if !plan.InputFormat.IsNull() {
		v := plan.InputFormat.ValueString()
		updateReq.InputFormat = &v
	}
	if !plan.OutputFormat.IsNull() {
		v := plan.OutputFormat.ValueString()
		updateReq.OutputFormat = &v
	}

	transform, err := r.client.UpdateTransform(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating transform", err.Error())
		return
	}

	mapTransformToState(transform, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *TransformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TransformResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteTransform(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting transform", err.Error())
	}
}

func (r *TransformResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapTransformToState(transform *client.Transform, state *TransformResourceModel, _ *diag.Diagnostics) {
	state.ID = types.StringValue(transform.ID)
	state.Name = types.StringValue(transform.Name)
	state.Slug = types.StringValue(transform.Slug)
	state.Description = stringPtrToValue(transform.Description)
	state.Code = types.StringValue(transform.Code)
	state.TransformType = types.StringValue(transform.GetTransformType())
	state.InputFormat = types.StringValue(transform.GetInputFormat())
	state.OutputFormat = types.StringValue(transform.GetOutputFormat())
	if transform.CreatedAt != "" {
		state.CreatedAt = types.StringValue(transform.CreatedAt)
	}
	if transform.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(transform.UpdatedAt)
	}
}
