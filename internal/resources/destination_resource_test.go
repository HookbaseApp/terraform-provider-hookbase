package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

// fakeDestinationAPI answers the destination endpoints the resource calls, in the API's shapes:
// the create reply carries signingSecret, PATCH answers {success:true}, and reveal-secret is 400
// for a type without a secret and 404 when none is stored.
type fakeDestinationAPI struct {
	t            *testing.T
	createReply  client.Destination
	dest         client.Destination // what GET answers
	secret       string
	revealStatus int
	calls        []string
	lastCreate   map[string]any
	lastPatch    map[string]any
}

func (f *fakeDestinationAPI) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/api/organizations/org-1")
		f.calls = append(f.calls, r.Method+" "+p)
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPost && p == "/destinations":
			_ = json.Unmarshal(body, &f.lastCreate)
			writeTestJSON(w, http.StatusCreated, map[string]any{"destination": f.createReply})
		case r.Method == http.MethodGet && p == "/destinations/"+f.dest.ID:
			writeTestJSON(w, http.StatusOK, map[string]any{"destination": f.dest})
		case r.Method == http.MethodPatch && p == "/destinations/"+f.dest.ID:
			f.lastPatch = map[string]any{}
			_ = json.Unmarshal(body, &f.lastPatch)
			writeTestJSON(w, http.StatusOK, map[string]any{"success": true})
		case r.Method == http.MethodGet && p == "/destinations/"+f.dest.ID+"/reveal-secret":
			switch {
			case f.revealStatus != 0:
				writeTestJSON(w, f.revealStatus, map[string]any{"error": "denied"})
			case f.dest.Type != "http":
				writeTestJSON(w, http.StatusBadRequest, map[string]any{"error": "Only http destinations sign their deliveries"})
			case f.secret == "":
				writeTestJSON(w, http.StatusNotFound, map[string]any{"error": "No signing secret"})
			default:
				writeTestJSON(w, http.StatusOK, map[string]any{"signingSecret": f.secret, "previousSecretExpiresAt": nil})
			}
		default:
			f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			writeTestJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		}
	}))
}

func (f *fakeDestinationAPI) count(call string) int {
	n := 0
	for _, c := range f.calls {
		if c == call {
			n++
		}
	}
	return n
}

func newTestDestinationResource(t *testing.T, f *fakeDestinationAPI) *DestinationResource {
	t.Helper()
	return &DestinationResource{client: testClient(t, f.server())}
}

// newDestinationPlan is a create plan with every optional attribute unset, as Terraform builds it
// from the schema's defaults and plan modifiers.
func newDestinationPlan(destType string) DestinationResourceModel {
	return DestinationResourceModel{
		ID:                     types.StringUnknown(),
		Name:                   types.StringValue("d"),
		Slug:                   types.StringValue("d"),
		URL:                    types.StringNull(),
		Method:                 types.StringValue("POST"),
		Headers:                types.MapNull(types.StringType),
		AuthType:               types.StringValue("none"),
		AuthConfig:             types.MapNull(types.StringType),
		TimeoutMs:              types.Int64Value(30000),
		ThrottleMode:           types.StringValue("off"),
		ThrottleRateLimit:      types.Int64Null(),
		ThrottleRateUnit:       types.StringNull(),
		ThrottleMaxConcurrency: types.Int64Null(),
		ThrottleQueueLimit:     types.Int64Null(),
		Type:                   types.StringValue(destType),
		Config:                 types.StringNull(),
		BatchSize:              types.Int64Null(),
		BatchWindowSeconds:     types.Int64Null(),
		UseStaticIP:            types.BoolValue(false),
		IsActive:               types.BoolValue(true),
		SigningSecret:          types.StringUnknown(),
		HasSigningSecret:       types.BoolUnknown(),
		CreatedAt:              types.StringUnknown(),
		UpdatedAt:              types.StringUnknown(),
	}
}

func baseDestination(destType string) client.Destination {
	authType, timeout := "none", 30000
	return client.Destination{
		ID: "dst_1", OrganizationID: "org-1", Name: "d", Slug: "d", Method: "POST",
		AuthType: &authType, TimeoutMs: &timeout, Type: destType,
		Throttle: &client.Throttle{Mode: "off"}, IsActive: true,
		CreatedAt: "2026-11-01T00:00:00Z", UpdatedAt: "2026-11-01T00:00:00Z",
	}
}

func createDestination(t *testing.T, r *DestinationResource, plan DestinationResourceModel) (tfsdk.State, DestinationResourceModel) {
	t.Helper()
	resp := resource.CreateResponse{State: nullStateOf(t, r)}
	r.Create(context.Background(), resource.CreateRequest{Plan: planOf(t, r, &plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	return resp.State, modelOf[DestinationResourceModel](t, resp.State)
}

func readDestination(t *testing.T, r *DestinationResource, st tfsdk.State) DestinationResourceModel {
	t.Helper()
	resp := resource.ReadResponse{State: nullStateOf(t, r)}
	r.Read(context.Background(), resource.ReadRequest{State: st}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	return modelOf[DestinationResourceModel](t, resp.State)
}

func TestDestinationCreate_HTTPSecretFromCreateReply(t *testing.T) {
	dest := baseDestination("http")
	dest.URL = "https://example.com/hook"
	dest.HasSigningSecret = true
	reply := dest
	reply.SigningSecret = strp("whsec_created")
	f := &fakeDestinationAPI{t: t, createReply: reply, dest: dest, secret: "whsec_created"}
	r := newTestDestinationResource(t, f)

	plan := newDestinationPlan("http")
	plan.URL = types.StringValue("https://example.com/hook")
	st, got := createDestination(t, r, plan)

	if got.SigningSecret.ValueString() != "whsec_created" || !got.HasSigningSecret.ValueBool() {
		t.Errorf("signing_secret = %v, has_signing_secret = %v; want the create reply's secret", got.SigningSecret, got.HasSigningSecret)
	}
	if n := f.count("GET /destinations/dst_1/reveal-secret"); n != 0 {
		t.Errorf("create called reveal-secret %d times; the create reply already has the secret", n)
	}

	// Refresh reads the current secret back, so one rotated outside Terraform shows up.
	f.secret = "whsec_rotated"
	if got := readDestination(t, r, st); got.SigningSecret.ValueString() != "whsec_rotated" {
		t.Errorf("after refresh signing_secret = %v, want whsec_rotated", got.SigningSecret)
	}

	// A reveal the key isn't allowed (write scope) keeps the secret in state instead of dropping it,
	// and warns that it may be stale.
	f.revealStatus = http.StatusForbidden
	resp := resource.ReadResponse{State: nullStateOf(t, r)}
	r.Read(context.Background(), resource.ReadRequest{State: st}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	if got := modelOf[DestinationResourceModel](t, resp.State); got.SigningSecret.ValueString() != "whsec_created" {
		t.Errorf("after a refused reveal signing_secret = %v, want the prior whsec_created", got.SigningSecret)
	}
	warnings := resp.Diagnostics.Warnings()
	if len(warnings) != 1 || !strings.Contains(warnings[0].Detail(), "403") || !strings.Contains(warnings[0].Detail(), "stale") {
		t.Errorf("warnings = %v, want one naming the 403 and that the kept secret may be stale", warnings)
	}
}

// A bridge destination has no URL and no secret: url stays null (sent as ""), signing_secret is
// null, nothing calls reveal-secret, and the config keeps its configured text though the API
// answers it with the keys in another order.
func TestDestinationCreate_BridgeHasNoURLOrSecret(t *testing.T) {
	configured := `{"eventTypeSource":"fixed","eventType":"order.created","applicationId":"app_1"}`
	dest := baseDestination("bridge")
	dest.Config = json.RawMessage(`{"applicationId":"app_1","eventType":"order.created","eventTypeSource":"fixed"}`)
	reply := dest
	reply.Config = json.RawMessage(configured)
	f := &fakeDestinationAPI{t: t, createReply: reply, dest: dest}
	r := newTestDestinationResource(t, f)

	plan := newDestinationPlan("bridge")
	plan.Config = types.StringValue(configured)
	st, got := createDestination(t, r, plan)

	if v, ok := f.lastCreate["url"]; !ok || v != "" {
		t.Errorf("create body url = %v (present %v), want \"\"", v, ok)
	}
	if cfg, ok := f.lastCreate["config"].(map[string]any); !ok || cfg["eventType"] != "order.created" {
		t.Errorf("create body config = %v, want the configured object", f.lastCreate["config"])
	}
	if !got.URL.IsNull() || !got.SigningSecret.IsNull() || got.HasSigningSecret.ValueBool() {
		t.Errorf("url = %v, signing_secret = %v, has_signing_secret = %v; want null, null, false", got.URL, got.SigningSecret, got.HasSigningSecret)
	}
	if got.Config.ValueString() != configured {
		t.Errorf("config = %s, want the configured text", got.Config.ValueString())
	}

	refreshed := readDestination(t, r, st)
	if refreshed.Config.ValueString() != configured {
		t.Errorf("after refresh config = %s, want the configured text (the API only reordered keys)", refreshed.Config.ValueString())
	}
	if !refreshed.SigningSecret.IsNull() {
		t.Errorf("after refresh signing_secret = %v, want null", refreshed.SigningSecret)
	}
	if n := f.count("GET /destinations/dst_1/reveal-secret"); n != 0 {
		t.Errorf("reveal-secret called %d times for a bridge", n)
	}

	// A real change made outside Terraform shows up as drift.
	f.dest.Config = json.RawMessage(`{"applicationId":"app_1","eventType":"order.updated","eventTypeSource":"fixed"}`)
	if got := readDestination(t, r, st); !strings.Contains(got.Config.ValueString(), "order.updated") {
		t.Errorf("after an outside change config = %s, want the API's", got.Config.ValueString())
	}
}

// PATCH answers {success:true} with no destination, so Update must read it back; the old code
// mapped the empty reply and wrote blank values into state.
func TestDestinationUpdate_ReadsBackAfterPatch(t *testing.T) {
	dest := baseDestination("http")
	dest.URL = "https://example.com/hook"
	dest.HasSigningSecret = true
	reply := dest
	reply.SigningSecret = strp("whsec_created")
	f := &fakeDestinationAPI{t: t, createReply: reply, dest: dest, secret: "whsec_created"}
	r := newTestDestinationResource(t, f)

	st, applied := createDestination(t, r, func() DestinationResourceModel {
		p := newDestinationPlan("http")
		p.URL = types.StringValue("https://example.com/hook")
		return p
	}())

	plan := applied
	plan.Name = types.StringValue("renamed")
	f.dest.Name = "renamed"
	f.dest.UpdatedAt = "2026-11-02T00:00:00Z"
	f.calls = nil

	resp := resource.UpdateResponse{State: nullStateOf(t, r)}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, r, &plan), State: st}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	got := modelOf[DestinationResourceModel](t, resp.State)
	if got.ID.ValueString() != "dst_1" || got.Name.ValueString() != "renamed" || got.URL.ValueString() != "https://example.com/hook" {
		t.Errorf("after update id=%v name=%v url=%v", got.ID, got.Name, got.URL)
	}
	if got.UpdatedAt.ValueString() != "2026-11-02T00:00:00Z" {
		t.Errorf("updated_at = %v, want the re-read value", got.UpdatedAt)
	}
	if got.SigningSecret.ValueString() != "whsec_created" {
		t.Errorf("signing_secret = %v, want it kept", got.SigningSecret)
	}
	if f.count("GET /destinations/dst_1/reveal-secret") != 0 {
		t.Error("update called reveal-secret for an unchanged secret")
	}
	if f.lastPatch["name"] != "renamed" {
		t.Errorf("patch body = %v", f.lastPatch)
	}
	if _, sent := f.lastPatch["signingSecret"]; sent {
		t.Error("patch body carried signingSecret, which the API refuses")
	}
}

// State from 0.6.x has no has_signing_secret or signing_secret. Without a refresh
// (-refresh=false), UseStateForUnknown plans those nulls, and the applied state must keep them
// rather than take the API's values; the next refresh fills them in.
func TestDestinationUpdate_UnrefreshedUpgradeKeepsPlannedNulls(t *testing.T) {
	dest := baseDestination("http")
	dest.URL = "https://example.com/hook"
	dest.HasSigningSecret = true
	f := &fakeDestinationAPI{t: t, dest: dest, secret: "whsec_existing"}
	r := newTestDestinationResource(t, f)

	old := newDestinationPlan("http")
	old.ID, old.URL = types.StringValue("dst_1"), types.StringValue("https://example.com/hook")
	old.SigningSecret, old.HasSigningSecret = types.StringNull(), types.BoolNull()
	old.CreatedAt, old.UpdatedAt = types.StringValue(dest.CreatedAt), types.StringValue(dest.UpdatedAt)

	plan := old
	plan.Name = types.StringValue("renamed")
	f.dest.Name = "renamed"
	resp := resource.UpdateResponse{State: nullStateOf(t, r)}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, r, &plan), State: stateOf(t, r, &old)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	got := modelOf[DestinationResourceModel](t, resp.State)
	if !got.HasSigningSecret.IsNull() || !got.SigningSecret.IsNull() {
		t.Errorf("has_signing_secret = %v, signing_secret = %v; want the planned nulls", got.HasSigningSecret, got.SigningSecret)
	}

	// The next refresh fills both in.
	if got := readDestination(t, r, resp.State); !got.HasSigningSecret.ValueBool() || got.SigningSecret.ValueString() != "whsec_existing" {
		t.Errorf("after refresh has_signing_secret = %v, signing_secret = %v", got.HasSigningSecret, got.SigningSecret)
	}
}

func TestDestinationRead_ImportRevealsSecret(t *testing.T) {
	dest := baseDestination("http")
	dest.URL = "https://example.com/hook"
	dest.HasSigningSecret = true
	dest.Headers = map[string]string{"X-Team": "core"}
	f := &fakeDestinationAPI{t: t, dest: dest, secret: "whsec_existing"}
	r := newTestDestinationResource(t, f)

	nullMap := types.MapNull(types.StringType)
	st := stateOf(t, r, &DestinationResourceModel{ID: types.StringValue("dst_1"), Headers: nullMap, AuthConfig: nullMap})
	got := readDestination(t, r, st)
	if got.SigningSecret.ValueString() != "whsec_existing" || !got.HasSigningSecret.ValueBool() {
		t.Errorf("signing_secret = %v, has_signing_secret = %v after import", got.SigningSecret, got.HasSigningSecret)
	}
	if got.URL.ValueString() != "https://example.com/hook" || got.Type.ValueString() != "http" {
		t.Errorf("url = %v, type = %v", got.URL, got.Type)
	}
	if !got.Config.IsNull() || !got.AuthConfig.IsNull() {
		t.Errorf("config = %v, auth_config = %v; want null", got.Config, got.AuthConfig)
	}
}

func TestBuildUpdateDestinationRequest(t *testing.T) {
	ctx := context.Background()
	state := newDestinationPlan("sqs")
	state.ID = types.StringValue("dst_1")
	state.URL = types.StringValue("https://sqs.example.com/queue")
	state.Headers = types.MapValueMust(types.StringType, map[string]attr.Value{"X-A": types.StringValue("1")})
	state.Config = types.StringValue(`{"queueUrl":"q","accessKeyId":"AKIA","secretAccessKey":"s"}`)
	state.BatchSize = types.Int64Value(10)
	state.BatchWindowSeconds = types.Int64Value(5)
	state.SigningSecret = types.StringNull()
	state.HasSigningSecret = types.BoolValue(false)

	t.Run("removed optionals are cleared", func(t *testing.T) {
		plan := state
		plan.URL, plan.Headers, plan.Config = types.StringNull(), types.MapNull(types.StringType), types.StringNull()
		plan.BatchSize, plan.BatchWindowSeconds = types.Int64Null(), types.Int64Null()
		req, diags := buildUpdateDestinationRequest(ctx, &plan, &state)
		if diags.HasError() {
			t.Fatal(diags)
		}
		body := toJSONMap(t, req)
		for key, want := range map[string]any{"url": "", "headers": nil, "config": nil, "batchSize": float64(1), "batchWindowSeconds": float64(0)} {
			got, ok := body[key]
			if !ok || got != want {
				t.Errorf("%s = %v (present %v), want %v", key, got, ok, want)
			}
		}
	})

	t.Run("unchanged config is not re-sent", func(t *testing.T) {
		plan := state
		plan.Name = types.StringValue("renamed")
		req, _ := buildUpdateDestinationRequest(ctx, &plan, &state)
		body := toJSONMap(t, req)
		if _, ok := body["config"]; ok {
			t.Error("config sent though unchanged")
		}
		if h, ok := body["headers"].(map[string]any); !ok || h["X-A"] != "1" {
			t.Errorf("headers = %v, want the planned map", body["headers"])
		}
	})

	// A rename of a bridge must not carry anything bridgeUnsupportedFields (api destinations.ts)
	// refuses: a truthy url, non-empty headers, an authType other than none, useStaticIp true. Nor
	// the unchanged config, which would re-run the bridge checks (and need events:send).
	t.Run("bridge rename", func(t *testing.T) {
		bridge := newDestinationPlan("bridge")
		bridge.ID = types.StringValue("dst_1")
		bridge.Config = types.StringValue(`{"applicationId":"app_1","eventTypeSource":"payload"}`)
		bridge.SigningSecret, bridge.HasSigningSecret = types.StringNull(), types.BoolValue(false)
		plan := bridge
		plan.Name = types.StringValue("renamed")
		req, diags := buildUpdateDestinationRequest(ctx, &plan, &bridge)
		if diags.HasError() {
			t.Fatal(diags)
		}
		body := toJSONMap(t, req)
		if v, ok := body["url"]; ok && v != "" {
			t.Errorf("url = %v, want absent or \"\"", v)
		}
		if h, ok := body["headers"]; ok && h != nil {
			if m, isMap := h.(map[string]any); !isMap || len(m) > 0 {
				t.Errorf("headers = %v, want absent, null or empty", h)
			}
		}
		if v, ok := body["authType"]; ok && v != "none" {
			t.Errorf("authType = %v, want absent or none", v)
		}
		if v, ok := body["useStaticIp"]; ok && v != false {
			t.Errorf("useStaticIp = %v, want absent or false", v)
		}
		for _, key := range []string{"config", "signingSecret"} {
			if v, ok := body[key]; ok {
				t.Errorf("%s = %v sent for a rename", key, v)
			}
		}
		if body["name"] != "renamed" || body["type"] != "bridge" {
			t.Errorf("name = %v, type = %v", body["name"], body["type"])
		}
	})

	t.Run("changed config is sent", func(t *testing.T) {
		plan := state
		plan.Config = types.StringValue(`{"queueUrl":"q2"}`)
		req, _ := buildUpdateDestinationRequest(ctx, &plan, &state)
		if cfg, ok := toJSONMap(t, req)["config"].(map[string]any); !ok || cfg["queueUrl"] != "q2" {
			t.Errorf("config = %v", cfg)
		}
	})
}

func TestDestinationEmptyHeadersKept(t *testing.T) {
	dest := baseDestination("http")
	dest.URL = "https://example.com/hook"
	f := &fakeDestinationAPI{t: t, createReply: dest, dest: dest}
	r := newTestDestinationResource(t, f)

	plan := newDestinationPlan("http")
	plan.URL = types.StringValue("https://example.com/hook")
	plan.Headers = types.MapValueMust(types.StringType, map[string]attr.Value{})
	st, got := createDestination(t, r, plan)
	if got.Headers.IsNull() || len(got.Headers.Elements()) != 0 {
		t.Errorf("headers = %v after create, want {}", got.Headers)
	}
	if got := readDestination(t, r, st); got.Headers.IsNull() {
		t.Error("headers read back as null, want {}")
	}
}

// jsonToState keeps the configured text when the API's value means the same thing.
func TestJSONToState(t *testing.T) {
	prior := types.StringValue(`{"region":"us-east-1","queueUrl":"https://q","secretAccessKey":"abcd1234","accessKeyId":"AKIAXYZ"}`)
	cases := []struct {
		name     string
		prior    types.String
		api      string
		redacted bool
		want     string // "" means null; "prior" means the prior value
	}{
		{"keys reordered", prior, `{"accessKeyId":"AKIAXYZ","queueUrl":"https://q","region":"us-east-1","secretAccessKey":"abcd1234"}`, false, "prior"},
		{"whitespace and number format", types.StringValue(`{ "a": 1, "b": [1, 2] }`), `{"b":[1.0,2],"a":1}`, false, "prior"},
		{"stored JSON text (http config)", types.StringValue(`{"a":1}`), `"{\"a\": 1}"`, false, "prior"},
		{"credentials masked", prior, `{"accessKeyId":"••••9f2a","queueUrl":"https://q","region":"us-east-1","secretAccessKey":"••••"}`, true, "prior"},
		{"masked only matters when redacted", prior, `{"accessKeyId":"••••9f2a","queueUrl":"https://q","region":"us-east-1","secretAccessKey":"••••"}`, false,
			`{"accessKeyId":"••••9f2a","queueUrl":"https://q","region":"us-east-1","secretAccessKey":"••••"}`},
		{"drift shows", prior, `{"accessKeyId":"••••9f2a","queueUrl":"https://other","region":"us-east-1","secretAccessKey":"••••"}`, true,
			`{"accessKeyId":"••••9f2a","queueUrl":"https://other","region":"us-east-1","secretAccessKey":"••••"}`},
		{"key added", types.StringValue(`{"a":1}`), `{"a":1,"b":2}`, false, `{"a":1,"b":2}`},
		{"API null", prior, `null`, false, ""},
		{"API absent", prior, ``, false, ""},
		{"stored JSON text null", prior, `"null"`, false, ""},
		{"import takes the API's", types.StringNull(), `{"b": 2, "a": 1}`, false, `{"b":2,"a":1}`},
		{"configured [] stored as nothing", types.StringValue(`[]`), `null`, false, "prior"},
		{"configured {} stored as nothing", types.StringValue(`{ }`), ``, false, "prior"},
		// State written by 0.6.x holds a route's filter_conditions as the API sent it: its stored
		// JSON text (jsonb's key order and spacing) in a JSON string. It is stored the way
		// jsonencode writes the value, so a jsonencode() configuration shows no diff.
		{"0.6.x prior: JSON string", types.StringValue(`"[{\"field\": \"type\", \"value\": \"order.created\", \"operator\": \"equals\"}]"`),
			`"[{\"field\": \"type\", \"value\": \"order.created\", \"operator\": \"equals\"}]"`, false,
			`[{"field":"type","operator":"equals","value":"order.created"}]`},
		{"0.6.x prior: drift still shows", types.StringValue(`"[{\"field\": \"type\", \"operator\": \"exists\"}]"`),
			`"[{\"field\": \"kind\", \"operator\": \"exists\"}]"`, false, `[{"field":"kind","operator":"exists"}]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := jsonToState(c.prior, json.RawMessage(c.api), c.redacted)
			switch c.want {
			case "":
				if !got.IsNull() {
					t.Errorf("got %v, want null", got)
				}
			case "prior":
				if !got.Equal(c.prior) {
					t.Errorf("got %v, want the prior %v", got, c.prior)
				}
			default:
				if got.ValueString() != c.want {
					t.Errorf("got %v, want %s", got, c.want)
				}
			}
		})
	}
}

func TestDestinationValidateConfig(t *testing.T) {
	r := &DestinationResource{}
	s := schemaOf(t, r)
	cases := []struct {
		name      string
		destType  tftypes.Value
		url       tftypes.Value
		wantError bool
	}{
		{"http with url", tftypes.NewValue(tftypes.String, "http"), tftypes.NewValue(tftypes.String, "https://x"), false},
		{"http without url", tftypes.NewValue(tftypes.String, "http"), tftypes.NewValue(tftypes.String, nil), true},
		{"default type without url", tftypes.NewValue(tftypes.String, nil), tftypes.NewValue(tftypes.String, nil), true},
		{"queue without url", tftypes.NewValue(tftypes.String, "sqs"), tftypes.NewValue(tftypes.String, nil), false},
		{"bridge without url", tftypes.NewValue(tftypes.String, "bridge"), tftypes.NewValue(tftypes.String, nil), false},
		{"bridge with url", tftypes.NewValue(tftypes.String, "bridge"), tftypes.NewValue(tftypes.String, "https://x"), true},
		{"url not yet known", tftypes.NewValue(tftypes.String, "http"), tftypes.NewValue(tftypes.String, tftypes.UnknownValue), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			objType := s.Type().TerraformType(context.Background()).(tftypes.Object)
			vals := map[string]tftypes.Value{}
			for name, typ := range objType.AttributeTypes {
				vals[name] = tftypes.NewValue(typ, nil)
			}
			vals["type"], vals["url"] = c.destType, c.url
			cfg := tfsdk.Config{Schema: s, Raw: tftypes.NewValue(objType, vals)}
			var resp resource.ValidateConfigResponse
			r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{Config: cfg}, &resp)
			if resp.Diagnostics.HasError() != c.wantError {
				t.Errorf("error = %v, want %v (%v)", resp.Diagnostics.HasError(), c.wantError, resp.Diagnostics)
			}
		})
	}
}

// type accepts exactly the API's DESTINATION_TYPES.
func TestDestinationTypeValidator(t *testing.T) {
	attrSchema := schemaOf(t, &DestinationResource{}).Attributes["type"].(schema.StringAttribute)
	check := func(v string) bool {
		var resp validator.StringResponse
		for _, val := range attrSchema.Validators {
			val.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("type"), ConfigValue: types.StringValue(v)}, &resp)
		}
		return !resp.Diagnostics.HasError()
	}
	for _, ok := range []string{"http", "sqs", "eventbridge", "servicebus", "pubsub", "oci_queue", "s3", "r2", "gcs", "azure_blob", "bridge"} {
		if !check(ok) {
			t.Errorf("%s rejected", ok)
		}
	}
	for _, bad := range []string{"websocket", "tunnel", "sns", "gcp_pubsub", "kafka", "rabbitmq"} {
		if check(bad) {
			t.Errorf("%s accepted", bad)
		}
	}
}
