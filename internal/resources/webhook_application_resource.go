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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var (
	_ resource.Resource                = &WebhookApplicationResource{}
	_ resource.ResourceWithImportState = &WebhookApplicationResource{}
)

type WebhookApplicationResource struct {
	client *client.Client
}

type WebhookApplicationResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ExternalID         types.String `tfsdk:"external_id"`
	Name               types.String `tfsdk:"name"`
	Metadata           types.String `tfsdk:"metadata"`
	RateLimitPerSecond types.Int64  `tfsdk:"rate_limit_per_second"`
	RateLimitPerMinute types.Int64  `tfsdk:"rate_limit_per_minute"`
	RateLimitPerHour   types.Int64  `tfsdk:"rate_limit_per_hour"`
	IsDisabled         types.Bool   `tfsdk:"is_disabled"`
	DisabledReason     types.String `tfsdk:"disabled_reason"`
	CreatedAt          types.String `tfsdk:"created_at"`
	UpdatedAt          types.String `tfsdk:"updated_at"`
}

func NewWebhookApplicationResource() resource.Resource {
	return &WebhookApplicationResource{}
}

func (r *WebhookApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_application"
}

func (r *WebhookApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hookbase webhook application.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Application ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_id": schema.StringAttribute{
				Description: "External identifier for the application. Changing this forces a new resource.",
				Optional:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Application name.",
				Required:    true,
			},
			"metadata": schema.StringAttribute{
				Description: "Arbitrary metadata as a JSON string.",
				Optional:    true,
			},
			"rate_limit_per_second": schema.Int64Attribute{
				Description: "Rate limit per second.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(100),
			},
			"rate_limit_per_minute": schema.Int64Attribute{
				Description: "Rate limit per minute.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1000),
			},
			"rate_limit_per_hour": schema.Int64Attribute{
				Description: "Rate limit per hour.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(10000),
			},
			"is_disabled": schema.BoolAttribute{
				Description: "Whether the application is disabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"disabled_reason": schema.StringAttribute{
				Description: "Reason the application was disabled.",
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

func (r *WebhookApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateWebhookApplicationRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.ExternalID.IsNull() {
		v := plan.ExternalID.ValueString()
		createReq.ExternalID = &v
	}
	if !plan.Metadata.IsNull() {
		createReq.Metadata = json.RawMessage(plan.Metadata.ValueString())
	}
	if !plan.RateLimitPerSecond.IsNull() {
		v := int(plan.RateLimitPerSecond.ValueInt64())
		createReq.RateLimitPerSecond = &v
	}
	if !plan.RateLimitPerMinute.IsNull() {
		v := int(plan.RateLimitPerMinute.ValueInt64())
		createReq.RateLimitPerMinute = &v
	}
	if !plan.RateLimitPerHour.IsNull() {
		v := int(plan.RateLimitPerHour.ValueInt64())
		createReq.RateLimitPerHour = &v
	}

	app, err := r.client.CreateWebhookApplication(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook application", err.Error())
		return
	}

	// Re-read to get full object with timestamps
	app, err = r.client.GetWebhookApplication(ctx, app.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook application after create", err.Error())
		return
	}

	mapWebhookApplicationToState(app, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := r.client.GetWebhookApplication(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook application", err.Error())
		return
	}

	mapWebhookApplicationToState(app, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WebhookApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateWebhookApplicationRequest{}

	if !plan.Name.IsNull() {
		v := plan.Name.ValueString()
		updateReq.Name = &v
	}
	if !plan.Metadata.IsNull() {
		updateReq.Metadata = json.RawMessage(plan.Metadata.ValueString())
	}
	if !plan.RateLimitPerSecond.IsNull() {
		v := int(plan.RateLimitPerSecond.ValueInt64())
		updateReq.RateLimitPerSecond = &v
	}
	if !plan.RateLimitPerMinute.IsNull() {
		v := int(plan.RateLimitPerMinute.ValueInt64())
		updateReq.RateLimitPerMinute = &v
	}
	if !plan.RateLimitPerHour.IsNull() {
		v := int(plan.RateLimitPerHour.ValueInt64())
		updateReq.RateLimitPerHour = &v
	}
	if !plan.IsDisabled.IsNull() {
		v := plan.IsDisabled.ValueBool()
		updateReq.IsDisabled = &v
	}
	if !plan.DisabledReason.IsNull() {
		v := plan.DisabledReason.ValueString()
		updateReq.DisabledReason = &v
	}

	app, err := r.client.UpdateWebhookApplication(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook application", err.Error())
		return
	}

	mapWebhookApplicationToState(app, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWebhookApplication(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting webhook application", err.Error())
	}
}

func (r *WebhookApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapWebhookApplicationToState(app *client.WebhookApplication, state *WebhookApplicationResourceModel, _ *diag.Diagnostics) {
	state.ID = types.StringValue(app.ID)
	state.ExternalID = stringPtrToValue(app.ExternalID)
	state.Name = types.StringValue(app.Name)
	state.RateLimitPerSecond = types.Int64Value(int64(app.RateLimitPerSecond))
	state.RateLimitPerMinute = types.Int64Value(int64(app.RateLimitPerMinute))
	state.RateLimitPerHour = types.Int64Value(int64(app.RateLimitPerHour))
	state.IsDisabled = types.BoolValue(app.IsDisabled)
	state.DisabledReason = stringPtrToValue(app.DisabledReason)
	if app.CreatedAt != "" {
		state.CreatedAt = types.StringValue(app.CreatedAt)
	}
	if app.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(app.UpdatedAt)
	}

	if app.Metadata != nil && len(app.Metadata) > 0 && string(app.Metadata) != "null" {
		state.Metadata = types.StringValue(string(app.Metadata))
	} else {
		state.Metadata = types.StringNull()
	}
}
