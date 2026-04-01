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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

var (
	_ resource.Resource                = &SourceResource{}
	_ resource.ResourceWithImportState = &SourceResource{}
)

type SourceResource struct {
	client *client.Client
}

type SourceResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	Slug                    types.String `tfsdk:"slug"`
	ProviderType            types.String `tfsdk:"provider_type"`
	Description             types.String `tfsdk:"description"`
	SigningSecret           types.String `tfsdk:"signing_secret"`
	RejectInvalidSignatures types.Bool   `tfsdk:"reject_invalid_signatures"`
	RateLimitPerMinute      types.Int64  `tfsdk:"rate_limit_per_minute"`
	IsActive                types.Bool   `tfsdk:"is_active"`
	IPFilterMode            types.String `tfsdk:"ip_filter_mode"`
	IPAllowlist             types.List   `tfsdk:"ip_allowlist"`
	IPDenylist              types.List   `tfsdk:"ip_denylist"`
	EncryptFields           types.List   `tfsdk:"encrypt_fields"`
	MaskFields              types.List   `tfsdk:"mask_fields"`
	DedupEnabled            types.Bool   `tfsdk:"dedup_enabled"`
	DedupStrategy           types.String `tfsdk:"dedup_strategy"`
	DedupWindowHours        types.Int64  `tfsdk:"dedup_window_hours"`
	DedupCustomHeader       types.String `tfsdk:"dedup_custom_header"`
	TransientMode           types.Bool   `tfsdk:"transient_mode"`
	IngestURL               types.String `tfsdk:"ingest_url"`
	CreatedAt               types.String `tfsdk:"created_at"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
}

func NewSourceResource() resource.Resource {
	return &SourceResource{}
}

func (r *SourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source"
}

func (r *SourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Hookbase inbound webhook source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Source ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Source name (1-100 characters).",
				Required:    true,
			},
			"slug": schema.StringAttribute{
				Description: "URL-safe identifier. Lowercase alphanumeric and hyphens only. Changing this forces a new resource.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"provider_type": schema.StringAttribute{
				Description: "Webhook provider type: github, stripe, shopify, slack, twilio, custom, or generic.",
				Optional:    true,
			},
			"description": schema.StringAttribute{
				Description: "Source description.",
				Optional:    true,
			},
			"signing_secret": schema.StringAttribute{
				Description: "Webhook signing secret. Auto-generated if not provided.",
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"reject_invalid_signatures": schema.BoolAttribute{
				Description: "Reject webhooks with invalid signatures.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"rate_limit_per_minute": schema.Int64Attribute{
				Description: "Rate limit per minute (1-100000). Requires paid plan.",
				Optional:    true,
			},
			"is_active": schema.BoolAttribute{
				Description: "Whether the source is active.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"ip_filter_mode": schema.StringAttribute{
				Description: "IP filtering mode: none, allowlist, denylist, or both.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("none"),
			},
			"ip_allowlist": schema.ListAttribute{
				Description: "IP addresses or CIDR ranges to allow.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"ip_denylist": schema.ListAttribute{
				Description: "IP addresses or CIDR ranges to deny.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"encrypt_fields": schema.ListAttribute{
				Description: "Field names to encrypt.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"mask_fields": schema.ListAttribute{
				Description: "Field names to mask in the UI.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"dedup_enabled": schema.BoolAttribute{
				Description: "Enable event deduplication.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"dedup_strategy": schema.StringAttribute{
				Description: "Deduplication strategy: auto, provider_id, payload_hash, idempotency_key, or none.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("auto"),
			},
			"dedup_window_hours": schema.Int64Attribute{
				Description: "Deduplication window in hours (1-168).",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(24),
			},
			"dedup_custom_header": schema.StringAttribute{
				Description: "Custom header name for idempotency key dedup strategy.",
				Optional:    true,
			},
			"transient_mode": schema.BoolAttribute{
				Description: "Discard payloads after routing (HIPAA/GDPR compliance).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"ingest_url": schema.StringAttribute{
				Description: "The public URL for sending webhooks to this source.",
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

func (r *SourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createReq := client.CreateSourceRequest{
		Name: plan.Name.ValueString(),
		Slug: plan.Slug.ValueString(),
	}

	if !plan.ProviderType.IsNull() {
		v := plan.ProviderType.ValueString()
		createReq.Provider = &v
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		createReq.Description = &v
	}
	if !plan.SigningSecret.IsNull() {
		v := plan.SigningSecret.ValueString()
		createReq.SigningSecret = &v
	}
	if !plan.RejectInvalidSignatures.IsNull() {
		v := plan.RejectInvalidSignatures.ValueBool()
		createReq.RejectInvalidSignatures = &v
	}
	if !plan.RateLimitPerMinute.IsNull() {
		v := int(plan.RateLimitPerMinute.ValueInt64())
		createReq.RateLimitPerMinute = &v
	}
	if !plan.IPFilterMode.IsNull() {
		v := plan.IPFilterMode.ValueString()
		createReq.IPFilterMode = &v
	}
	if !plan.DedupEnabled.IsNull() {
		v := plan.DedupEnabled.ValueBool()
		createReq.DedupEnabled = &v
	}
	if !plan.DedupStrategy.IsNull() {
		v := plan.DedupStrategy.ValueString()
		createReq.DedupStrategy = &v
	}
	if !plan.DedupWindowHours.IsNull() {
		v := int(plan.DedupWindowHours.ValueInt64())
		createReq.DedupWindowHours = &v
	}
	if !plan.DedupCustomHeader.IsNull() {
		v := plan.DedupCustomHeader.ValueString()
		createReq.DedupCustomHeader = &v
	}
	if !plan.TransientMode.IsNull() {
		v := plan.TransientMode.ValueBool()
		createReq.TransientMode = &v
	}

	resp.Diagnostics.Append(stringListToSlice(ctx, plan.IPAllowlist, &createReq.IPAllowlist)...)
	resp.Diagnostics.Append(stringListToSlice(ctx, plan.IPDenylist, &createReq.IPDenylist)...)
	resp.Diagnostics.Append(stringListToSlice(ctx, plan.EncryptFields, &createReq.EncryptFields)...)
	resp.Diagnostics.Append(stringListToSlice(ctx, plan.MaskFields, &createReq.MaskFields)...)
	if resp.Diagnostics.HasError() {
		return
	}

	source, err := r.client.CreateSource(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating source", err.Error())
		return
	}

	// Save the signing secret from create (only time it's returned in full)
	signingSecret := source.SigningSecret

	// Re-read to get full object with timestamps
	source, err = r.client.GetSource(ctx, source.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading source after create", err.Error())
		return
	}
	source.SigningSecret = signingSecret

	mapSourceToState(ctx, source, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state SourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	source, err := r.client.GetSource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading source", err.Error())
		return
	}

	// Reveal the signing secret since API redacts it on normal GET
	secret, err := r.client.RevealSourceSecret(ctx, state.ID.ValueString())
	if err == nil && secret != "" {
		source.SigningSecret = &secret
	}

	mapSourceToState(ctx, source, &state, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan SourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := client.UpdateSourceRequest{}

	if !plan.Name.IsNull() {
		v := plan.Name.ValueString()
		updateReq.Name = &v
	}
	if !plan.Description.IsNull() {
		v := plan.Description.ValueString()
		updateReq.Description = &v
	}
	if !plan.RejectInvalidSignatures.IsNull() {
		v := plan.RejectInvalidSignatures.ValueBool()
		updateReq.RejectInvalidSignatures = &v
	}
	if !plan.RateLimitPerMinute.IsNull() {
		v := int(plan.RateLimitPerMinute.ValueInt64())
		updateReq.RateLimitPerMinute = &v
	}
	if !plan.IsActive.IsNull() {
		v := plan.IsActive.ValueBool()
		updateReq.IsActive = &v
	}
	if !plan.IPFilterMode.IsNull() {
		v := plan.IPFilterMode.ValueString()
		updateReq.IPFilterMode = &v
	}
	if !plan.DedupEnabled.IsNull() {
		v := plan.DedupEnabled.ValueBool()
		updateReq.DedupEnabled = &v
	}
	if !plan.DedupStrategy.IsNull() {
		v := plan.DedupStrategy.ValueString()
		updateReq.DedupStrategy = &v
	}
	if !plan.DedupWindowHours.IsNull() {
		v := int(plan.DedupWindowHours.ValueInt64())
		updateReq.DedupWindowHours = &v
	}
	if !plan.DedupCustomHeader.IsNull() {
		v := plan.DedupCustomHeader.ValueString()
		updateReq.DedupCustomHeader = &v
	}
	if !plan.TransientMode.IsNull() {
		v := plan.TransientMode.ValueBool()
		updateReq.TransientMode = &v
	}

	resp.Diagnostics.Append(stringListToSlice(ctx, plan.IPAllowlist, &updateReq.IPAllowlist)...)
	resp.Diagnostics.Append(stringListToSlice(ctx, plan.IPDenylist, &updateReq.IPDenylist)...)
	resp.Diagnostics.Append(stringListToSlice(ctx, plan.EncryptFields, &updateReq.EncryptFields)...)
	resp.Diagnostics.Append(stringListToSlice(ctx, plan.MaskFields, &updateReq.MaskFields)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.UpdateSource(ctx, plan.ID.ValueString(), updateReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating source", err.Error())
		return
	}

	// Re-read to get full object with timestamps
	source, err := r.client.GetSource(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading source after update", err.Error())
		return
	}

	// Preserve signing secret from state since GET response redacts it
	if !plan.SigningSecret.IsNull() && !plan.SigningSecret.IsUnknown() {
		source.SigningSecret = strPtr(plan.SigningSecret.ValueString())
	} else {
		// Try to reveal it
		secret, secretErr := r.client.RevealSourceSecret(ctx, source.ID)
		if secretErr == nil && secret != "" {
			source.SigningSecret = &secret
		}
	}

	mapSourceToState(ctx, source, &plan, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteSource(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error deleting source", err.Error())
	}
}

func (r *SourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func mapSourceToState(ctx context.Context, source *client.Source, state *SourceResourceModel, diags *diag.Diagnostics) {
	state.ID = types.StringValue(source.ID)
	state.Name = types.StringValue(source.Name)
	state.Slug = types.StringValue(source.Slug)
	state.ProviderType = stringPtrToValue(source.Provider)
	state.Description = stringPtrToValue(source.Description)
	state.RejectInvalidSignatures = types.BoolValue(source.RejectInvalidSignatures)
	state.IsActive = types.BoolValue(source.IsActive)
	state.IPFilterMode = types.StringValue(source.IPFilterMode)
	state.DedupEnabled = types.BoolValue(source.DedupEnabled)
	state.DedupStrategy = types.StringValue(source.DedupStrategy)
	state.DedupWindowHours = types.Int64Value(int64(source.DedupWindowHours))
	state.DedupCustomHeader = stringPtrToValue(source.DedupCustomHeader)
	state.TransientMode = types.BoolValue(source.TransientMode)
	state.IngestURL = stringPtrToValue(source.IngestURL)
	if source.CreatedAt != "" {
		state.CreatedAt = types.StringValue(source.CreatedAt)
	}
	if source.UpdatedAt != "" {
		state.UpdatedAt = types.StringValue(source.UpdatedAt)
	}

	if source.SigningSecret != nil {
		state.SigningSecret = types.StringValue(*source.SigningSecret)
	}

	if source.RateLimitPerMinute != nil {
		state.RateLimitPerMinute = types.Int64Value(int64(*source.RateLimitPerMinute))
	} else {
		state.RateLimitPerMinute = types.Int64Null()
	}

	state.IPAllowlist = sliceToStringList(ctx, source.IPAllowlist, diags)
	state.IPDenylist = sliceToStringList(ctx, source.IPDenylist, diags)
	state.EncryptFields = sliceToStringList(ctx, source.EncryptFields, diags)
	state.MaskFields = sliceToStringList(ctx, source.MaskFields, diags)
}

// Helpers used across resources

func stringPtrToValue(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func strPtr(s string) *string {
	return &s
}

func stringListToSlice(ctx context.Context, list types.List, target *[]string) diag.Diagnostics {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var items []string
	diags := list.ElementsAs(ctx, &items, false)
	*target = items
	return diags
}

func sliceToStringList(ctx context.Context, items []string, diags *diag.Diagnostics) types.List {
	if items == nil || len(items) == 0 {
		return types.ListNull(types.StringType)
	}
	list, d := types.ListValueFrom(ctx, types.StringType, items)
	diags.Append(d...)
	return list
}
