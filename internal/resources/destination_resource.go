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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var (
	_ resource.Resource                   = &DestinationResource{}
	_ resource.ResourceWithImportState    = &DestinationResource{}
	_ resource.ResourceWithValidateConfig = &DestinationResource{}
)

// destinationTypes are the API's DESTINATION_TYPES (api/src/routes/destinations.ts).
var destinationTypes = []string{
	"http", "sqs", "eventbridge", "servicebus", "pubsub", "oci_queue",
	"s3", "r2", "gcs", "azure_blob", "bridge",
}

// destinationConfigRedacted reports whether the API masks credentials in this type's config on
// read: the queue and storage types (isQueueType / isWarehouseType in the API).
func destinationConfigRedacted(destType string) bool {
	return destType != "http" && destType != "bridge"
}

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
	SigningSecret          types.String `tfsdk:"signing_secret"`
	HasSigningSecret       types.Bool   `tfsdk:"has_signing_secret"`
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
				Description: "Destination URL to deliver webhooks to. Required for http destinations. Leave it " +
					"unset for queue and storage types, which deliver through config, and for bridge " +
					"destinations, which take no URL.",
				Optional: true,
				Validators: []validator.String{
					// The API stores "" for "no URL" and reads it back as null, so "" could never match.
					stringvalidator.LengthAtLeast(1),
				},
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
				Description: "Authentication type: none, basic, bearer, api_key, or custom_header. Defaults to none.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("none"),
				Validators: []validator.String{
					stringvalidator.OneOf("none", "basic", "bearer", "api_key", "custom_header"),
				},
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
				Description: "Destination type: http (the default), the queues sqs, eventbridge, servicebus, " +
					"pubsub and oci_queue, the storage types s3, r2, gcs and azure_blob, or bridge, which " +
					"publishes each delivery as an outbound event. Queue and storage types need a Pro plan " +
					"or above. Changing this forces a new resource.",
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("http"),
				Validators: []validator.String{
					stringvalidator.OneOf(destinationTypes...),
				},
				PlanModifiers: []planmodifier.String{
					// The API refuses to change a destination's type.
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.StringAttribute{
				Description: "JSON configuration (use jsonencode) for the queue, storage and bridge types. A " +
					"bridge takes {eventTypeSource = \"fixed\" | \"field\", eventType, eventTypeField, " +
					"eventTypePrefix, applicationId, allowUnverifiedSources}: a fixed eventType must name an " +
					"enabled event type, and creating or changing a bridge also needs the events:send " +
					"permission. Credentials in a queue or storage config are masked when read back, so " +
					"refresh compares them only for presence; key order and formatting are ignored.",
				Optional:  true,
				Sensitive: true,
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
			"signing_secret": schema.StringAttribute{
				Description: "The secret Hookbase signs this destination's deliveries with (http destinations " +
					"only), for the receiver to verify the webhook-signature header. Hookbase generates it at " +
					"create; it is refreshed from the API on read. Null for other types. Rotate it in the " +
					"dashboard or API, not here.",
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"has_signing_secret": schema.BoolAttribute{
				Description: "Whether the destination signs its deliveries (true for http destinations).",
				Computed:    true,
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

// ValidateConfig checks what the API would reject about url at apply, at plan time instead: an
// http destination needs one, and a bridge takes none.
func (r *DestinationResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var destType, url types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("type"), &destType)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("url"), &url)...)
	if resp.Diagnostics.HasError() || destType.IsUnknown() || url.IsUnknown() {
		return
	}
	t := "http"
	if !destType.IsNull() {
		t = destType.ValueString()
	}
	switch {
	case t == "http" && url.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Missing url", "An http destination needs a url to deliver to.")
	case t == "bridge" && !url.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("url"), "Unexpected url",
			"A bridge destination publishes outbound events and takes no url.")
	}
}

func (r *DestinationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq, diags := buildCreateDestinationRequest(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateDestination(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating destination", err.Error())
		return
	}

	// Re-read to get full object with timestamps
	dest, err := r.client.GetDestination(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading destination after create", err.Error())
		return
	}

	// config and auth_config keep their planned values (mapDestinationToState leaves them alone):
	// the API masks credentials in both and re-encodes config, and Terraform requires the applied
	// value to match the plan.
	mapDestinationToState(ctx, dest, &plan, &resp.Diagnostics)
	if plan.AuthConfig.IsUnknown() {
		plan.AuthConfig = authConfigFromAPI(ctx, dest, &resp.Diagnostics)
	}
	// The create reply is the one moment the API returns the signing secret without a reveal call.
	// Null for queue, storage and bridge destinations, which don't sign.
	if secret := nonEmpty(created.SigningSecret); secret != nil {
		plan.SigningSecret = types.StringValue(*secret)
	} else {
		plan.SigningSecret = r.readSigningSecret(ctx, dest, types.StringNull())
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DestinationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DestinationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	prior := state

	dest, err := r.client.GetDestination(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading destination", err.Error())
		return
	}

	mapDestinationToState(ctx, dest, &state, &resp.Diagnostics)

	// auth_config: the API masks it for principals without credential access, so a value in state
	// is kept; with none (an import) the API's is taken.
	if prior.AuthConfig.IsNull() || prior.AuthConfig.IsUnknown() {
		state.AuthConfig = authConfigFromAPI(ctx, dest, &resp.Diagnostics)
	}
	state.Config = jsonToState(prior.Config, dest.Config, destinationConfigRedacted(dest.Type))
	state.SigningSecret = r.readSigningSecret(ctx, dest, prior.SigningSecret)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DestinationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state DestinationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := buildUpdateDestinationRequest(ctx, &plan, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// PATCH answers {success:true} only, so the reply carries nothing to map.
	if err := r.client.UpdateDestination(ctx, plan.ID.ValueString(), updateReq); err != nil {
		resp.Diagnostics.AddError("Error updating destination", err.Error())
		return
	}

	dest, err := r.client.GetDestination(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading destination after update", err.Error())
		return
	}

	plannedSecret := plan.SigningSecret
	mapDestinationToState(ctx, dest, &plan, &resp.Diagnostics)
	if plan.AuthConfig.IsUnknown() {
		plan.AuthConfig = authConfigFromAPI(ctx, dest, &resp.Diagnostics)
	}
	// An update never changes the secret, so the planned value (the prior state's) stands. It is
	// read back only if the plan left it unknown, which UseStateForUnknown should prevent.
	if plannedSecret.IsUnknown() {
		plan.SigningSecret = r.readSigningSecret(ctx, dest, types.StringNull())
	}
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

// readSigningSecret returns the destination's signing secret for state: null when it has none
// (every type but http), otherwise the reveal-secret answer. When the reveal call fails (e.g. an
// API key without write scope), prior is kept rather than guessing.
func (r *DestinationResource) readSigningSecret(ctx context.Context, dest *client.Destination, prior types.String) types.String {
	if !dest.HasSigningSecret {
		return types.StringNull()
	}
	secret, err := r.client.RevealDestinationSecret(ctx, dest.ID)
	if err != nil || secret == "" {
		return prior
	}
	return types.StringValue(secret)
}

// authConfigFromAPI is auth_config as the API reports it, for when state has no value of its own.
func authConfigFromAPI(ctx context.Context, dest *client.Destination, diags *diag.Diagnostics) types.Map {
	if len(dest.AuthConfig) == 0 {
		return types.MapNull(types.StringType)
	}
	m, d := types.MapValueFrom(ctx, types.StringType, dest.AuthConfig)
	diags.Append(d...)
	return m
}

func buildCreateDestinationRequest(ctx context.Context, plan *DestinationResourceModel) (client.CreateDestinationRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	createReq := client.CreateDestinationRequest{
		Name: plan.Name.ValueString(),
		Slug: plan.Slug.ValueString(),
		// "" when unset, which the API accepts for every type but http.
		URL: plan.URL.ValueString(),
	}

	createReq.Method = knownString(plan.Method)
	createReq.AuthType = knownString(plan.AuthType)
	createReq.TimeoutMs = knownInt(plan.TimeoutMs)
	createReq.Type = knownString(plan.Type)
	createReq.BatchSize = knownInt(plan.BatchSize)
	createReq.BatchWindowSeconds = knownInt(plan.BatchWindowSeconds)
	createReq.UseStaticIP = knownBool(plan.UseStaticIP)
	createReq.Throttle = throttleFromPlan(plan)

	diags.Append(mapToStringMap(ctx, plan.Headers, &createReq.Headers)...)
	diags.Append(mapToStringMap(ctx, plan.AuthConfig, &createReq.AuthConfig)...)

	if !plan.Config.IsNull() && !plan.Config.IsUnknown() {
		createReq.Config = json.RawMessage(plan.Config.ValueString())
	}
	return createReq, diags
}

// buildUpdateDestinationRequest builds the PATCH body. Values in the plan are sent; an optional
// attribute removed from configuration is cleared (url to "", headers and config to null, batching
// to the API's defaults) so the stored value doesn't outlive it. auth_config is Computed and keeps
// its state value when removed, so it is simply re-sent. type is sent unchanged (the API accepts
// the current type and refuses any other; RequiresReplace plans a change as a new destination).
func buildUpdateDestinationRequest(ctx context.Context, plan, state *DestinationResourceModel) (client.UpdateDestinationRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	updateReq := client.UpdateDestinationRequest{
		Name:        knownString(plan.Name),
		URL:         knownString(plan.URL),
		Method:      knownString(plan.Method),
		AuthType:    knownString(plan.AuthType),
		TimeoutMs:   knownInt(plan.TimeoutMs),
		Type:        knownString(plan.Type),
		UseStaticIP: knownBool(plan.UseStaticIP),
		IsActive:    knownBool(plan.IsActive),
		Throttle:    throttleFromPlan(plan),
	}
	if plan.URL.IsNull() && !state.URL.IsNull() {
		empty := ""
		updateReq.URL = &empty
	}

	updateReq.BatchSize = knownInt(plan.BatchSize)
	if plan.BatchSize.IsNull() && !state.BatchSize.IsNull() {
		one := 1
		updateReq.BatchSize = &one
	}
	updateReq.BatchWindowSeconds = knownInt(plan.BatchWindowSeconds)
	if plan.BatchWindowSeconds.IsNull() && !state.BatchWindowSeconds.IsNull() {
		zero := 0
		updateReq.BatchWindowSeconds = &zero
	}

	switch {
	case !plan.Headers.IsNull() && !plan.Headers.IsUnknown():
		headers := map[string]string{}
		diags.Append(plan.Headers.ElementsAs(ctx, &headers, false)...)
		updateReq.Headers = &headers
	case plan.Headers.IsNull() && !state.Headers.IsNull():
		var cleared map[string]string
		updateReq.Headers = &cleared
	}
	diags.Append(mapToStringMap(ctx, plan.AuthConfig, &updateReq.AuthConfig)...)

	// config is sent only when it changed. The API re-validates a config it is given (for a bridge:
	// that its event type is still enabled), and a rename must not fail on that.
	switch {
	case plan.Config.Equal(state.Config) || plan.Config.IsUnknown():
	case !plan.Config.IsNull():
		updateReq.Config = json.RawMessage(plan.Config.ValueString())
	default:
		updateReq.Config = json.RawMessage("null")
	}
	return updateReq, diags
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

// mapDestinationToState copies the API's view of a destination into state. It leaves config,
// auth_config and signing_secret alone: the API masks or re-encodes the first two and returns the
// secret only on create, so each operation decides where they come from.
func mapDestinationToState(ctx context.Context, dest *client.Destination, state *DestinationResourceModel, diags *diag.Diagnostics) {
	state.ID = types.StringValue(dest.ID)
	state.Name = types.StringValue(dest.Name)
	state.Slug = types.StringValue(dest.Slug)

	// Queue, storage and bridge destinations store "" for no URL.
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
	state.HasSigningSecret = types.BoolValue(dest.HasSigningSecret)
	if dest.CreatedAt != "" {
		state.CreatedAt = types.StringValue(dest.CreatedAt)
	}
	if dest.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(dest.UpdatedAt)
	}

	// Headers. An empty answer keeps a planned or prior empty map (headers = {}), as for lists.
	switch {
	case len(dest.Headers) > 0:
		m, d := types.MapValueFrom(ctx, types.StringType, dest.Headers)
		diags.Append(d...)
		state.Headers = m
	case !state.Headers.IsNull() && !state.Headers.IsUnknown() && len(state.Headers.Elements()) == 0:
		// keep {}
	default:
		state.Headers = types.MapNull(types.StringType)
	}
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
	// An unknown configuration value (e.g. a reference to a resource not yet created) must stay
	// unknown, or the plan would promise the prior state and the apply would contradict it.
	if !req.PlanValue.IsUnknown() || req.ConfigValue.IsUnknown() {
		return
	}
	resp.PlanValue = req.StateValue
}
