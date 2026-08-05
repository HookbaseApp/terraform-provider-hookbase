package resources

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var (
	_ resource.Resource                = &DestinationResource{}
	_ resource.ResourceWithImportState = &DestinationResource{}
)

type DestinationResource struct {
	client *client.Client
}

type DestinationResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Slug                   types.String `tfsdk:"slug"`
	URL                    types.String `tfsdk:"url"`
	Method                 types.String `tfsdk:"method"`
	Headers                types.Map    `tfsdk:"headers"`
	AuthType               types.String `tfsdk:"auth_type"`
	AuthConfig             types.Map    `tfsdk:"auth_config"`
	TimeoutMs              types.Int64  `tfsdk:"timeout_ms"`
	ThrottleMode           types.String `tfsdk:"throttle_mode"`
	ThrottleRateLimit      types.Int64  `tfsdk:"throttle_rate_limit"`
	ThrottleRateUnit       types.String `tfsdk:"throttle_rate_unit"`
	ThrottleMaxConcurrency types.Int64  `tfsdk:"throttle_max_concurrency"`
	ThrottleQueueLimit     types.Int64  `tfsdk:"throttle_queue_limit"`
	Type                   types.String `tfsdk:"type"`
	Config                 types.String `tfsdk:"config"`
	BatchSize              types.Int64  `tfsdk:"batch_size"`
	BatchWindowSeconds     types.Int64  `tfsdk:"batch_window_seconds"`
	UseStaticIP            types.Bool   `tfsdk:"use_static_ip"`
	IsActive               types.Bool   `tfsdk:"is_active"`
	CreatedAt              types.String `tfsdk:"created_at"`
	UpdatedAt              types.String `tfsdk:"updated_at"`
}

func NewDestinationResource() resource.Resource {
	return &DestinationResource{}
}

func (r *DestinationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_destination"
}

func (r *DestinationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hookbase webhook destination.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Destination ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Destination name (1-100 characters).",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "URL-safe identifier. Lowercase alphanumeric and hyphens only. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"url": schema.StringAttribute{
				Description: "Destination URL to deliver webhooks to.",
				Optional:    true,
			},
			"method": schema.StringAttribute{
				Description: "HTTP method for delivery. Defaults to POST.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("POST"),
			},
			"headers": schema.MapAttribute{
				Description: "Custom HTTP headers to include in deliveries.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"auth_type": schema.StringAttribute{
				Description: "Authentication type: none, basic, bearer, api_key, oauth2, or custom.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("none"),
			},
			"auth_config": schema.MapAttribute{
				Description: "Authentication configuration. Sensitive — API redacts this on read.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapUseStateForUnknown{},
				},
			},
			"timeout_ms": schema.Int64Attribute{
				Description: "Delivery timeout in milliseconds.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30000),
			},
			"throttle_mode": schema.StringAttribute{
				Description: "Delivery throttling mode: \"off\" (no throttling), \"rate\" (fixed rate limit, requires throttle_rate_limit and throttle_rate_unit), or \"concurrency\" (max concurrent in-flight deliveries, requires throttle_max_concurrency). Defaults to \"off\".",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("off"),
				Validators: []validator.String{
					stringvalidator.OneOf("off", "rate", "concurrency"),
				},
			},
			"throttle_rate_limit": schema.Int64Attribute{
				Description: "Maximum number of deliveries per throttle_rate_unit. Required when throttle_mode is \"rate\".",
				Optional:    true,
			},
			"throttle_rate_unit": schema.StringAttribute{
				Description: "Time unit for throttle_rate_limit: second, minute, or hour. Required when throttle_mode is \"rate\".",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("second", "minute", "hour"),
				},
			},
			"throttle_max_concurrency": schema.Int64Attribute{
				Description: "Maximum number of concurrent in-flight deliveries to this destination. Required when throttle_mode is \"concurrency\".",
				Optional:    true,
			},
			"throttle_queue_limit": schema.Int64Attribute{
				Description: "Maximum number of deliveries to queue while throttled. Optional for both \"rate\" and \"concurrency\" modes.",
				Optional:    true,
			},
			"type": schema.StringAttribute{
				Description: "Destination type: http, websocket, tunnel, sqs, sns, gcp_pubsub, azure_servicebus, kafka, or rabbitmq.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("http"),
			},
			"config": schema.StringAttribute{
				Description: "JSON configuration for non-HTTP destination types. Sensitive — API redacts this on read.",
				Optional:    true,
				Sensitive:   true,
			},
			"batch_size": schema.Int64Attribute{
				Description: "Number of events to batch per delivery.",
				Optional:    true,
			},
			"batch_window_seconds": schema.Int64Attribute{
				Description: "Time window in seconds to collect events before batching.",
				Optional:    true,
			},
			"use_static_ip": schema.BoolAttribute{
				Description: "Use a static IP for deliveries.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether the destination is active.",
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

func (r *DestinationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *DestinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateDestinationRequest{
		Name: plan.Name.ValueString(),
		Slug: plan.Slug.ValueString(),
		URL:  plan.URL.ValueString(),
	}

	if !plan.Method.IsNull() {
		v := plan.Method.ValueString()
		createReq.Method = &v
	}
	if !plan.AuthType.IsNull() {
		v := plan.AuthType.ValueString()
		createReq.AuthType = &v
	}
	if !plan.TimeoutMs.IsNull() {
		v := int(plan.TimeoutMs.ValueInt64())
		createReq.TimeoutMs = &v
	}
	if !plan.Type.IsNull() {
		v := plan.Type.ValueString()
		createReq.Type = &v
	}
	if !plan.BatchSize.IsNull() {
		v := int(plan.BatchSize.ValueInt64())
		createReq.BatchSize = &v
	}
	if !plan.BatchWindowSeconds.IsNull() {
		v := int(plan.BatchWindowSeconds.ValueInt64())
		createReq.BatchWindowSeconds = &v
	}
	if !plan.UseStaticIP.IsNull() {
		v := plan.UseStaticIP.ValueBool()
		createReq.UseStaticIP = &v
	}
	createReq.Throttle = throttleFromPlan(&plan)

	resp.Diagnostics.Append(mapToStringMap(ctx, plan.Headers, &createReq.Headers)...)
	resp.Diagnostics.Append(mapToStringMap(ctx, plan.AuthConfig, &createReq.AuthConfig)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		createReq.Config = json.RawMessage(plan.Config.ValueString())
	}

	dest, err := r.client.CreateDestination(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating destination", err.Error())
		return
	}

	// Save sensitive fields from create response (redacted on read)
	savedAuthConfig := dest.AuthConfig
	savedConfig := dest.Config

	// Re-read to get full object with timestamps
	dest, err = r.client.GetDestination(ctx, dest.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading destination after create", err.Error())
		return
	}
	dest.AuthConfig = savedAuthConfig
	dest.Config = savedConfig

	mapDestinationToState(ctx, dest, &plan, &resp.Diagnostics)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DestinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve sensitive fields before overwriting state
	prevAuthConfig := state.AuthConfig
	prevConfig := state.Config

	dest, err := r.client.GetDestination(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading destination", err.Error())
		return
	}

	mapDestinationToState(ctx, dest, &state, &resp.Diagnostics)

	// Restore sensitive fields that the API redacts
	if !prevAuthConfig.IsNull() && !prevAuthConfig.IsUnknown() {
		state.AuthConfig = prevAuthConfig
	}
	if !prevConfig.IsNull() && !prevConfig.IsUnknown() {
		state.Config = prevConfig
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DestinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan DestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateDestinationRequest{}

	if !plan.Name.IsNull() {
		v := plan.Name.ValueString()
		updateReq.Name = &v
	}
	if !plan.URL.IsNull() {
		v := plan.URL.ValueString()
		updateReq.URL = &v
	}
	if !plan.Method.IsNull() {
		v := plan.Method.ValueString()
		updateReq.Method = &v
	}
	if !plan.AuthType.IsNull() {
		v := plan.AuthType.ValueString()
		updateReq.AuthType = &v
	}
	if !plan.TimeoutMs.IsNull() {
		v := int(plan.TimeoutMs.ValueInt64())
		updateReq.TimeoutMs = &v
	}
	if !plan.Type.IsNull() {
		v := plan.Type.ValueString()
		updateReq.Type = &v
	}
	if !plan.BatchSize.IsNull() {
		v := int(plan.BatchSize.ValueInt64())
		updateReq.BatchSize = &v
	}
	if !plan.BatchWindowSeconds.IsNull() {
		v := int(plan.BatchWindowSeconds.ValueInt64())
		updateReq.BatchWindowSeconds = &v
	}
	if !plan.UseStaticIP.IsNull() {
		v := plan.UseStaticIP.ValueBool()
		updateReq.UseStaticIP = &v
	}
	if !plan.IsActive.IsNull() {
		v := plan.IsActive.ValueBool()
		updateReq.IsActive = &v
	}
	updateReq.Throttle = throttleFromPlan(&plan)

	resp.Diagnostics.Append(mapToStringMap(ctx, plan.Headers, &updateReq.Headers)...)
	resp.Diagnostics.Append(mapToStringMap(ctx, plan.AuthConfig, &updateReq.AuthConfig)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		updateReq.Config = json.RawMessage(plan.Config.ValueString())
	}

	dest, err := r.client.UpdateDestination(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating destination", err.Error())
		return
	}

	mapDestinationToState(ctx, dest, &plan, &resp.Diagnostics)

	// Sensitive fields (auth_config, config) are redacted in API responses; mapDestinationToState
	// above already preserves the plan values into `plan`, so no extra copy is needed here.

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DestinationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteDestination(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting destination", err.Error())
	}
}

func (r *DestinationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// throttleFromPlan builds a client.Throttle from the throttle_* plan attributes.
// throttle_mode is Optional+Computed with a default of "off", so it is always
// known by the time Create/Update run; the sub-fields remain nullable.
func throttleFromPlan(plan *DestinationResourceModel) *client.Throttle {
	throttle := &client.Throttle{Mode: plan.ThrottleMode.ValueString()}
	if !plan.ThrottleRateLimit.IsNull() {
		v := int(plan.ThrottleRateLimit.ValueInt64())
		throttle.RateLimit = &v
	}
	if !plan.ThrottleRateUnit.IsNull() {
		v := plan.ThrottleRateUnit.ValueString()
		throttle.RateUnit = &v
	}
	if !plan.ThrottleMaxConcurrency.IsNull() {
		v := int(plan.ThrottleMaxConcurrency.ValueInt64())
		throttle.MaxConcurrency = &v
	}
	if !plan.ThrottleQueueLimit.IsNull() {
		v := int(plan.ThrottleQueueLimit.ValueInt64())
		throttle.QueueLimit = &v
	}
	return throttle
}

func mapDestinationToState(ctx context.Context, dest *client.Destination, state *DestinationResourceModel, diags *diag.Diagnostics) {
	state.ID = types.StringValue(dest.ID)
	state.Name = types.StringValue(dest.Name)
	state.Slug = types.StringValue(dest.Slug)

	if dest.URL != "" {
		state.URL = types.StringValue(dest.URL)
	} else {
		state.URL = types.StringNull()
	}

	state.Method = types.StringValue(dest.Method)

	if dest.AuthType != nil {
		state.AuthType = types.StringValue(*dest.AuthType)
	} else {
		state.AuthType = types.StringValue("none")
	}

	if dest.TimeoutMs != nil {
		state.TimeoutMs = types.Int64Value(int64(*dest.TimeoutMs))
	} else {
		state.TimeoutMs = types.Int64Value(30000)
	}

	if dest.Throttle != nil {
		state.ThrottleMode = types.StringValue(dest.Throttle.Mode)
		if dest.Throttle.RateLimit != nil {
			state.ThrottleRateLimit = types.Int64Value(int64(*dest.Throttle.RateLimit))
		} else {
			state.ThrottleRateLimit = types.Int64Null()
		}
		if dest.Throttle.RateUnit != nil {
			state.ThrottleRateUnit = types.StringValue(*dest.Throttle.RateUnit)
		} else {
			state.ThrottleRateUnit = types.StringNull()
		}
		if dest.Throttle.MaxConcurrency != nil {
			state.ThrottleMaxConcurrency = types.Int64Value(int64(*dest.Throttle.MaxConcurrency))
		} else {
			state.ThrottleMaxConcurrency = types.Int64Null()
		}
		if dest.Throttle.QueueLimit != nil {
			state.ThrottleQueueLimit = types.Int64Value(int64(*dest.Throttle.QueueLimit))
		} else {
			state.ThrottleQueueLimit = types.Int64Null()
		}
	} else {
		state.ThrottleMode = types.StringValue("off")
		state.ThrottleRateLimit = types.Int64Null()
		state.ThrottleRateUnit = types.StringNull()
		state.ThrottleMaxConcurrency = types.Int64Null()
		state.ThrottleQueueLimit = types.Int64Null()
	}

	state.Type = types.StringValue(dest.Type)

	if dest.BatchSize != nil && *dest.BatchSize > 1 {
		state.BatchSize = types.Int64Value(int64(*dest.BatchSize))
	} else if !state.BatchSize.IsNull() && !state.BatchSize.IsUnknown() {
		// Preserve user-set value
		state.BatchSize = types.Int64Value(state.BatchSize.ValueInt64())
	} else {
		state.BatchSize = types.Int64Null()
	}

	if dest.BatchWindowSeconds != nil && *dest.BatchWindowSeconds > 0 {
		state.BatchWindowSeconds = types.Int64Value(int64(*dest.BatchWindowSeconds))
	} else if !state.BatchWindowSeconds.IsNull() && !state.BatchWindowSeconds.IsUnknown() {
		state.BatchWindowSeconds = types.Int64Value(state.BatchWindowSeconds.ValueInt64())
	} else {
		state.BatchWindowSeconds = types.Int64Null()
	}

	state.UseStaticIP = types.BoolValue(dest.UseStaticIP)
	state.IsActive = types.BoolValue(dest.IsActive)
	if dest.CreatedAt != "" {
		state.CreatedAt = types.StringValue(dest.CreatedAt)
	}
	if dest.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(dest.UpdatedAt)
	}

	// Headers
	if dest.Headers != nil && len(dest.Headers) > 0 {
		m, d := types.MapValueFrom(ctx, types.StringType, dest.Headers)
		diags.Append(d...)
		state.Headers = m
	} else {
		state.Headers = types.MapNull(types.StringType)
	}

	// AuthConfig — API redacts, so only set if non-empty; otherwise leave as-is (handled by caller)
	if dest.AuthConfig != nil && len(dest.AuthConfig) > 0 {
		m, d := types.MapValueFrom(ctx, types.StringType, dest.AuthConfig)
		diags.Append(d...)
		state.AuthConfig = m
	}
	// If AuthConfig is nil/empty from API, leave state.AuthConfig unchanged (caller preserves)

	// Config — API redacts, so only set if non-empty
	if dest.Config != nil && len(dest.Config) > 0 && string(dest.Config) != "null" {
		state.Config = types.StringValue(string(dest.Config))
	}
	// If Config is nil/empty from API, leave state.Config unchanged (caller preserves)
}

// Helper to convert types.Map to map[string]string for client requests
func mapToStringMap(ctx context.Context, m types.Map, target *map[string]string) diag.Diagnostics {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	result := make(map[string]string)
	diags := m.ElementsAs(ctx, &result, false)
	*target = result
	return diags
}

// mapUseStateForUnknown is a plan modifier for Map attributes that preserves
// the prior state value when the planned value is unknown.
type mapUseStateForUnknown struct{}

func (m mapUseStateForUnknown) Description(_ context.Context) string {
	return "Use the prior state value when the planned value is unknown."
}

func (m mapUseStateForUnknown) MarkdownDescription(_ context.Context) string {
	return "Use the prior state value when the planned value is unknown."
}

func (m mapUseStateForUnknown) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if !req.PlanValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}
