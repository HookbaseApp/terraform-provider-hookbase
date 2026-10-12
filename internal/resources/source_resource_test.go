package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

// newSourceModel returns a model with every attribute null (lists typed), the shape Terraform
// hands Create for a minimal config once defaults have been applied.
func newSourceModel(name, slug string) SourceResourceModel {
	nullList := types.ListNull(types.StringType)
	return SourceResourceModel{
		ID:                      types.StringUnknown(),
		Name:                    types.StringValue(name),
		Slug:                    types.StringValue(slug),
		ProviderType:            types.StringNull(),
		Description:             types.StringNull(),
		SigningSecret:           types.StringUnknown(),
		VerifyToken:             types.StringUnknown(),
		HasSigningSecret:        types.BoolUnknown(),
		VerifiesWithoutSecret:   types.BoolUnknown(),
		AwaitingSigningSecret:   types.BoolUnknown(),
		RejectInvalidSignatures: types.BoolValue(false),
		RateLimitPerMinute:      types.Int64Null(),
		IsActive:                types.BoolValue(true),
		IPFilterMode:            types.StringValue("none"),
		IPAllowlist:             nullList,
		IPDenylist:              nullList,
		EncryptFields:           nullList,
		MaskFields:              nullList,
		DedupEnabled:            types.BoolValue(false),
		DedupStrategy:           types.StringValue("auto"),
		DedupWindowHours:        types.Int64Value(24),
		DedupCustomHeader:       types.StringNull(),
		TransientMode:           types.BoolValue(false),
		AllowedMethods:          nullList,
		IngestURL:               types.StringUnknown(),
		CreatedAt:               types.StringUnknown(),
		UpdatedAt:               types.StringUnknown(),
	}
}

// fakeSourceAPI is a minimal in-memory stand-in for the source endpoints the resource calls.
type fakeSourceAPI struct {
	t            *testing.T
	createReply  client.Source
	source       client.Source
	secret       string // reveal-secret answer; "" answers 404 like the API
	revealStatus int    // overrides the reveal-secret status when non-zero
	verifyToken  string
	calls        []string
	lastPatch    map[string]any
	lastCreate   map[string]any
}

func (f *fakeSourceAPI) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/api/organizations/org-1")
		f.calls = append(f.calls, r.Method+" "+p)
		w.Header().Set("Content-Type", "application/json")
		writeJSON := func(status int, v any) {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(v)
		}
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPost && p == "/sources":
			_ = json.Unmarshal(body, &f.lastCreate)
			writeJSON(http.StatusCreated, map[string]any{"source": f.createReply})
		case r.Method == http.MethodGet && p == "/sources/"+f.source.ID:
			writeJSON(http.StatusOK, map[string]any{"source": f.source})
		case r.Method == http.MethodPatch && p == "/sources/"+f.source.ID:
			f.lastPatch = map[string]any{}
			_ = json.Unmarshal(body, &f.lastPatch)
			writeJSON(http.StatusOK, map[string]any{"success": true})
		case r.Method == http.MethodGet && p == "/sources/"+f.source.ID+"/reveal-secret":
			if f.revealStatus != 0 {
				writeJSON(f.revealStatus, map[string]any{"error": "denied"})
				return
			}
			if f.secret == "" {
				writeJSON(http.StatusNotFound, map[string]any{"error": "Source not found or no signing secret configured"})
				return
			}
			writeJSON(http.StatusOK, map[string]any{"signingSecret": f.secret})
		case r.Method == http.MethodGet && p == "/sources/"+f.source.ID+"/verify-token":
			if f.source.Provider == nil || *f.source.Provider != "whatsapp" {
				writeJSON(http.StatusBadRequest, map[string]any{"error": "no verify token"})
				return
			}
			writeJSON(http.StatusOK, map[string]any{"verifyToken": f.verifyToken})
		default:
			f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			writeJSON(http.StatusNotFound, map[string]any{"error": "not found"})
		}
	}))
}

func (f *fakeSourceAPI) called(call string) bool {
	for _, c := range f.calls {
		if c == call {
			return true
		}
	}
	return false
}

func newTestSourceResource(t *testing.T, f *fakeSourceAPI) *SourceResource {
	t.Helper()
	srv := f.server()
	t.Cleanup(srv.Close)
	return &SourceResource{client: &client.Client{
		BaseURL:        srv.URL,
		OrganizationID: "org-1",
		APIKey:         "key",
		HTTPClient:     srv.Client(),
	}}
}

func sourceSchema(t *testing.T, r *SourceResource) resource.SchemaResponse {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	return resp
}

func emptyState(t *testing.T, r *SourceResource) tfsdk.State {
	s := sourceSchema(t, r).Schema
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
}

func planFrom(t *testing.T, r *SourceResource, m SourceResourceModel) tfsdk.Plan {
	t.Helper()
	s := sourceSchema(t, r).Schema
	plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
	if d := plan.Set(context.Background(), &m); d.HasError() {
		t.Fatalf("plan.Set: %v", d)
	}
	return plan
}

func stateFrom(t *testing.T, r *SourceResource, m SourceResourceModel) tfsdk.State {
	t.Helper()
	st := emptyState(t, r)
	if d := st.Set(context.Background(), &m); d.HasError() {
		t.Fatalf("state.Set: %v", d)
	}
	return st
}

func readModel(t *testing.T, st tfsdk.State) SourceResourceModel {
	t.Helper()
	var m SourceResourceModel
	if d := st.Get(context.Background(), &m); d.HasError() {
		t.Fatalf("state.Get: %v", d)
	}
	if !st.Raw.IsFullyKnown() {
		t.Fatal("state has unknown values after the operation")
	}
	return m
}

func baseSource(provider string) client.Source {
	s := client.Source{
		ID:            "src_1",
		Name:          "s",
		Slug:          "s",
		IsActive:      true,
		IPFilterMode:  "none",
		DedupStrategy: "auto",
		CreatedAt:     "2026-10-01T00:00:00Z",
		UpdatedAt:     "2026-10-01T00:00:00Z",
	}
	if provider != "" {
		s.Provider = &provider
	}
	return s
}

func TestSourceCreate_NoSecretProvidersLeaveSecretNull(t *testing.T) {
	cases := []struct {
		provider              string
		awaiting, withoutSecr bool
	}{
		{"asana", true, false},
		{"notion", true, false},
		{"wise", false, true},
		{"wise-sandbox", false, true},
		{"aws-sns", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.provider, func(t *testing.T) {
			src := baseSource(tc.provider)
			src.AwaitingSigningSecret = tc.awaiting
			src.VerifiesWithoutSecret = tc.withoutSecr
			reply := src
			reply.SigningSecret = nil // the API's create reply carries "signingSecret": null
			f := &fakeSourceAPI{t: t, createReply: reply, source: src}
			r := newTestSourceResource(t, f)

			m := newSourceModel("s", "s")
			m.ProviderType = types.StringValue(tc.provider)
			resp := resource.CreateResponse{State: emptyState(t, r)}
			r.Create(context.Background(), resource.CreateRequest{Plan: planFrom(t, r, m)}, &resp)
			if resp.Diagnostics.HasError() {
				t.Fatalf("Create: %v", resp.Diagnostics)
			}

			got := readModel(t, resp.State)
			if !got.SigningSecret.IsNull() {
				t.Errorf("signing_secret = %v, want null", got.SigningSecret)
			}
			if !got.VerifyToken.IsNull() {
				t.Errorf("verify_token = %v, want null", got.VerifyToken)
			}
			if got.AwaitingSigningSecret.ValueBool() != tc.awaiting || got.VerifiesWithoutSecret.ValueBool() != tc.withoutSecr {
				t.Errorf("awaiting=%v verifiesWithoutSecret=%v, want %v/%v",
					got.AwaitingSigningSecret, got.VerifiesWithoutSecret, tc.awaiting, tc.withoutSecr)
			}
			if _, sent := f.lastCreate["signingSecret"]; sent {
				t.Error("create body carried signingSecret for an unset attribute")
			}
		})
	}
}

func TestSourceCreate_OperatorSecretFromCreateReply(t *testing.T) {
	src := baseSource("github")
	src.HasSigningSecret = true
	reply := src
	reply.SigningSecret = strp("generated-secret")
	f := &fakeSourceAPI{t: t, createReply: reply, source: src}
	r := newTestSourceResource(t, f)

	m := newSourceModel("s", "s")
	m.ProviderType = types.StringValue("github")
	resp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(context.Background(), resource.CreateRequest{Plan: planFrom(t, r, m)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	got := readModel(t, resp.State)
	if got.SigningSecret.ValueString() != "generated-secret" {
		t.Errorf("signing_secret = %v, want generated-secret", got.SigningSecret)
	}
	if !got.HasSigningSecret.ValueBool() {
		t.Error("has_signing_secret = false, want true")
	}
}

func TestSourceCreate_ConfiguredSecretAndWhatsAppToken(t *testing.T) {
	src := baseSource("whatsapp")
	src.HasSigningSecret = true
	reply := src
	reply.SigningSecret = strp("app-secret")
	reply.VerifyToken = strp("generated-verify-token")
	f := &fakeSourceAPI{t: t, createReply: reply, source: src}
	r := newTestSourceResource(t, f)

	m := newSourceModel("s", "s")
	m.ProviderType = types.StringValue("whatsapp")
	m.SigningSecret = types.StringValue("app-secret")
	resp := resource.CreateResponse{State: emptyState(t, r)}
	r.Create(context.Background(), resource.CreateRequest{Plan: planFrom(t, r, m)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	got := readModel(t, resp.State)
	if got.SigningSecret.ValueString() != "app-secret" {
		t.Errorf("signing_secret = %v, want app-secret", got.SigningSecret)
	}
	if got.VerifyToken.ValueString() != "generated-verify-token" {
		t.Errorf("verify_token = %v, want generated-verify-token", got.VerifyToken)
	}
	if f.lastCreate["signingSecret"] != "app-secret" {
		t.Errorf("create body signingSecret = %v", f.lastCreate["signingSecret"])
	}
	if _, sent := f.lastCreate["verifyToken"]; sent {
		t.Error("create body carried verifyToken for an unset attribute")
	}
}

func TestSourceRead_Import(t *testing.T) {
	t.Run("no secret stays null and skips reveal", func(t *testing.T) {
		src := baseSource("wise")
		src.VerifiesWithoutSecret = true
		f := &fakeSourceAPI{t: t, source: src}
		r := newTestSourceResource(t, f)

		resp := resource.ReadResponse{State: emptyState(t, r)}
		r.Read(context.Background(), resource.ReadRequest{State: importState(t, r, "src_1")}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", resp.Diagnostics)
		}
		got := readModel(t, resp.State)
		if !got.SigningSecret.IsNull() || !got.VerifyToken.IsNull() {
			t.Errorf("signing_secret=%v verify_token=%v, want both null", got.SigningSecret, got.VerifyToken)
		}
		if f.called("GET /sources/src_1/reveal-secret") || f.called("GET /sources/src_1/verify-token") {
			t.Errorf("unexpected credential reads: %v", f.calls)
		}
	})

	t.Run("whatsapp reveals secret and verify token", func(t *testing.T) {
		src := baseSource("whatsapp")
		src.HasSigningSecret = true
		f := &fakeSourceAPI{t: t, source: src, secret: "app-secret", verifyToken: "vt-12345678"}
		r := newTestSourceResource(t, f)

		resp := resource.ReadResponse{State: emptyState(t, r)}
		r.Read(context.Background(), resource.ReadRequest{State: importState(t, r, "src_1")}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", resp.Diagnostics)
		}
		got := readModel(t, resp.State)
		if got.SigningSecret.ValueString() != "app-secret" || got.VerifyToken.ValueString() != "vt-12345678" {
			t.Errorf("signing_secret=%v verify_token=%v", got.SigningSecret, got.VerifyToken)
		}
	})

	t.Run("reveal failure keeps prior state", func(t *testing.T) {
		src := baseSource("github")
		src.HasSigningSecret = true
		f := &fakeSourceAPI{t: t, source: src, revealStatus: http.StatusForbidden}
		r := newTestSourceResource(t, f)

		prior := newSourceModel("s", "s")
		prior.ID = types.StringValue("src_1")
		prior.SigningSecret = types.StringValue("kept")
		prior.VerifyToken = types.StringNull()
		prior.HasSigningSecret = types.BoolValue(true)
		prior.VerifiesWithoutSecret = types.BoolValue(false)
		prior.AwaitingSigningSecret = types.BoolValue(false)
		prior.IngestURL = types.StringNull()
		prior.CreatedAt = types.StringNull()
		prior.UpdatedAt = types.StringNull()
		resp := resource.ReadResponse{State: emptyState(t, r)}
		r.Read(context.Background(), resource.ReadRequest{State: stateFrom(t, r, prior)}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Read: %v", resp.Diagnostics)
		}
		if got := readModel(t, resp.State); got.SigningSecret.ValueString() != "kept" {
			t.Errorf("signing_secret = %v, want kept", got.SigningSecret)
		}
	})
}

// importState mirrors ImportStatePassthroughID: only the id is known, everything else null.
func importState(t *testing.T, r *SourceResource, id string) tfsdk.State {
	t.Helper()
	st := emptyState(t, r)
	nullList := types.ListNull(types.StringType)
	m := SourceResourceModel{
		ID: types.StringValue(id), IPAllowlist: nullList, IPDenylist: nullList,
		EncryptFields: nullList, MaskFields: nullList, AllowedMethods: nullList,
	}
	if d := st.Set(context.Background(), &m); d.HasError() {
		t.Fatalf("state.Set: %v", d)
	}
	return st
}

// appliedModel is the state after a create of a github source with a generated secret.
func appliedModel() SourceResourceModel {
	m := newSourceModel("s", "s")
	m.ID = types.StringValue("src_1")
	m.ProviderType = types.StringValue("github")
	m.Description = types.StringValue("old description")
	m.RateLimitPerMinute = types.Int64Value(60)
	m.SigningSecret = types.StringValue("generated")
	m.VerifyToken = types.StringNull()
	m.HasSigningSecret = types.BoolValue(true)
	m.VerifiesWithoutSecret = types.BoolValue(false)
	m.AwaitingSigningSecret = types.BoolValue(false)
	m.IngestURL = types.StringNull()
	m.CreatedAt = types.StringValue("2026-10-01T00:00:00Z")
	m.UpdatedAt = types.StringValue("2026-10-01T00:00:00Z")
	m.IPAllowlist, _ = types.ListValueFrom(context.Background(), types.StringType, []string{"10.0.0.0/8"})
	return m
}

func marshalUpdate(t *testing.T, plan, state SourceResourceModel) map[string]any {
	t.Helper()
	req, diags := buildUpdateSourceRequest(context.Background(), plan, state)
	if diags.HasError() {
		t.Fatalf("buildUpdateSourceRequest: %v", diags)
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

func TestBuildUpdateSourceRequest_ClearsRemovedOptionals(t *testing.T) {
	state := appliedModel()
	plan := state
	plan.Description = types.StringNull()
	plan.RateLimitPerMinute = types.Int64Null()
	plan.IPAllowlist = types.ListNull(types.StringType)

	body := marshalUpdate(t, plan, state)
	for _, k := range []string{"description", "rateLimitPerMinute", "ipAllowlist", "dedupCustomHeader", "allowedMethods"} {
		v, ok := body[k]
		if !ok || v != nil {
			t.Errorf("%s = %v (present=%v), want JSON null", k, v, ok)
		}
	}
	for _, k := range []string{"provider", "signingSecret", "verifyToken"} {
		if _, ok := body[k]; ok {
			t.Errorf("%s sent although unchanged: %v", k, body)
		}
	}
}

func TestBuildUpdateSourceRequest_ProviderAndCredentials(t *testing.T) {
	state := appliedModel()

	t.Run("new secret is sent", func(t *testing.T) {
		plan := state
		plan.SigningSecret = types.StringValue("rotated")
		if body := marshalUpdate(t, plan, state); body["signingSecret"] != "rotated" {
			t.Errorf("signingSecret = %v, want rotated", body["signingSecret"])
		}
	})

	t.Run("unknown secret is not sent", func(t *testing.T) {
		plan := state
		plan.ProviderType = types.StringValue("asana")
		plan.SigningSecret = types.StringUnknown()
		body := marshalUpdate(t, plan, state)
		if body["provider"] != "asana" {
			t.Errorf("provider = %v, want asana", body["provider"])
		}
		if _, ok := body["signingSecret"]; ok {
			t.Error("signingSecret sent while unknown")
		}
	})

	t.Run("configured secret is re-sent on provider change", func(t *testing.T) {
		plan := state
		plan.ProviderType = types.StringValue("custom")
		body := marshalUpdate(t, plan, state)
		if body["signingSecret"] != "generated" {
			t.Errorf("signingSecret = %v, want generated", body["signingSecret"])
		}
	})

	t.Run("removed provider sends null", func(t *testing.T) {
		plan := state
		plan.ProviderType = types.StringNull()
		body := marshalUpdate(t, plan, state)
		if v, ok := body["provider"]; !ok || v != nil {
			t.Errorf("provider = %v (present=%v), want JSON null", v, ok)
		}
	})

	t.Run("verify token sent only when changed", func(t *testing.T) {
		ws := state
		ws.ProviderType = types.StringValue("whatsapp")
		ws.VerifyToken = types.StringValue("vt-old-token")
		plan := ws
		if _, ok := marshalUpdate(t, plan, ws)["verifyToken"]; ok {
			t.Error("verifyToken sent although unchanged")
		}
		plan.VerifyToken = types.StringValue("vt-new-token")
		if v := marshalUpdate(t, plan, ws)["verifyToken"]; v != "vt-new-token" {
			t.Errorf("verifyToken = %v, want vt-new-token", v)
		}
	})
}

func TestSourceUpdate_ProviderSwitchReadsCredentialsBack(t *testing.T) {
	// github -> asana: the API clears the old secret, so the plan's unknown secret resolves to null.
	src := baseSource("asana")
	src.AwaitingSigningSecret = true
	f := &fakeSourceAPI{t: t, source: src}
	r := newTestSourceResource(t, f)

	state := appliedModel()
	plan := state
	plan.ProviderType = types.StringValue("asana")
	plan.SigningSecret = types.StringUnknown()
	plan.VerifyToken = types.StringUnknown()
	plan.HasSigningSecret = types.BoolUnknown()
	plan.VerifiesWithoutSecret = types.BoolUnknown()
	plan.AwaitingSigningSecret = types.BoolUnknown()
	plan.Description = types.StringNull()
	plan.RateLimitPerMinute = types.Int64Null()
	plan.IPAllowlist = types.ListNull(types.StringType)

	resp := resource.UpdateResponse{State: emptyState(t, r)}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:  planFrom(t, r, plan),
		State: stateFrom(t, r, state),
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	got := readModel(t, resp.State)
	if !got.SigningSecret.IsNull() || !got.VerifyToken.IsNull() {
		t.Errorf("signing_secret=%v verify_token=%v, want null", got.SigningSecret, got.VerifyToken)
	}
	if !got.AwaitingSigningSecret.ValueBool() {
		t.Error("awaiting_signing_secret = false, want true")
	}
	if f.lastPatch["provider"] != "asana" {
		t.Errorf("PATCH provider = %v", f.lastPatch["provider"])
	}
	if v, ok := f.lastPatch["description"]; !ok || v != nil {
		t.Errorf("PATCH description = %v (present=%v), want null", v, ok)
	}
}

func TestSourceUpdate_UnchangedSecretKeptWithoutReveal(t *testing.T) {
	src := baseSource("github")
	src.HasSigningSecret = true
	f := &fakeSourceAPI{t: t, source: src, revealStatus: http.StatusForbidden}
	r := newTestSourceResource(t, f)

	state := appliedModel()
	plan := state
	plan.Name = types.StringValue("renamed")

	resp := resource.UpdateResponse{State: emptyState(t, r)}
	r.Update(context.Background(), resource.UpdateRequest{
		Plan:  planFrom(t, r, plan),
		State: stateFrom(t, r, state),
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	if got := readModel(t, resp.State); got.SigningSecret.ValueString() != "generated" {
		t.Errorf("signing_secret = %v, want generated", got.SigningSecret)
	}
	if _, ok := f.lastPatch["signingSecret"]; ok {
		t.Error("PATCH carried an unchanged signingSecret")
	}
}

func strp(s string) *string { return &s }

func TestSourceModifyPlan_ProviderChange(t *testing.T) {
	r := &SourceResource{}
	ctx := context.Background()
	s := sourceSchema(t, r).Schema
	state := appliedModel()

	run := func(t *testing.T, plan SourceResourceModel, configSecret types.String) SourceResourceModel {
		t.Helper()
		cfg := plan
		cfg.ID, cfg.SigningSecret, cfg.VerifyToken = types.StringNull(), configSecret, types.StringNull()
		cfg.HasSigningSecret, cfg.VerifiesWithoutSecret, cfg.AwaitingSigningSecret = types.BoolNull(), types.BoolNull(), types.BoolNull()
		cfg.IngestURL, cfg.CreatedAt, cfg.UpdatedAt = types.StringNull(), types.StringNull(), types.StringNull()
		config := tfsdk.Config{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)}
		// tfsdk.Config has no Set; round-trip through a State to build its raw value.
		config.Raw = stateFrom(t, r, cfg).Raw

		resp := resource.ModifyPlanResponse{Plan: planFrom(t, r, plan)}
		r.ModifyPlan(ctx, resource.ModifyPlanRequest{
			Config: config, Plan: planFrom(t, r, plan), State: stateFrom(t, r, state),
		}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ModifyPlan: %v", resp.Diagnostics)
		}
		var got SourceResourceModel
		resp.Plan.Get(ctx, &got)
		return got
	}

	t.Run("unchanged provider keeps state values", func(t *testing.T) {
		plan := state
		plan.Name = types.StringValue("renamed")
		got := run(t, plan, types.StringNull())
		if got.SigningSecret.ValueString() != "generated" || got.HasSigningSecret.IsUnknown() {
			t.Errorf("signing_secret=%v has_signing_secret=%v", got.SigningSecret, got.HasSigningSecret)
		}
	})

	t.Run("provider change with unset secret marks it unknown", func(t *testing.T) {
		plan := state
		plan.ProviderType = types.StringValue("wise")
		got := run(t, plan, types.StringNull())
		if !got.SigningSecret.IsUnknown() || !got.VerifyToken.IsUnknown() || !got.VerifiesWithoutSecret.IsUnknown() {
			t.Errorf("signing_secret=%v verify_token=%v verifies_without_secret=%v, want unknown",
				got.SigningSecret, got.VerifyToken, got.VerifiesWithoutSecret)
		}
	})

	t.Run("provider change keeps a configured secret", func(t *testing.T) {
		plan := state
		plan.ProviderType = types.StringValue("custom")
		got := run(t, plan, types.StringValue("generated"))
		if got.SigningSecret.ValueString() != "generated" {
			t.Errorf("signing_secret = %v, want generated", got.SigningSecret)
		}
	})
}
