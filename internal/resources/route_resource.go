package resources

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var (
	_ resource.Resource                = &RouteResource{}
	_ resource.ResourceWithImportState = &RouteResource{}
)

type RouteResource struct {
	client *client.Client
}

type RouteResourceModel struct {
	ID                           types.String `tfsdk:"id"`
	Name                         types.String `tfsdk:"name"`
	SourceID                     types.String `tfsdk:"source_id"`
	DestinationID                types.String `tfsdk:"destination_id"`
	FilterID                     types.String `tfsdk:"filter_id"`
	FilterConditions             types.String `tfsdk:"filter_conditions"`
	FilterLogic                  types.String `tfsdk:"filter_logic"`
	TransformID                  types.String `tfsdk:"transform_id"`
	SchemaID                     types.String `tfsdk:"schema_id"`
	Priority                     types.Int64  `tfsdk:"priority"`
	IsActive                     types.Bool   `tfsdk:"is_active"`
	NotifyOnFailure              types.Bool   `tfsdk:"notify_on_failure"`
	NotifyOnSuccess              types.Bool   `tfsdk:"notify_on_success"`
	NotifyOnRecovery             types.Bool   `tfsdk:"notify_on_recovery"`
	NotifyEmails                 types.String `tfsdk:"notify_emails"`
	FailureThreshold             types.Int64  `tfsdk:"failure_threshold"`
	FailoverDestinationIds       types.List   `tfsdk:"failover_destination_ids"`
	FailoverAfterAttempts        types.Int64  `tfsdk:"failover_after_attempts"`
	ExpectedResponse             types.String `tfsdk:"expected_response"`
	CircuitCooldownSeconds       types.Int64  `tfsdk:"circuit_cooldown_seconds"`
	CircuitProbeSuccessThreshold types.Int64  `tfsdk:"circuit_probe_success_threshold"`
	CircuitFailureThreshold      types.Int64  `tfsdk:"circuit_failure_threshold"`
	CreatedAt                    types.String `tfsdk:"created_at"`
	UpdatedAt                    types.String `tfsdk:"updated_at"`
}

func NewRouteResource() resource.Resource {
	return &RouteResource{}
}

func (r *RouteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route"
}

func (r *RouteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hookbase route connecting a source to a destination.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Route ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Route name (1-100 characters).",
				Required:    true,
			},
			"source_id": schema.StringAttribute{
				Description: "ID of the source this route listens to.",
				Required:    true,
			},
			"destination_id": schema.StringAttribute{
				Description: "ID of the destination this route delivers to.",
				Required:    true,
			},
			"filter_id": schema.StringAttribute{
				Description: "ID of a saved filter to apply.",
				Optional:    true,
			},
			"filter_conditions": schema.StringAttribute{
				Description: "Inline filter conditions as a JSON string.",
				Optional:    true,
			},
			"filter_logic": schema.StringAttribute{
				Description: "Logic for combining filter conditions: AND or OR.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("AND"),
			},
			"transform_id": schema.StringAttribute{
				Description: "ID of a transform to apply before delivery.",
				Optional:    true,
			},
			"schema_id": schema.StringAttribute{
				Description: "ID of a schema to validate events against.",
				Optional:    true,
			},
			"priority": schema.Int64Attribute{
				Description: "Route priority. Lower values are evaluated first.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether the route is active.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"notify_on_failure": schema.BoolAttribute{
				Description: "Send notification on delivery failure.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"notify_on_success": schema.BoolAttribute{
				Description: "Send notification on delivery success.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"notify_on_recovery": schema.BoolAttribute{
				Description: "Send notification when a failing route recovers.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"notify_emails": schema.StringAttribute{
				Description: "Comma-separated email addresses for notifications.",
				Optional:    true,
			},
			"failure_threshold": schema.Int64Attribute{
				Description: "Number of consecutive failures before triggering a notification.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(3),
			},
			"failover_destination_ids": schema.ListAttribute{
				Description: "IDs of destinations to failover to.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"failover_after_attempts": schema.Int64Attribute{
				Description: "Number of delivery attempts before failing over.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5),
			},
			"expected_response": schema.StringAttribute{
				Description: "Expected response configuration as a JSON string.",
				Optional:    true,
			},
			"circuit_cooldown_seconds": schema.Int64Attribute{
				Description: "Seconds to wait before probing after circuit opens.",
				Optional:    true,
			},
			"circuit_probe_success_threshold": schema.Int64Attribute{
				Description: "Number of successful probes to close the circuit.",
				Optional:    true,
			},
			"circuit_failure_threshold": schema.Int64Attribute{
				Description: "Number of failures to open the circuit.",
				Optional:    true,
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

func (r *RouteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RouteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateRouteRequest{
		Name:          plan.Name.ValueString(),
		SourceID:      plan.SourceID.ValueString(),
		DestinationID: plan.DestinationID.ValueString(),
	}

	if !plan.FilterID.IsNull() {
		v := plan.FilterID.ValueString()
		createReq.FilterID = &v
	}
	if !plan.FilterConditions.IsNull() {
		createReq.FilterConditions = json.RawMessage(plan.FilterConditions.ValueString())
	}
	if !plan.FilterLogic.IsNull() {
		v := plan.FilterLogic.ValueString()
		createReq.FilterLogic = &v
	}
	if !plan.TransformID.IsNull() {
		v := plan.TransformID.ValueString()
		createReq.TransformID = &v
	}
	if !plan.SchemaID.IsNull() {
		v := plan.SchemaID.ValueString()
		createReq.SchemaID = &v
	}
	if !plan.Priority.IsNull() {
		v := int(plan.Priority.ValueInt64())
		createReq.Priority = &v
	}
	if !plan.NotifyOnFailure.IsNull() {
		v := plan.NotifyOnFailure.ValueBool()
		createReq.NotifyOnFailure = &v
	}
	if !plan.NotifyOnSuccess.IsNull() {
		v := plan.NotifyOnSuccess.ValueBool()
		createReq.NotifyOnSuccess = &v
	}
	if !plan.NotifyOnRecovery.IsNull() {
		v := plan.NotifyOnRecovery.ValueBool()
		createReq.NotifyOnRecovery = &v
	}
	if !plan.NotifyEmails.IsNull() {
		v := plan.NotifyEmails.ValueString()
		createReq.NotifyEmails = &v
	}
	if !plan.FailureThreshold.IsNull() {
		v := int(plan.FailureThreshold.ValueInt64())
		createReq.FailureThreshold = &v
	}
	if !plan.FailoverAfterAttempts.IsNull() {
		v := int(plan.FailoverAfterAttempts.ValueInt64())
		createReq.FailoverAfterAttempts = &v
	}
	if !plan.ExpectedResponse.IsNull() {
		createReq.ExpectedResponse = json.RawMessage(plan.ExpectedResponse.ValueString())
	}
	if !plan.CircuitCooldownSeconds.IsNull() {
		v := int(plan.CircuitCooldownSeconds.ValueInt64())
		createReq.CircuitCooldownSeconds = &v
	}
	if !plan.CircuitProbeSuccessThreshold.IsNull() {
		v := int(plan.CircuitProbeSuccessThreshold.ValueInt64())
		createReq.CircuitProbeSuccessThreshold = &v
	}
	if !plan.CircuitFailureThreshold.IsNull() {
		v := int(plan.CircuitFailureThreshold.ValueInt64())
		createReq.CircuitFailureThreshold = &v
	}

	resp.Diagnostics.Append(stringListToSlice(ctx, plan.FailoverDestinationIds, &createReq.FailoverDestinationIds)...)
	if resp.Diagnostics.HasError() {
		return
	}

	route, err := r.client.CreateRoute(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating route", err.Error())
		return
	}

	// Re-read to get full object with timestamps
	route, err = r.client.GetRoute(ctx, route.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading route after create", err.Error())
		return
	}

	mapRouteToState(ctx, route, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RouteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state RouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	route, err := r.client.GetRoute(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading route", err.Error())
		return
	}

	mapRouteToState(ctx, route, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *RouteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan RouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateRouteRequest{}

	if !plan.Name.IsNull() {
		v := plan.Name.ValueString()
		updateReq.Name = &v
	}
	if !plan.SourceID.IsNull() {
		v := plan.SourceID.ValueString()
		updateReq.SourceID = &v
	}
	if !plan.DestinationID.IsNull() {
		v := plan.DestinationID.ValueString()
		updateReq.DestinationID = &v
	}
	if !plan.FilterID.IsNull() {
		v := plan.FilterID.ValueString()
		updateReq.FilterID = &v
	}
	if !plan.FilterConditions.IsNull() {
		updateReq.FilterConditions = json.RawMessage(plan.FilterConditions.ValueString())
	}
	if !plan.FilterLogic.IsNull() {
		v := plan.FilterLogic.ValueString()
		updateReq.FilterLogic = &v
	}
	if !plan.TransformID.IsNull() {
		v := plan.TransformID.ValueString()
		updateReq.TransformID = &v
	}
	if !plan.SchemaID.IsNull() {
		v := plan.SchemaID.ValueString()
		updateReq.SchemaID = &v
	}
	if !plan.Priority.IsNull() {
		v := int(plan.Priority.ValueInt64())
		updateReq.Priority = &v
	}
	if !plan.IsActive.IsNull() {
		v := plan.IsActive.ValueBool()
		updateReq.IsActive = &v
	}
	if !plan.NotifyOnFailure.IsNull() {
		v := plan.NotifyOnFailure.ValueBool()
		updateReq.NotifyOnFailure = &v
	}
	if !plan.NotifyOnSuccess.IsNull() {
		v := plan.NotifyOnSuccess.ValueBool()
		updateReq.NotifyOnSuccess = &v
	}
	if !plan.NotifyOnRecovery.IsNull() {
		v := plan.NotifyOnRecovery.ValueBool()
		updateReq.NotifyOnRecovery = &v
	}
	if !plan.NotifyEmails.IsNull() {
		v := plan.NotifyEmails.ValueString()
		updateReq.NotifyEmails = &v
	}
	if !plan.FailureThreshold.IsNull() {
		v := int(plan.FailureThreshold.ValueInt64())
		updateReq.FailureThreshold = &v
	}
	if !plan.FailoverAfterAttempts.IsNull() {
		v := int(plan.FailoverAfterAttempts.ValueInt64())
		updateReq.FailoverAfterAttempts = &v
	}
	if !plan.ExpectedResponse.IsNull() {
		updateReq.ExpectedResponse = json.RawMessage(plan.ExpectedResponse.ValueString())
	}
	if !plan.CircuitCooldownSeconds.IsNull() {
		v := int(plan.CircuitCooldownSeconds.ValueInt64())
		updateReq.CircuitCooldownSeconds = &v
	}
	if !plan.CircuitProbeSuccessThreshold.IsNull() {
		v := int(plan.CircuitProbeSuccessThreshold.ValueInt64())
		updateReq.CircuitProbeSuccessThreshold = &v
	}
	if !plan.CircuitFailureThreshold.IsNull() {
		v := int(plan.CircuitFailureThreshold.ValueInt64())
		updateReq.CircuitFailureThreshold = &v
	}

	resp.Diagnostics.Append(stringListToSlice(ctx, plan.FailoverDestinationIds, &updateReq.FailoverDestinationIds)...)
	if resp.Diagnostics.HasError() {
		return
	}

	route, err := r.client.UpdateRoute(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating route", err.Error())
		return
	}

	mapRouteToState(ctx, route, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *RouteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state RouteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteRoute(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting route", err.Error())
	}
}

func (r *RouteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapRouteToState(ctx context.Context, route *client.Route, state *RouteResourceModel, diags *diag.Diagnostics) {
	state.ID = types.StringValue(route.ID)
	state.Name = types.StringValue(route.Name)
	state.SourceID = types.StringValue(route.SourceID)
	state.DestinationID = types.StringValue(route.DestinationID)
	state.FilterID = stringPtrToValue(route.FilterID)
	if route.FilterLogic != nil {
		state.FilterLogic = types.StringValue(*route.FilterLogic)
	} else {
		state.FilterLogic = types.StringValue("AND")
	}
	state.TransformID = stringPtrToValue(route.TransformID)
	state.SchemaID = stringPtrToValue(route.SchemaID)
	state.Priority = types.Int64Value(int64(route.Priority))
	state.IsActive = types.BoolValue(route.IsActive)
	state.NotifyOnFailure = types.BoolValue(route.NotifyOnFailure)
	state.NotifyOnSuccess = types.BoolValue(route.NotifyOnSuccess)
	state.NotifyOnRecovery = types.BoolValue(route.NotifyOnRecovery)

	if route.NotifyEmails != nil && *route.NotifyEmails != "" {
		state.NotifyEmails = types.StringValue(*route.NotifyEmails)
	} else {
		state.NotifyEmails = types.StringNull()
	}

	if route.FailureThreshold != nil {
		state.FailureThreshold = types.Int64Value(int64(*route.FailureThreshold))
	} else {
		state.FailureThreshold = types.Int64Value(3)
	}

	if route.FailoverAfterAttempts != nil {
		state.FailoverAfterAttempts = types.Int64Value(int64(*route.FailoverAfterAttempts))
	} else {
		state.FailoverAfterAttempts = types.Int64Value(5)
	}

	if route.CircuitCooldownSeconds != nil {
		state.CircuitCooldownSeconds = types.Int64Value(int64(*route.CircuitCooldownSeconds))
	} else {
		state.CircuitCooldownSeconds = types.Int64Null()
	}

	if route.CircuitProbeSuccessThreshold != nil {
		state.CircuitProbeSuccessThreshold = types.Int64Value(int64(*route.CircuitProbeSuccessThreshold))
	} else {
		state.CircuitProbeSuccessThreshold = types.Int64Null()
	}

	if route.CircuitFailureThreshold != nil {
		state.CircuitFailureThreshold = types.Int64Value(int64(*route.CircuitFailureThreshold))
	} else {
		state.CircuitFailureThreshold = types.Int64Null()
	}

	// FilterConditions: JSON field
	if route.FilterConditions != nil && len(route.FilterConditions) > 0 && string(route.FilterConditions) != "null" {
		state.FilterConditions = types.StringValue(string(route.FilterConditions))
	} else {
		state.FilterConditions = types.StringNull()
	}

	// ExpectedResponse: JSON field
	if route.ExpectedResponse != nil && len(route.ExpectedResponse) > 0 && string(route.ExpectedResponse) != "null" {
		state.ExpectedResponse = types.StringValue(string(route.ExpectedResponse))
	} else {
		state.ExpectedResponse = types.StringNull()
	}

	state.FailoverDestinationIds = sliceToStringList(ctx, route.FailoverDestinationIds, diags)

	if route.CreatedAt != "" {
		state.CreatedAt = types.StringValue(route.CreatedAt)
	}
	if route.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(route.UpdatedAt)
	}
}
