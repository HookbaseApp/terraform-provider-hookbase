package resources

import (
	"context"

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

// metaVerifyTokenProviders are the providers whose sources carry a Meta "Verify token"
// (META_PROVIDERS in api/src/utils/verifyToken.ts). Only these are asked for one on read: the
// API answers GET sources/:id/verify-token with a 400 for every other provider.
var metaVerifyTokenProviders = map[string]bool{
	"whatsapp": true,
}

var (
	_ resource.Resource                = &SourceResource{}
	_ resource.ResourceWithImportState = &SourceResource{}
	_ resource.ResourceWithModifyPlan  = &SourceResource{}
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
	VerifyToken             types.String `tfsdk:"verify_token"`
	HasSigningSecret        types.Bool   `tfsdk:"has_signing_secret"`
	VerifiesWithoutSecret   types.Bool   `tfsdk:"verifies_without_secret"`
	AwaitingSigningSecret   types.Bool   `tfsdk:"awaiting_signing_secret"`
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
	AllowedMethods          types.List   `tfsdk:"allowed_methods"`
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
				Description: "Webhook provider, which selects the signature scheme used to verify incoming requests. " +
					"The value is the integration's provider id, as shown for each integration at " +
					"https://www.hookbase.app/integrations (for example github, stripe or standard-webhooks). " +
					"Omit for a source that accepts unsigned requests. The API validates the value, so an " +
					"unsupported provider fails at apply with the API's error. Changing it updates the source " +
					"in place; switching to a provider whose secret the vendor issues needs signing_secret set " +
					"in the same change.",
				Optional: true,
			},
			"description": schema.StringAttribute{
				Description: "Source description.",
				Optional:    true,
			},
			"signing_secret": schema.StringAttribute{
				Description: "Webhook signing secret, or for public-key providers the vendor's public key. " +
					"Whether to set it depends on the provider (see each integration at " +
					"https://www.hookbase.app/integrations): " +
					"for providers where Hookbase issues the secret (e.g. github, gitlab, bitbucket, typeform, " +
					"or no provider) leave it unset and Hookbase generates one, which is stored in state at " +
					"create so you can paste it into the vendor; " +
					"for providers where the vendor issues the secret or public key (e.g. stripe, shopify, " +
					"slack, standard-webhooks/svix, airtable, gitlab-signing-token, discord, sendgrid) copy it " +
					"from the vendor — public keys and format-checked keys (standard-webhooks/svix, airtable, " +
					"gitlab-signing-token) are rejected without it, and the rest are created with no secret " +
					"(awaiting_signing_secret is true, events unverified) until it is set; " +
					"asana and notion issue it only after the ingest URL is registered, so leave it unset at " +
					"create (awaiting_signing_secret is true) and set it once received; " +
					"wise, wise-sandbox and aws-sns verify against keys the vendor publishes, so it must be " +
					"omitted (verifies_without_secret is true). " +
					"Null in state when the source has no secret. Removing it from configuration leaves the " +
					"stored secret unchanged.",
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"verify_token": schema.StringAttribute{
				Description: "Meta \"Verify token\" for WhatsApp sources (provider_type = \"whatsapp\" only; the API " +
					"rejects it for any other provider). Copy it into Meta's webhook \"Verify token\" field. " +
					"Generated by Hookbase if not set (8-256 visible ASCII characters when set). Null for " +
					"providers without one.",
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				Validators: []validator.String{
					// Mirrors the API's format check. An empty string in particular would make the API
					// generate a fresh token, which can never match the planned "".
					stringvalidator.LengthBetween(8, 256),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"has_signing_secret": schema.BoolAttribute{
				Description: "Whether a signing secret is stored for this source.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"verifies_without_secret": schema.BoolAttribute{
				Description: "True when the provider signs with a key it publishes (wise, wise-sandbox, aws-sns), " +
					"so no signing secret is stored or needed.",
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"awaiting_signing_secret": schema.BoolAttribute{
				Description: "True when no secret is set and only the vendor can issue one (e.g. stripe or " +
					"shopify created without signing_secret, or asana/notion, which issue it only after the " +
					"ingest URL is registered); events are accepted unverified until it is set.",
				Computed: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
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
			"allowed_methods": schema.ListAttribute{
				Description: "HTTP methods the ingest endpoint accepts: GET, POST, PUT, PATCH, DELETE, HEAD. " +
					"Omit or set to an empty list to accept any method.",
				Optional:    true,
				ElementType: types.StringType,
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

	createReq, diags := buildCreateSourceRequest(ctx, plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateSource(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating source", err.Error())
		return
	}

	// Re-read to get full object with timestamps
	source, err := r.client.GetSource(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading source after create", err.Error())
		return
	}

	mapSourceToState(ctx, source, &plan, &resp.Diagnostics)
	applyCreatedCredentials(created, &plan)
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

	mapSourceToState(ctx, source, &state, &resp.Diagnostics)
	state.SigningSecret = r.readSigningSecret(ctx, source, state.SigningSecret)
	state.VerifyToken = r.readVerifyToken(ctx, source, state.VerifyToken)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state SourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq, diags := buildUpdateSourceRequest(ctx, plan, state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// PATCH answers {success:true} only, so the reply carries nothing to map.
	if _, err := r.client.UpdateSource(ctx, plan.ID.ValueString(), updateReq); err != nil {
		resp.Diagnostics.AddError("Error updating source", err.Error())
		return
	}

	// Re-read to get full object with timestamps
	source, err := r.client.GetSource(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading source after update", err.Error())
		return
	}

	plannedSecret, plannedToken := plan.SigningSecret, plan.VerifyToken
	mapSourceToState(ctx, source, &plan, &resp.Diagnostics)

	// A credential that was sent is stored as sent. One that wasn't keeps its planned value when
	// that is known (it equals the prior state), and is read back from the API only when the plan
	// left it unknown — a provider change, after which the API may have cleared the secret
	// (built-in and secret-after-registration providers) or generated a verify token (WhatsApp).
	switch {
	case updateReq.SigningSecret != nil:
		plan.SigningSecret = types.StringValue(*updateReq.SigningSecret)
	case plannedSecret.IsUnknown():
		plan.SigningSecret = r.readSigningSecret(ctx, source, types.StringNull())
	default:
		plan.SigningSecret = plannedSecret
	}
	switch {
	case updateReq.VerifyToken != nil:
		plan.VerifyToken = types.StringValue(*updateReq.VerifyToken)
	case plannedToken.IsUnknown():
		plan.VerifyToken = r.readVerifyToken(ctx, source, types.StringNull())
	default:
		plan.VerifyToken = plannedToken
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// ModifyPlan marks the values a provider_type change can alter server-side as unknown, so the
// plan doesn't promise the prior state for them: the API clears the secret when switching to a
// provider that takes none (or issues it after registration), generates a verify token when
// switching to WhatsApp, and derives the three status flags from provider and secret.
func (r *SourceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Create or destroy: nothing in state to carry over.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state SourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configSecret, configToken types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("signing_secret"), &configSecret)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("verify_token"), &configToken)...)
	if resp.Diagnostics.HasError() {
		return
	}

	providerChanged := !plan.ProviderType.Equal(state.ProviderType)
	if providerChanged {
		if configSecret.IsNull() {
			plan.SigningSecret = types.StringUnknown()
		}
		if configToken.IsNull() {
			plan.VerifyToken = types.StringUnknown()
		}
	}
	if providerChanged || !plan.SigningSecret.Equal(state.SigningSecret) {
		plan.HasSigningSecret = types.BoolUnknown()
		plan.VerifiesWithoutSecret = types.BoolUnknown()
		plan.AwaitingSigningSecret = types.BoolUnknown()
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// readSigningSecret returns the source's secret for state. A source with no secret (built-in
// providers, asana/notion before the vendor issues one, an import of either) is null — never "".
// When the reveal call fails, prior is kept rather than guessing.
func (r *SourceResource) readSigningSecret(ctx context.Context, source *client.Source, prior types.String) types.String {
	if !source.HasSigningSecret {
		return types.StringNull()
	}
	secret, err := r.client.RevealSourceSecret(ctx, source.ID)
	if err != nil || secret == "" {
		return prior
	}
	return types.StringValue(secret)
}

// readVerifyToken returns a WhatsApp source's verify token for state, and null for every other
// provider. When the read fails (e.g. an API key without write scope), prior is kept.
func (r *SourceResource) readVerifyToken(ctx context.Context, source *client.Source, prior types.String) types.String {
	if source.Provider == nil || !metaVerifyTokenProviders[*source.Provider] {
		return types.StringNull()
	}
	token, err := r.client.GetSourceVerifyToken(ctx, source.ID)
	if err != nil {
		return prior
	}
	if token == "" {
		return types.StringNull()
	}
	return types.StringValue(token)
}

// applyCreatedCredentials sets signing_secret and verify_token after create. A configured value
// is kept as planned; otherwise the create reply is the one moment the API returns them in full.
// A null in the reply (wise, wise-sandbox, aws-sns, asana, notion; any non-WhatsApp verify token)
// becomes null in state, so no attribute is left unknown after apply.
func applyCreatedCredentials(created *client.Source, state *SourceResourceModel) {
	if state.SigningSecret.IsNull() || state.SigningSecret.IsUnknown() {
		state.SigningSecret = stringPtrToValue(nonEmpty(created.SigningSecret))
	}
	if state.VerifyToken.IsNull() || state.VerifyToken.IsUnknown() {
		state.VerifyToken = stringPtrToValue(nonEmpty(created.VerifyToken))
	}
}

func buildCreateSourceRequest(ctx context.Context, plan SourceResourceModel) (client.CreateSourceRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	createReq := client.CreateSourceRequest{
		Name: plan.Name.ValueString(),
		Slug: plan.Slug.ValueString(),
	}

	createReq.Provider = knownString(plan.ProviderType)
	createReq.Description = knownString(plan.Description)
	createReq.SigningSecret = knownString(plan.SigningSecret)
	createReq.VerifyToken = knownString(plan.VerifyToken)
	createReq.IPFilterMode = knownString(plan.IPFilterMode)
	createReq.DedupStrategy = knownString(plan.DedupStrategy)
	createReq.DedupCustomHeader = knownString(plan.DedupCustomHeader)
	createReq.RejectInvalidSignatures = knownBool(plan.RejectInvalidSignatures)
	createReq.DedupEnabled = knownBool(plan.DedupEnabled)
	createReq.TransientMode = knownBool(plan.TransientMode)
	createReq.RateLimitPerMinute = knownInt(plan.RateLimitPerMinute)
	createReq.DedupWindowHours = knownInt(plan.DedupWindowHours)

	diags.Append(stringListToSlice(ctx, plan.IPAllowlist, &createReq.IPAllowlist)...)
	diags.Append(stringListToSlice(ctx, plan.IPDenylist, &createReq.IPDenylist)...)
	diags.Append(stringListToSlice(ctx, plan.EncryptFields, &createReq.EncryptFields)...)
	diags.Append(stringListToSlice(ctx, plan.MaskFields, &createReq.MaskFields)...)
	diags.Append(stringListToSlice(ctx, plan.AllowedMethods, &createReq.AllowedMethods)...)
	return createReq, diags
}

// buildUpdateSourceRequest builds the PATCH body. Ordinary fields are always sent, with JSON null
// for an optional attribute removed from configuration so it is cleared rather than left behind.
// provider, signingSecret and verifyToken are sent only when they change, because the API acts on
// their mere presence: a null provider wipes the secret and the reject flag, and switching
// provider re-validates (or clears) the secret. When the provider changes, a configured secret or
// token is re-sent so it survives — and is checked against — the new provider.
func buildUpdateSourceRequest(ctx context.Context, plan, state SourceResourceModel) (client.UpdateSourceRequest, diag.Diagnostics) {
	var diags diag.Diagnostics
	updateReq := client.UpdateSourceRequest{
		Name:                    knownString(plan.Name),
		Description:             nullableString(plan.Description),
		RejectInvalidSignatures: knownBool(plan.RejectInvalidSignatures),
		RateLimitPerMinute:      nullableInt(plan.RateLimitPerMinute),
		IsActive:                knownBool(plan.IsActive),
		IPFilterMode:            knownString(plan.IPFilterMode),
		DedupEnabled:            knownBool(plan.DedupEnabled),
		DedupStrategy:           knownString(plan.DedupStrategy),
		DedupWindowHours:        knownInt(plan.DedupWindowHours),
		DedupCustomHeader:       nullableString(plan.DedupCustomHeader),
		TransientMode:           knownBool(plan.TransientMode),
	}

	providerChanged := !plan.ProviderType.Equal(state.ProviderType)
	if providerChanged && !plan.ProviderType.IsUnknown() {
		updateReq.Provider = nullableString(plan.ProviderType)
	}
	if v := knownString(plan.SigningSecret); v != nil && (providerChanged || !plan.SigningSecret.Equal(state.SigningSecret)) {
		updateReq.SigningSecret = v
	}
	if v := knownString(plan.VerifyToken); v != nil && (providerChanged || !plan.VerifyToken.Equal(state.VerifyToken)) {
		updateReq.VerifyToken = v
	}

	updateReq.IPAllowlist = nullableList(ctx, plan.IPAllowlist, &diags)
	updateReq.IPDenylist = nullableList(ctx, plan.IPDenylist, &diags)
	updateReq.EncryptFields = nullableList(ctx, plan.EncryptFields, &diags)
	updateReq.MaskFields = nullableList(ctx, plan.MaskFields, &diags)
	updateReq.AllowedMethods = nullableList(ctx, plan.AllowedMethods, &diags)
	return updateReq, diags
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

// mapSourceToState copies the API's view of a source into state. It leaves signing_secret and
// verify_token alone: GET redacts both, so each operation decides where they come from.
func mapSourceToState(ctx context.Context, source *client.Source, state *SourceResourceModel, diags *diag.Diagnostics) {
	state.ID = types.StringValue(source.ID)
	state.Name = types.StringValue(source.Name)
	state.Slug = types.StringValue(source.Slug)
	state.ProviderType = stringPtrToValue(source.Provider)
	state.Description = stringPtrToValue(source.Description)
	state.HasSigningSecret = types.BoolValue(source.HasSigningSecret)
	state.VerifiesWithoutSecret = types.BoolValue(source.VerifiesWithoutSecret)
	state.AwaitingSigningSecret = types.BoolValue(source.AwaitingSigningSecret)
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

	if source.RateLimitPerMinute != nil {
		state.RateLimitPerMinute = types.Int64Value(int64(*source.RateLimitPerMinute))
	} else {
		state.RateLimitPerMinute = types.Int64Null()
	}

	state.IPAllowlist = sliceToStringList(ctx, source.IPAllowlist, diags)
	state.IPDenylist = sliceToStringList(ctx, source.IPDenylist, diags)
	state.EncryptFields = sliceToStringList(ctx, source.EncryptFields, diags)
	state.MaskFields = sliceToStringList(ctx, source.MaskFields, diags)
	state.AllowedMethods = sliceToStringList(ctx, source.AllowedMethods, diags)
}

// Request-building helpers: known* return nil (omit) for a null or unknown value; nullable*
// return client.Null for null (send JSON null to clear) and nil only for unknown.

func knownString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func knownBool(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func knownInt(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := int(v.ValueInt64())
	return &i
}

func nullableString(v types.String) **string {
	switch {
	case v.IsUnknown():
		return nil
	case v.IsNull():
		return client.Null[string]()
	}
	return client.Set(v.ValueString())
}

func nullableInt(v types.Int64) **int {
	switch {
	case v.IsUnknown():
		return nil
	case v.IsNull():
		return client.Null[int]()
	}
	return client.Set(int(v.ValueInt64()))
}

// nullableList returns a pointer to the list's items; a null list points at a nil slice, which
// marshals as JSON null.
func nullableList(ctx context.Context, list types.List, diags *diag.Diagnostics) *[]string {
	if list.IsUnknown() {
		return nil
	}
	var items []string
	diags.Append(stringListToSlice(ctx, list, &items)...)
	return &items
}

// nonEmpty treats an empty string like an absent one.
func nonEmpty(s *string) *string {
	if s == nil || *s == "" {
		return nil
	}
	return s
}

// Helpers used across resources

func stringPtrToValue(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
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
	if len(items) == 0 {
		return types.ListNull(types.StringType)
	}
	list, d := types.ListValueFrom(ctx, types.StringType, items)
	diags.Append(d...)
	return list
}
