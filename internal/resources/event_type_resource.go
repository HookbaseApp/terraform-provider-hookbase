package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var (
	_ resource.Resource                = &EventTypeResource{}
	_ resource.ResourceWithImportState = &EventTypeResource{}
)

type EventTypeResource struct {
	client *client.Client
}

type EventTypeResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	DisplayName       types.String `tfsdk:"display_name"`
	Description       types.String `tfsdk:"description"`
	Category          types.String `tfsdk:"category"`
	Schema            types.String `tfsdk:"schema"`
	ExamplePayload    types.String `tfsdk:"example_payload"`
	DocumentationURL  types.String `tfsdk:"documentation_url"`
	IsEnabled         types.Bool   `tfsdk:"is_enabled"`
	IsDeprecated      types.Bool   `tfsdk:"is_deprecated"`
	DeprecatedMessage types.String `tfsdk:"deprecated_message"`
	DefaultPriority   types.Int64  `tfsdk:"default_priority"`
	SchemaVersion     types.Int64  `tfsdk:"schema_version"`
	CreatedAt         types.String `tfsdk:"created_at"`
	UpdatedAt         types.String `tfsdk:"updated_at"`
}

func NewEventTypeResource() resource.Resource {
	return &EventTypeResource{}
}

func (r *EventTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_event_type"
}

func (r *EventTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hookbase event type.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Event type ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Event type name (e.g. order.created). Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"display_name": schema.StringAttribute{
				Description: "Human-readable display name. Auto-generated from name if omitted.",
				Optional:    true,
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: "Event type description.",
				Optional:    true,
			},
			"category": schema.StringAttribute{
				Description: "Event type category for grouping.",
				Optional:    true,
			},
			"schema": schema.StringAttribute{
				Description: "JSON Schema for payload validation.",
				Optional:    true,
			},
			"example_payload": schema.StringAttribute{
				Description: "Example payload as JSON string.",
				Optional:    true,
			},
			"documentation_url": schema.StringAttribute{
				Description: "URL to external documentation.",
				Optional:    true,
			},
			"is_enabled": schema.BoolAttribute{
				Description: "Whether the event type is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"is_deprecated": schema.BoolAttribute{
				Description: "Whether the event type is deprecated.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"deprecated_message": schema.StringAttribute{
				Description: "Deprecation message shown to consumers.",
				Optional:    true,
			},
			"default_priority": schema.Int64Attribute{
				Description: "Default delivery priority (0=Critical, 1=High, 2=Normal, 3=Low, 4=Bulk).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"schema_version": schema.Int64Attribute{
				Description: "Auto-incremented schema version.",
				Computed:    true,
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

func (r *EventTypeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *EventTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EventTypeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateEventTypeRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.DisplayName.IsNull() {
		v := plan.DisplayName.ValueString()
		createReq.DisplayName = &v
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		createReq.Description = &v
	}
	if !plan.Category.IsNull() {
		v := plan.Category.ValueString()
		createReq.Category = &v
	}
	if !plan.Schema.IsNull() {
		v := plan.Schema.ValueString()
		createReq.Schema = &v
	}
	if !plan.ExamplePayload.IsNull() {
		v := plan.ExamplePayload.ValueString()
		createReq.ExamplePayload = &v
	}
	if !plan.DocumentationURL.IsNull() {
		v := plan.DocumentationURL.ValueString()
		createReq.DocumentationURL = &v
	}
	if !plan.IsEnabled.IsNull() {
		v := plan.IsEnabled.ValueBool()
		createReq.IsEnabled = &v
	}
	if !plan.DefaultPriority.IsNull() {
		v := int(plan.DefaultPriority.ValueInt64())
		createReq.DefaultPriority = &v
	}

	eventType, err := r.client.CreateEventType(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating event type", err.Error())
		return
	}

	// Re-read to get full object with timestamps
	eventType, err = r.client.GetEventType(ctx, eventType.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading event type after create", err.Error())
		return
	}

	mapEventTypeToState(eventType, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EventTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state EventTypeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	eventType, err := r.client.GetEventType(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading event type", err.Error())
		return
	}

	mapEventTypeToState(eventType, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *EventTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan EventTypeResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateEventTypeRequest{}

	if !plan.DisplayName.IsNull() {
		v := plan.DisplayName.ValueString()
		updateReq.DisplayName = &v
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		updateReq.Description = &v
	}
	if !plan.Category.IsNull() {
		v := plan.Category.ValueString()
		updateReq.Category = &v
	}
	if !plan.Schema.IsNull() {
		v := plan.Schema.ValueString()
		updateReq.Schema = &v
	}
	if !plan.ExamplePayload.IsNull() {
		v := plan.ExamplePayload.ValueString()
		updateReq.ExamplePayload = &v
	}
	if !plan.DocumentationURL.IsNull() {
		v := plan.DocumentationURL.ValueString()
		updateReq.DocumentationURL = &v
	}
	if !plan.IsEnabled.IsNull() {
		v := plan.IsEnabled.ValueBool()
		updateReq.IsEnabled = &v
	}
	if !plan.IsDeprecated.IsNull() {
		v := plan.IsDeprecated.ValueBool()
		updateReq.IsDeprecated = &v
	}
	if !plan.DeprecatedMessage.IsNull() {
		v := plan.DeprecatedMessage.ValueString()
		updateReq.DeprecatedMessage = &v
	}
	if !plan.DefaultPriority.IsNull() {
		v := int(plan.DefaultPriority.ValueInt64())
		updateReq.DefaultPriority = &v
	}

	eventType, err := r.client.UpdateEventType(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating event type", err.Error())
		return
	}

	mapEventTypeToState(eventType, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *EventTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EventTypeResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteEventType(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting event type", err.Error())
	}
}

func (r *EventTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapEventTypeToState(et *client.EventType, state *EventTypeResourceModel, _ *diag.Diagnostics) {
	state.ID = types.StringValue(et.ID)
	state.Name = types.StringValue(et.Name)
	state.DisplayName = types.StringValue(et.DisplayName)
	state.Description = stringPtrToValue(et.Description)
	state.Category = stringPtrToValue(et.Category)
	state.Schema = stringPtrToValue(et.Schema)
	state.ExamplePayload = stringPtrToValue(et.ExamplePayload)
	state.DocumentationURL = stringPtrToValue(et.DocumentationURL)
	state.IsEnabled = types.BoolValue(et.IsEnabled)
	state.IsDeprecated = types.BoolValue(et.IsDeprecated)
	state.DeprecatedMessage = stringPtrToValue(et.DeprecatedMessage)
	state.DefaultPriority = types.Int64Value(int64(et.DefaultPriority))
	state.SchemaVersion = types.Int64Value(int64(et.SchemaVersion))
	if et.CreatedAt != "" {
		state.CreatedAt = types.StringValue(et.CreatedAt)
	}
	if et.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(et.UpdatedAt)
	}
}
