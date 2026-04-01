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
	_ resource.Resource                = &WebhookEndpointResource{}
	_ resource.ResourceWithImportState = &WebhookEndpointResource{}
)

type WebhookEndpointResource struct {
	client *client.Client
}

type WebhookEndpointResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	ApplicationID           types.String `tfsdk:"application_id"`
	URL                     types.String `tfsdk:"url"`
	Description             types.String `tfsdk:"description"`
	Headers                 types.String `tfsdk:"headers"`
	TimeoutSeconds          types.Int64  `tfsdk:"timeout_seconds"`
	RateLimitPerSecond      types.Int64  `tfsdk:"rate_limit_per_second"`
	SuccessStatusCodes      types.String `tfsdk:"success_status_codes"`
	BackoffType             types.String `tfsdk:"backoff_type"`
	RetryDelays             types.String `tfsdk:"retry_delays"`
	UseStaticIP             types.Bool   `tfsdk:"use_static_ip"`
	CircuitFailureThreshold types.Int64  `tfsdk:"circuit_failure_threshold"`
	CircuitSuccessThreshold types.Int64  `tfsdk:"circuit_success_threshold"`
	CircuitCooldownSeconds  types.Int64  `tfsdk:"circuit_cooldown_seconds"`
	IsDisabled              types.Bool   `tfsdk:"is_disabled"`
	DisabledReason          types.String `tfsdk:"disabled_reason"`
	Secret                  types.String `tfsdk:"secret"`
	SecretVersion           types.Int64  `tfsdk:"secret_version"`
	CreatedAt               types.String `tfsdk:"created_at"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
}

func NewWebhookEndpointResource() resource.Resource {
	return &WebhookEndpointResource{}
}

func (r *WebhookEndpointResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_endpoint"
}

func (r *WebhookEndpointResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hookbase webhook endpoint.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Endpoint ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"application_id": schema.StringAttribute{
				Description: "ID of the parent webhook application.",
				Required:    true,
			},
			"url": schema.StringAttribute{
				Description: "Destination URL (HTTPS only).",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Endpoint description.",
				Optional:    true,
			},
			"headers": schema.StringAttribute{
				Description: "Custom headers as a JSON array of {name, value} objects.",
				Optional:    true,
			},
			"timeout_seconds": schema.Int64Attribute{
				Description: "Request timeout in seconds.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
			},
			"rate_limit_per_second": schema.Int64Attribute{
				Description: "Rate limit per second (0 = unlimited).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"success_status_codes": schema.StringAttribute{
				Description: "Accepted HTTP status codes as a JSON array.",
				Optional:    true,
			},
			"backoff_type": schema.StringAttribute{
				Description: "Retry backoff strategy: exponential, linear, or fixed.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("exponential"),
			},
			"retry_delays": schema.StringAttribute{
				Description: "Custom retry delays as a JSON array of integers (seconds).",
				Optional:    true,
			},
			"use_static_ip": schema.BoolAttribute{
				Description: "Route traffic through a static IP address.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"circuit_failure_threshold": schema.Int64Attribute{
				Description: "Number of consecutive failures before opening the circuit.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(5),
			},
			"circuit_success_threshold": schema.Int64Attribute{
				Description: "Number of consecutive successes before closing the circuit.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"circuit_cooldown_seconds": schema.Int64Attribute{
				Description: "Seconds to wait before retrying after circuit opens.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(60),
			},
			"is_disabled": schema.BoolAttribute{
				Description: "Whether the endpoint is disabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"disabled_reason": schema.StringAttribute{
				Description: "Reason the endpoint was disabled.",
				Optional:    true,
			},
			"secret": schema.StringAttribute{
				Description: "Webhook signing secret (whsec_...). Only returned on create.",
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"secret_version": schema.Int64Attribute{
				Description: "Secret version number.",
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

func (r *WebhookEndpointResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookEndpointResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookEndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateWebhookEndpointRequest{
		ApplicationID: plan.ApplicationID.ValueString(),
		URL:           plan.URL.ValueString(),
	}

	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		createReq.Description = &v
	}
	if !plan.Headers.IsNull() {
		createReq.Headers = json.RawMessage(plan.Headers.ValueString())
	}
	if !plan.TimeoutSeconds.IsNull() {
		v := int(plan.TimeoutSeconds.ValueInt64())
		createReq.TimeoutSeconds = &v
	}
	if !plan.RateLimitPerSecond.IsNull() {
		v := int(plan.RateLimitPerSecond.ValueInt64())
		createReq.RateLimitPerSecond = &v
	}
	if !plan.SuccessStatusCodes.IsNull() {
		createReq.SuccessStatusCodes = json.RawMessage(plan.SuccessStatusCodes.ValueString())
	}
	if !plan.BackoffType.IsNull() {
		v := plan.BackoffType.ValueString()
		createReq.BackoffType = &v
	}
	if !plan.RetryDelays.IsNull() {
		createReq.RetryDelays = json.RawMessage(plan.RetryDelays.ValueString())
	}
	if !plan.UseStaticIP.IsNull() {
		v := plan.UseStaticIP.ValueBool()
		createReq.UseStaticIp = &v
	}
	if !plan.CircuitFailureThreshold.IsNull() {
		v := int(plan.CircuitFailureThreshold.ValueInt64())
		createReq.CircuitFailureThreshold = &v
	}
	if !plan.CircuitSuccessThreshold.IsNull() {
		v := int(plan.CircuitSuccessThreshold.ValueInt64())
		createReq.CircuitSuccessThreshold = &v
	}
	if !plan.CircuitCooldownSeconds.IsNull() {
		v := int(plan.CircuitCooldownSeconds.ValueInt64())
		createReq.CircuitCooldownSeconds = &v
	}

	endpoint, err := r.client.CreateWebhookEndpoint(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook endpoint", err.Error())
		return
	}

	// Save secret from create response (only returned on create)
	savedSecret := endpoint.Secret

	// Re-read to get full object with timestamps
	endpoint, err = r.client.GetWebhookEndpoint(ctx, endpoint.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook endpoint after create", err.Error())
		return
	}
	endpoint.Secret = savedSecret

	mapWebhookEndpointToState(endpoint, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookEndpointResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve secret from state since API does not return it on GET
	existingSecret := state.Secret

	endpoint, err := r.client.GetWebhookEndpoint(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook endpoint", err.Error())
		return
	}

	mapWebhookEndpointToState(endpoint, &state, &resp.Diagnostics)

	// Restore secret from prior state since API only returns it on create
	if !existingSecret.IsNull() && !existingSecret.IsUnknown() {
		state.Secret = existingSecret
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WebhookEndpointResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookEndpointResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve secret from current state
	var currentState WebhookEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &currentState)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateWebhookEndpointRequest{}

	if !plan.URL.IsNull() {
		v := plan.URL.ValueString()
		updateReq.URL = &v
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		updateReq.Description = &v
	}
	if !plan.Headers.IsNull() {
		updateReq.Headers = json.RawMessage(plan.Headers.ValueString())
	}
	if !plan.TimeoutSeconds.IsNull() {
		v := int(plan.TimeoutSeconds.ValueInt64())
		updateReq.TimeoutSeconds = &v
	}
	if !plan.IsDisabled.IsNull() {
		v := plan.IsDisabled.ValueBool()
		updateReq.IsDisabled = &v
	}
	if !plan.DisabledReason.IsNull() {
		v := plan.DisabledReason.ValueString()
		updateReq.DisabledReason = &v
	}
	if !plan.RateLimitPerSecond.IsNull() {
		v := int(plan.RateLimitPerSecond.ValueInt64())
		updateReq.RateLimitPerSecond = &v
	}
	if !plan.SuccessStatusCodes.IsNull() {
		updateReq.SuccessStatusCodes = json.RawMessage(plan.SuccessStatusCodes.ValueString())
	}
	if !plan.BackoffType.IsNull() {
		v := plan.BackoffType.ValueString()
		updateReq.BackoffType = &v
	}
	if !plan.RetryDelays.IsNull() {
		updateReq.RetryDelays = json.RawMessage(plan.RetryDelays.ValueString())
	}
	if !plan.UseStaticIP.IsNull() {
		v := plan.UseStaticIP.ValueBool()
		updateReq.UseStaticIp = &v
	}
	if !plan.CircuitFailureThreshold.IsNull() {
		v := int(plan.CircuitFailureThreshold.ValueInt64())
		updateReq.CircuitFailureThreshold = &v
	}
	if !plan.CircuitSuccessThreshold.IsNull() {
		v := int(plan.CircuitSuccessThreshold.ValueInt64())
		updateReq.CircuitSuccessThreshold = &v
	}
	if !plan.CircuitCooldownSeconds.IsNull() {
		v := int(plan.CircuitCooldownSeconds.ValueInt64())
		updateReq.CircuitCooldownSeconds = &v
	}

	endpoint, err := r.client.UpdateWebhookEndpoint(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook endpoint", err.Error())
		return
	}

	mapWebhookEndpointToState(endpoint, &plan, &resp.Diagnostics)

	// Preserve secret from state since update response doesn't include it
	if !currentState.Secret.IsNull() && !currentState.Secret.IsUnknown() {
		plan.Secret = currentState.Secret
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookEndpointResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookEndpointResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWebhookEndpoint(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting webhook endpoint", err.Error())
	}
}

func (r *WebhookEndpointResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapWebhookEndpointToState(ep *client.WebhookEndpoint, state *WebhookEndpointResourceModel, _ *diag.Diagnostics) {
	state.ID = types.StringValue(ep.ID)
	state.ApplicationID = types.StringValue(ep.ApplicationID)
	state.URL = types.StringValue(ep.URL)
	state.Description = stringPtrToValue(ep.Description)
	state.TimeoutSeconds = types.Int64Value(int64(ep.TimeoutSeconds))
	state.RateLimitPerSecond = types.Int64Value(int64(ep.RateLimitPerSecond))
	if ep.BackoffType != nil {
		state.BackoffType = types.StringValue(*ep.BackoffType)
	} else {
		state.BackoffType = types.StringValue("exponential")
	}
	state.UseStaticIP = types.BoolValue(ep.UseStaticIp)
	state.CircuitFailureThreshold = types.Int64Value(int64(ep.CircuitFailureThreshold))
	state.CircuitSuccessThreshold = types.Int64Value(int64(ep.CircuitSuccessThreshold))
	state.CircuitCooldownSeconds = types.Int64Value(int64(ep.CircuitCooldownSeconds))
	state.IsDisabled = types.BoolValue(ep.IsDisabled)
	state.DisabledReason = stringPtrToValue(ep.DisabledReason)
	state.SecretVersion = types.Int64Value(int64(ep.SecretVersion))
	if ep.CreatedAt != "" {
		state.CreatedAt = types.StringValue(ep.CreatedAt)
	}
	if ep.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(ep.UpdatedAt)
	}

	// Secret is only returned on create
	if ep.Secret != nil {
		state.Secret = types.StringValue(*ep.Secret)
	}

	if ep.Headers != nil && len(ep.Headers) > 0 && string(ep.Headers) != "null" && string(ep.Headers) != "[]" {
		state.Headers = types.StringValue(string(ep.Headers))
	} else {
		state.Headers = types.StringNull()
	}

	if ep.SuccessStatusCodes != nil && len(ep.SuccessStatusCodes) > 0 && string(ep.SuccessStatusCodes) != "null" {
		state.SuccessStatusCodes = types.StringValue(string(ep.SuccessStatusCodes))
	} else {
		state.SuccessStatusCodes = types.StringNull()
	}

	if ep.RetryDelays != nil && len(ep.RetryDelays) > 0 && string(ep.RetryDelays) != "null" {
		state.RetryDelays = types.StringValue(string(ep.RetryDelays))
	} else {
		state.RetryDelays = types.StringNull()
	}
}
