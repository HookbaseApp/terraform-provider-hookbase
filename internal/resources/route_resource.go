package resources

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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
	HoldOnCircuitOpen            types.Bool   `tfsdk:"hold_on_circuit_open"`
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
				Description: "Inline filter conditions as a JSON array (use jsonencode). Key order and " +
					"formatting are ignored when comparing with the API's copy.",
				Optional: true,
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
				Description: "Comma-separated email addresses that receive the route's notifications, for " +
					"example \"oncall@example.com, team@example.com\". Each entry must be an email address. " +
					"The API trims the entries, keeps only the first of an address listed twice (in any " +
					"letter case), and returns the list joined with \", \"; that shows no diff as long as " +
					"it lists the same addresses in the same order as the configured value. Removing the " +
					"attribute clears the list.",
				Optional: true,
				Validators: []validator.String{
					notifyEmailsValidator{},
				},
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
				Description: "Expected response configuration as a JSON object (use jsonencode). Key order " +
					"and formatting are ignored when comparing with the API's copy.",
				Optional: true,
			},
			"circuit_cooldown_seconds": schema.Int64Attribute{
				Description: "Seconds to wait before probing after the circuit opens (30-3600). Changing a " +
					"circuit setting on an existing route needs the circuit breaker plan feature. The API " +
					"cannot clear these settings, so removing one from configuration keeps its current value.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"circuit_probe_success_threshold": schema.Int64Attribute{
				Description: "Number of successful probes to close the circuit (1-10). See circuit_cooldown_seconds.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"circuit_failure_threshold": schema.Int64Attribute{
				Description: "Number of failures to open the circuit (1-50). See circuit_cooldown_seconds.",
				Optional:    true,
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"hold_on_circuit_open": schema.BoolAttribute{
				Description: "While the route's circuit is open, hold its deliveries and send them once the " +
					"destination recovers, instead of dead-lettering them. When not set, the route keeps its " +
					"current value (a new route gets the API default, true), so a value changed outside " +
					"Terraform is left alone. Turning it off does not release deliveries already held; they " +
					"are dead-lettered on their next attempt while the circuit stays open.",
				Optional: true,
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
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
		createReq.NotifyEmails = notifyEmailsBody(plan.NotifyEmails.ValueString())
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
	createReq.CircuitCooldownSeconds = knownInt(plan.CircuitCooldownSeconds)
	createReq.CircuitProbeSuccessThreshold = knownInt(plan.CircuitProbeSuccessThreshold)
	createReq.CircuitFailureThreshold = knownInt(plan.CircuitFailureThreshold)
	// Sent only when configured; otherwise the API's default applies and is read back.
	createReq.HoldOnCircuitOpen = configuredBool(ctx, req.Config, path.Root("hold_on_circuit_open"), &resp.Diagnostics)

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
	var plan, state RouteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := buildUpdateRouteRequest(ctx, &plan, &state)
	resp.Diagnostics.Append(diags...)
	// Sent only when configured, so a value changed outside Terraform is kept.
	updateReq.HoldOnCircuitOpen = configuredBool(ctx, req.Config, path.Root("hold_on_circuit_open"), &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// PATCH ignores the circuit settings; they have an endpoint of their own, which needs the
	// circuit_breaker feature. It goes first, and only when a circuit setting changed, so that a
	// refusal fails the apply before any other change is made.
	if circuit, changed := routeCircuitConfigChange(&plan, &state); changed {
		if err := r.client.UpdateRouteCircuitConfig(ctx, plan.ID.ValueString(), circuit); err != nil {
			resp.Diagnostics.AddError("Error updating route circuit breaker settings", err.Error())
			return
		}
	}

	// PATCH answers {success:true} only, so the reply carries nothing to map.
	if err := r.client.UpdateRoute(ctx, plan.ID.ValueString(), updateReq); err != nil {
		resp.Diagnostics.AddError("Error updating route", err.Error())
		return
	}

	route, err := r.client.GetRoute(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading route after update", err.Error())
		return
	}

	plannedHold := plan.HoldOnCircuitOpen
	mapRouteToState(ctx, route, &plan, &resp.Diagnostics)
	// A known planned value must be the applied one. It is null when the state predates the
	// attribute and was not refreshed; the next refresh fills it in.
	if !plannedHold.IsUnknown() {
		plan.HoldOnCircuitOpen = plannedHold
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// buildUpdateRouteRequest builds the PATCH body, apart from hold_on_circuit_open, which Update adds
// from configuration. Settings with a default (priority, the notify flags, thresholds) are always
// sent. Optional ones are sent when they changed, with the API's clear value when removed from
// configuration. The wiring fields (source_id, destination_id, transform_id,
// failover_destination_ids) are sent only when they changed: the API checks the events:send
// permission whenever one is given for a route that feeds a bridge destination, and a rename must
// not need it.
func buildUpdateRouteRequest(ctx context.Context, plan, state *RouteResourceModel) (client.UpdateRouteRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	updateReq := client.UpdateRouteRequest{
		Name:                  knownString(plan.Name),
		Priority:              knownInt(plan.Priority),
		IsActive:              knownBool(plan.IsActive),
		NotifyOnFailure:       knownBool(plan.NotifyOnFailure),
		NotifyOnSuccess:       knownBool(plan.NotifyOnSuccess),
		NotifyOnRecovery:      knownBool(plan.NotifyOnRecovery),
		FailureThreshold:      knownInt(plan.FailureThreshold),
		FailoverAfterAttempts: knownInt(plan.FailoverAfterAttempts),
	}
	if !plan.SourceID.Equal(state.SourceID) {
		updateReq.SourceID = knownString(plan.SourceID)
	}
	if !plan.DestinationID.Equal(state.DestinationID) {
		updateReq.DestinationID = knownString(plan.DestinationID)
	}

	// changedOrCleared is the value to send for an optional string: nil when unchanged, "" when
	// removed from configuration (the API's clear value for these).
	changedOrCleared := func(p, s types.String) *string {
		if p.Equal(s) || p.IsUnknown() {
			return nil
		}
		if p.IsNull() {
			empty := ""
			return &empty
		}
		return knownString(p)
	}
	updateReq.FilterID = changedOrCleared(plan.FilterID, state.FilterID)
	updateReq.TransformID = changedOrCleared(plan.TransformID, state.TransformID)
	updateReq.SchemaID = changedOrCleared(plan.SchemaID, state.SchemaID)
	if v := changedOrCleared(plan.NotifyEmails, state.NotifyEmails); v != nil {
		updateReq.NotifyEmails = notifyEmailsBody(*v)
	}

	// The API applies filterLogic only together with filterConditions, so a change to either sends
	// both.
	if !plan.FilterConditions.IsUnknown() &&
		(!plan.FilterConditions.Equal(state.FilterConditions) || !plan.FilterLogic.Equal(state.FilterLogic)) {
		if plan.FilterConditions.IsNull() {
			updateReq.FilterConditions = json.RawMessage("[]")
		} else {
			updateReq.FilterConditions = json.RawMessage(plan.FilterConditions.ValueString())
			updateReq.FilterLogic = knownString(plan.FilterLogic)
		}
	}

	if !plan.ExpectedResponse.Equal(state.ExpectedResponse) && !plan.ExpectedResponse.IsUnknown() {
		if plan.ExpectedResponse.IsNull() {
			updateReq.ExpectedResponse = json.RawMessage("null")
		} else {
			updateReq.ExpectedResponse = json.RawMessage(plan.ExpectedResponse.ValueString())
		}
	}

	if !plan.FailoverDestinationIds.Equal(state.FailoverDestinationIds) && !plan.FailoverDestinationIds.IsUnknown() {
		ids := []string{}
		diags.Append(stringListToSlice(ctx, plan.FailoverDestinationIds, &ids)...)
		updateReq.FailoverDestinationIds = &ids
	}
	return updateReq, diags
}

// routeCircuitConfigChange returns the circuit settings that changed to a new value. A setting
// removed from configuration plans as its current value (UseStateForUnknown), since the API has
// no way to clear one.
func routeCircuitConfigChange(plan, state *RouteResourceModel) (client.RouteCircuitConfig, bool) {
	var cfg client.RouteCircuitConfig
	changed := false
	pick := func(p, s types.Int64) *int {
		if p.IsNull() || p.IsUnknown() || p.Equal(s) {
			return nil
		}
		changed = true
		return knownInt(p)
	}
	cfg.CircuitCooldownSeconds = pick(plan.CircuitCooldownSeconds, state.CircuitCooldownSeconds)
	cfg.CircuitProbeSuccessThreshold = pick(plan.CircuitProbeSuccessThreshold, state.CircuitProbeSuccessThreshold)
	cfg.CircuitFailureThreshold = pick(plan.CircuitFailureThreshold, state.CircuitFailureThreshold)
	return cfg, changed
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
	switch {
	case route.FilterLogic != nil:
		state.FilterLogic = types.StringValue(*route.FilterLogic)
	case !state.FilterLogic.IsNull() && !state.FilterLogic.IsUnknown():
		// The API stores no logic for a route without inline conditions; keep the configured one.
	default:
		state.FilterLogic = types.StringValue("AND")
	}
	state.TransformID = stringPtrToValue(route.TransformID)
	state.SchemaID = stringPtrToValue(route.SchemaID)
	state.Priority = types.Int64Value(int64(route.Priority))
	state.IsActive = types.BoolValue(route.IsActive)
	state.NotifyOnFailure = types.BoolValue(route.NotifyOnFailure)
	state.NotifyOnSuccess = types.BoolValue(route.NotifyOnSuccess)
	state.NotifyOnRecovery = types.BoolValue(route.NotifyOnRecovery)

	state.NotifyEmails = notifyEmailsToState(state.NotifyEmails, route.NotifyEmails)

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

	// JSON fields. GET answers filterConditions as its stored JSON text and expectedResponse
	// parsed; jsonToState keeps the configured text whenever it holds the same value.
	state.FilterConditions = jsonToState(state.FilterConditions, route.FilterConditions, false)
	state.ExpectedResponse = jsonToState(state.ExpectedResponse, route.ExpectedResponse, false)

	state.FailoverDestinationIds = listToState(ctx, route.FailoverDestinationIds, state.FailoverDestinationIds, diags)
	state.HoldOnCircuitOpen = types.BoolValue(route.HoldOnCircuitOpen == nil || *route.HoldOnCircuitOpen)

	if route.CreatedAt != "" {
		state.CreatedAt = types.StringValue(route.CreatedAt)
	}
	if route.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(route.UpdatedAt)
	}
}
