package resources

import (
	"context"
	"encoding/json"

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
	_ resource.Resource                = &WebhookSubscriptionResource{}
	_ resource.ResourceWithImportState = &WebhookSubscriptionResource{}
)

type WebhookSubscriptionResource struct {
	client *client.Client
}

type WebhookSubscriptionResourceModel struct {
	ID               types.String `tfsdk:"id"`
	EndpointID       types.String `tfsdk:"endpoint_id"`
	EventTypeID      types.String `tfsdk:"event_type_id"`
	FilterExpression types.String `tfsdk:"filter_expression"`
	LabelFilters     types.String `tfsdk:"label_filters"`
	LabelFilterMode  types.String `tfsdk:"label_filter_mode"`
	TransformID      types.String `tfsdk:"transform_id"`
	IsEnabled        types.Bool   `tfsdk:"is_enabled"`
	PriorityOverride types.Int64  `tfsdk:"priority_override"`
	CreatedAt        types.String `tfsdk:"created_at"`
}

func NewWebhookSubscriptionResource() resource.Resource {
	return &WebhookSubscriptionResource{}
}

func (r *WebhookSubscriptionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook_subscription"
}

func (r *WebhookSubscriptionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hookbase webhook subscription linking an endpoint to an event type.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Subscription ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"endpoint_id": schema.StringAttribute{
				Description: "ID of the webhook endpoint. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"event_type_id": schema.StringAttribute{
				Description: "ID of the event type. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"filter_expression": schema.StringAttribute{
				Description: "CEL or JSONPath filter expression for event filtering.",
				Optional:    true,
			},
			"label_filters": schema.StringAttribute{
				Description: "Label filters as a JSON object.",
				Optional:    true,
			},
			"label_filter_mode": schema.StringAttribute{
				Description: "Label filter matching mode: all or any.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("all"),
			},
			"transform_id": schema.StringAttribute{
				Description: "ID of the transform to apply before delivery.",
				Optional:    true,
			},
			"is_enabled": schema.BoolAttribute{
				Description: "Whether the subscription is enabled.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"priority_override": schema.Int64Attribute{
				Description: "Override delivery priority (0=Critical, 1=High, 2=Normal, 3=Low, 4=Bulk).",
				Optional:    true,
			},
			"created_at": schema.StringAttribute{
				Description: "Creation timestamp.",
				Computed:    true,
			},
		},
	}
}

func (r *WebhookSubscriptionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookSubscriptionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan WebhookSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateWebhookSubscriptionRequest{
		EndpointID:  plan.EndpointID.ValueString(),
		EventTypeID: plan.EventTypeID.ValueString(),
	}

	if !plan.FilterExpression.IsNull() {
		v := plan.FilterExpression.ValueString()
		createReq.FilterExpression = &v
	}
	if !plan.LabelFilters.IsNull() {
		createReq.LabelFilters = json.RawMessage(plan.LabelFilters.ValueString())
	}
	if !plan.LabelFilterMode.IsNull() {
		v := plan.LabelFilterMode.ValueString()
		createReq.LabelFilterMode = &v
	}
	if !plan.TransformID.IsNull() {
		v := plan.TransformID.ValueString()
		createReq.TransformID = &v
	}
	if !plan.IsEnabled.IsNull() {
		v := plan.IsEnabled.ValueBool()
		createReq.IsEnabled = &v
	}
	if !plan.PriorityOverride.IsNull() {
		v := int(plan.PriorityOverride.ValueInt64())
		createReq.PriorityOverride = &v
	}

	sub, err := r.client.CreateWebhookSubscription(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook subscription", err.Error())
		return
	}

	// Re-read to get full object with timestamps
	sub, err = r.client.GetWebhookSubscription(ctx, sub.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook subscription after create", err.Error())
		return
	}

	mapWebhookSubscriptionToState(sub, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookSubscriptionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state WebhookSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sub, err := r.client.GetWebhookSubscription(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook subscription", err.Error())
		return
	}

	mapWebhookSubscriptionToState(sub, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WebhookSubscriptionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan WebhookSubscriptionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateWebhookSubscriptionRequest{}

	if !plan.FilterExpression.IsNull() {
		v := plan.FilterExpression.ValueString()
		updateReq.FilterExpression = &v
	}
	if !plan.LabelFilters.IsNull() {
		updateReq.LabelFilters = json.RawMessage(plan.LabelFilters.ValueString())
	}
	if !plan.LabelFilterMode.IsNull() {
		v := plan.LabelFilterMode.ValueString()
		updateReq.LabelFilterMode = &v
	}
	if !plan.TransformID.IsNull() {
		v := plan.TransformID.ValueString()
		updateReq.TransformID = &v
	}
	if !plan.IsEnabled.IsNull() {
		v := plan.IsEnabled.ValueBool()
		updateReq.IsEnabled = &v
	}
	if !plan.PriorityOverride.IsNull() {
		v := int(plan.PriorityOverride.ValueInt64())
		updateReq.PriorityOverride = &v
	}

	sub, err := r.client.UpdateWebhookSubscription(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook subscription", err.Error())
		return
	}

	mapWebhookSubscriptionToState(sub, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookSubscriptionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state WebhookSubscriptionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteWebhookSubscription(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting webhook subscription", err.Error())
	}
}

func (r *WebhookSubscriptionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapWebhookSubscriptionToState(sub *client.WebhookSubscription, state *WebhookSubscriptionResourceModel, _ *diag.Diagnostics) {
	state.ID = types.StringValue(sub.ID)
	state.EndpointID = types.StringValue(sub.EndpointID)
	state.EventTypeID = types.StringValue(sub.EventTypeID)
	state.FilterExpression = stringPtrToValue(sub.FilterExpression)
	state.LabelFilterMode = types.StringValue(sub.LabelFilterMode)
	state.TransformID = stringPtrToValue(sub.TransformID)
	state.IsEnabled = types.BoolValue(sub.IsEnabled)
	if sub.CreatedAt != "" {
		state.CreatedAt = types.StringValue(sub.CreatedAt)
	}

	if sub.PriorityOverride != nil {
		state.PriorityOverride = types.Int64Value(int64(*sub.PriorityOverride))
	} else {
		state.PriorityOverride = types.Int64Null()
	}

	if sub.LabelFilters != nil && len(sub.LabelFilters) > 0 && string(sub.LabelFilters) != "null" {
		state.LabelFilters = types.StringValue(string(sub.LabelFilters))
	} else {
		state.LabelFilters = types.StringNull()
	}
}
