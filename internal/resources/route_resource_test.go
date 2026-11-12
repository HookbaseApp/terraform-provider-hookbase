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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

// fakeRouteAPI answers the route endpoints in the API's shapes: GET returns filterConditions as
// its stored JSON text, PATCH /routes/:id and PATCH /routes/:id/circuit-config answer
// {success:true}. circuitStatus, when set, refuses circuit-config the way the API does for an
// organization without the circuit_breaker feature.
type fakeRouteAPI struct {
	t             *testing.T
	route         map[string]any // what GET answers under "route"
	circuitStatus int
	calls         []string
	lastCreate    map[string]any
	lastPatch     map[string]any
	lastCircuit   map[string]any
}

func (f *fakeRouteAPI) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/api/organizations/org-1")
		f.calls = append(f.calls, r.Method+" "+p)
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPost && p == "/routes":
			f.lastCreate = map[string]any{}
			_ = json.Unmarshal(body, &f.lastCreate)
			writeTestJSON(w, http.StatusCreated, map[string]any{"route": map[string]any{"id": "rt_1"}})
		case r.Method == http.MethodGet && p == "/routes/rt_1":
			writeTestJSON(w, http.StatusOK, map[string]any{"route": f.route})
		case r.Method == http.MethodPatch && p == "/routes/rt_1":
			f.lastPatch = map[string]any{}
			_ = json.Unmarshal(body, &f.lastPatch)
			writeTestJSON(w, http.StatusOK, map[string]any{"success": true})
		case r.Method == http.MethodPatch && p == "/routes/rt_1/circuit-config" && f.circuitStatus != 0:
			writeTestJSON(w, f.circuitStatus, map[string]any{"error": "Circuit breaker configuration requires a plan with the circuit_breaker feature"})
		case r.Method == http.MethodPatch && p == "/routes/rt_1/circuit-config":
			f.lastCircuit = map[string]any{}
			_ = json.Unmarshal(body, &f.lastCircuit)
			writeTestJSON(w, http.StatusOK, map[string]any{"success": true})
		default:
			f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			writeTestJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		}
	}))
}

func (f *fakeRouteAPI) called(call string) bool {
	for _, c := range f.calls {
		if c == call {
			return true
		}
	}
	return false
}

// baseRouteJSON is a GET reply for a route created with no optional settings.
func baseRouteJSON() map[string]any {
	return map[string]any{
		"id": "rt_1", "organizationId": "org-1", "name": "r", "sourceId": "src_1", "destinationId": "dst_1",
		"filterId": nil, "filterConditions": nil, "filterLogic": nil, "transformId": nil, "schemaId": nil,
		"priority": 0, "isActive": true, "notifyOnFailure": false, "notifyOnSuccess": false,
		"notifyOnRecovery": false, "notifyEmails": nil, "failureThreshold": 3,
		"failoverDestinationIds": []string{}, "failoverAfterAttempts": 5, "expectedResponse": nil,
		"circuitCooldownSeconds": nil, "circuitProbeSuccessThreshold": nil, "circuitFailureThreshold": nil,
		"holdOnCircuitOpen": true,
		"createdAt":         "2026-11-01T00:00:00Z", "updatedAt": "2026-11-01T00:00:00Z",
	}
}

// newRoutePlan is a create plan with every optional attribute unset.
func newRoutePlan() RouteResourceModel {
	return RouteResourceModel{
		ID:                           types.StringUnknown(),
		Name:                         types.StringValue("r"),
		SourceID:                     types.StringValue("src_1"),
		DestinationID:                types.StringValue("dst_1"),
		FilterID:                     types.StringNull(),
		FilterConditions:             types.StringNull(),
		FilterLogic:                  types.StringValue("AND"),
		TransformID:                  types.StringNull(),
		SchemaID:                     types.StringNull(),
		Priority:                     types.Int64Value(0),
		IsActive:                     types.BoolValue(true),
		NotifyOnFailure:              types.BoolValue(false),
		NotifyOnSuccess:              types.BoolValue(false),
		NotifyOnRecovery:             types.BoolValue(false),
		NotifyEmails:                 types.StringNull(),
		FailureThreshold:             types.Int64Value(3),
		FailoverDestinationIds:       types.ListNull(types.StringType),
		FailoverAfterAttempts:        types.Int64Value(5),
		ExpectedResponse:             types.StringNull(),
		CircuitCooldownSeconds:       types.Int64Unknown(),
		CircuitProbeSuccessThreshold: types.Int64Unknown(),
		CircuitFailureThreshold:      types.Int64Unknown(),
		HoldOnCircuitOpen:            types.BoolUnknown(),
		CreatedAt:                    types.StringUnknown(),
		UpdatedAt:                    types.StringUnknown(),
	}
}

// createRoute applies a create plan. Its unknown values are the attributes left out of
// configuration.
func createRoute(t *testing.T, r *RouteResource, plan RouteResourceModel) (tfsdk.State, RouteResourceModel) {
	t.Helper()
	resp := resource.CreateResponse{State: nullStateOf(t, r)}
	r.Create(context.Background(), resource.CreateRequest{Plan: planOf(t, r, &plan), Config: configOf(t, r, &plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	return resp.State, modelOf[RouteResourceModel](t, resp.State)
}

// A configured hold_on_circuit_open is sent on create, and inline filter conditions, which the API
// answers as their stored JSON text with the keys sorted, keep their configured text.
func TestRouteCreate_HoldAndFilterConditions(t *testing.T) {
	configured := `[{"field":"type","operator":"equals","value":"order.created"}]`
	route := baseRouteJSON()
	route["filterConditions"] = `[{"field": "type", "value": "order.created", "operator": "equals"}]`
	route["filterLogic"] = "AND"
	route["holdOnCircuitOpen"] = false
	f := &fakeRouteAPI{t: t, route: route}
	r := &RouteResource{client: testClient(t, f.server())}

	plan := newRoutePlan()
	plan.FilterConditions = types.StringValue(configured)
	plan.HoldOnCircuitOpen = types.BoolValue(false)
	plan.FailoverDestinationIds = types.ListValueMust(types.StringType, []attr.Value{})
	st, got := createRoute(t, r, plan)

	if v, ok := f.lastCreate["holdOnCircuitOpen"]; !ok || v != false {
		t.Errorf("create body holdOnCircuitOpen = %v (present %v), want false", v, ok)
	}
	if got.HoldOnCircuitOpen.ValueBool() {
		t.Error("hold_on_circuit_open = true, want false")
	}
	if got.FilterConditions.ValueString() != configured {
		t.Errorf("filter_conditions = %s, want the configured text", got.FilterConditions.ValueString())
	}
	if got.FailoverDestinationIds.IsNull() {
		t.Error("failover_destination_ids = [] read back as null")
	}
	if !got.CircuitCooldownSeconds.IsNull() {
		t.Errorf("circuit_cooldown_seconds = %v, want null", got.CircuitCooldownSeconds)
	}

	resp := resource.ReadResponse{State: nullStateOf(t, r)}
	r.Read(context.Background(), resource.ReadRequest{State: st}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	if refreshed := modelOf[RouteResourceModel](t, resp.State); refreshed.FilterConditions.ValueString() != configured {
		t.Errorf("after refresh filter_conditions = %s", refreshed.FilterConditions.ValueString())
	}

	// Unset, nothing is sent and the API's default is read back.
	f.route = baseRouteJSON()
	if _, got := createRoute(t, r, newRoutePlan()); !got.HoldOnCircuitOpen.ValueBool() {
		t.Error("hold_on_circuit_open = false, want the API default true")
	}
	if v, ok := f.lastCreate["holdOnCircuitOpen"]; ok {
		t.Errorf("create body carried holdOnCircuitOpen = %v though it is not configured", v)
	}
}

// Provider 0.6.x stored filter_conditions exactly as GET returns it, a JSON string holding the
// stored JSON text. A refresh stores the same value the way jsonencode writes it, so a
// jsonencode() configuration shows no diff after upgrading.
func TestRouteRead_UnwrapsFilterConditionsFrom06State(t *testing.T) {
	stored := `[{"field": "type", "value": "order.created", "operator": "equals"}]`
	route := baseRouteJSON()
	route["filterConditions"], route["filterLogic"] = stored, "AND"
	f := &fakeRouteAPI{t: t, route: route}
	r := &RouteResource{client: testClient(t, f.server())}

	wrapped, _ := json.Marshal(stored)
	old := newRoutePlan()
	old.ID, old.FilterConditions = types.StringValue("rt_1"), types.StringValue(string(wrapped))
	old.CircuitCooldownSeconds, old.CircuitProbeSuccessThreshold, old.CircuitFailureThreshold = types.Int64Null(), types.Int64Null(), types.Int64Null()
	old.HoldOnCircuitOpen = types.BoolNull()
	old.CreatedAt, old.UpdatedAt = types.StringNull(), types.StringNull()

	resp := resource.ReadResponse{State: nullStateOf(t, r)}
	r.Read(context.Background(), resource.ReadRequest{State: stateOf(t, r, &old)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	got := modelOf[RouteResourceModel](t, resp.State)
	if want := `[{"field":"type","operator":"equals","value":"order.created"}]`; got.FilterConditions.ValueString() != want {
		t.Errorf("filter_conditions = %s, want %s (jsonencode's form)", got.FilterConditions.ValueString(), want)
	}
}

// An API that predates hold_on_circuit_open omits it; the route behaved as true.
func TestRouteRead_HoldDefaultsToTrue(t *testing.T) {
	route := baseRouteJSON()
	delete(route, "holdOnCircuitOpen")
	f := &fakeRouteAPI{t: t, route: route}
	r := &RouteResource{client: testClient(t, f.server())}
	if _, got := createRoute(t, r, newRoutePlan()); !got.HoldOnCircuitOpen.ValueBool() {
		t.Error("hold_on_circuit_open = false, want true")
	}
}

// PATCH answers {success:true} with no route, so Update reads it back; circuit settings go
// through circuit-config, which PATCH /routes/:id ignores.
func TestRouteUpdate_ReadsBackAndSendsCircuitConfig(t *testing.T) {
	f := &fakeRouteAPI{t: t, route: baseRouteJSON()}
	r := &RouteResource{client: testClient(t, f.server())}
	st, applied := createRoute(t, r, newRoutePlan())

	update := func(plan, config RouteResourceModel) RouteResourceModel {
		t.Helper()
		f.lastPatch, f.lastCircuit = nil, nil
		resp := resource.UpdateResponse{State: nullStateOf(t, r)}
		r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, r, &plan), Config: configOf(t, r, &config), State: st}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Update: %v", resp.Diagnostics)
		}
		return modelOf[RouteResourceModel](t, resp.State)
	}

	plan := applied
	plan.Name = types.StringValue("renamed")
	plan.HoldOnCircuitOpen = types.BoolValue(false)
	f.route["name"], f.route["holdOnCircuitOpen"] = "renamed", false
	got := update(plan, plan)
	if got.ID.ValueString() != "rt_1" || got.Name.ValueString() != "renamed" || got.SourceID.ValueString() != "src_1" {
		t.Errorf("after update id=%v name=%v source_id=%v", got.ID, got.Name, got.SourceID)
	}
	if f.lastPatch["holdOnCircuitOpen"] != false || f.lastPatch["name"] != "renamed" {
		t.Errorf("patch body = %v", f.lastPatch)
	}
	for _, key := range []string{"sourceId", "destinationId", "transformId", "failoverDestinationIds", "circuitCooldownSeconds"} {
		if _, ok := f.lastPatch[key]; ok {
			t.Errorf("patch body carried unchanged or unsupported %s", key)
		}
	}
	if f.called("PATCH /routes/rt_1/circuit-config") {
		t.Error("circuit-config called with no circuit change")
	}

	plan = applied
	plan.CircuitCooldownSeconds = types.Int64Value(120)
	config := plan
	config.HoldOnCircuitOpen = types.BoolNull()
	f.route["circuitCooldownSeconds"] = 120
	got = update(plan, config)
	if f.lastCircuit == nil || f.lastCircuit["circuitCooldownSeconds"] != float64(120) || len(f.lastCircuit) != 1 {
		t.Errorf("circuit-config body = %v, want only circuitCooldownSeconds 120", f.lastCircuit)
	}
	if got.CircuitCooldownSeconds.ValueInt64() != 120 {
		t.Errorf("circuit_cooldown_seconds = %v", got.CircuitCooldownSeconds)
	}
	if !f.called("PATCH /routes/rt_1") || f.calls[len(f.calls)-3] != "PATCH /routes/rt_1/circuit-config" {
		t.Errorf("calls = %v, want circuit-config before the route PATCH", f.calls)
	}
}

// circuit-config needs the circuit_breaker feature. When the API refuses it, the apply fails
// before the route PATCH, so no other change is applied without it.
func TestRouteUpdate_CircuitConfigRefusedBeforePatch(t *testing.T) {
	f := &fakeRouteAPI{t: t, route: baseRouteJSON(), circuitStatus: http.StatusForbidden}
	r := &RouteResource{client: testClient(t, f.server())}
	st, applied := createRoute(t, r, newRoutePlan())

	plan := applied
	plan.Name = types.StringValue("renamed")
	plan.CircuitFailureThreshold = types.Int64Value(10)
	config := plan
	config.HoldOnCircuitOpen = types.BoolNull()
	f.calls = nil
	resp := resource.UpdateResponse{State: nullStateOf(t, r)}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, r, &plan), Config: configOf(t, r, &config), State: st}, &resp)

	if !resp.Diagnostics.HasError() || resp.Diagnostics.Errors()[0].Summary() != "Error updating route circuit breaker settings" {
		t.Fatalf("diagnostics = %v, want the circuit-config refusal", resp.Diagnostics)
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "403") {
		t.Errorf("error detail = %q, want the API's 403", resp.Diagnostics.Errors()[0].Detail())
	}
	if f.called("PATCH /routes/rt_1") {
		t.Errorf("calls = %v: the route PATCH was sent after circuit-config was refused", f.calls)
	}
}

// hold_on_circuit_open is sent only when configured. Unset, the plan keeps the value in state, so a
// route switched off in the dashboard shows no diff and is not switched back on.
func TestRouteHoldOnCircuitOpen_UnsetKeepsServerValue(t *testing.T) {
	ctx := context.Background()
	f := &fakeRouteAPI{t: t, route: baseRouteJSON()}
	r := &RouteResource{client: testClient(t, f.server())}
	st, _ := createRoute(t, r, newRoutePlan())

	// Turned off outside Terraform; the refresh picks it up.
	f.route["holdOnCircuitOpen"] = false
	readResp := resource.ReadResponse{State: nullStateOf(t, r)}
	r.Read(ctx, resource.ReadRequest{State: st}, &readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", readResp.Diagnostics)
	}
	st = readResp.State
	refreshed := modelOf[RouteResourceModel](t, st)
	if refreshed.HoldOnCircuitOpen.ValueBool() {
		t.Fatal("hold_on_circuit_open = true after refresh, want the API's false")
	}

	// With no default, the attribute's plan modifiers keep the state's value for an unset
	// attribute that Terraform marks unknown because something else changed.
	holdAttr := schemaOf(t, r).Attributes["hold_on_circuit_open"].(schema.BoolAttribute)
	if holdAttr.Default != nil {
		t.Fatal("hold_on_circuit_open has a default, which would plan true over the API's value")
	}
	modReq := planmodifier.BoolRequest{
		Path: path.Root("hold_on_circuit_open"), State: st, StateValue: types.BoolValue(false),
		PlanValue: types.BoolUnknown(), ConfigValue: types.BoolNull(),
	}
	modResp := &planmodifier.BoolResponse{PlanValue: modReq.PlanValue}
	for _, m := range holdAttr.PlanModifiers {
		m.PlanModifyBool(ctx, modReq, modResp)
	}
	if !modResp.PlanValue.Equal(types.BoolValue(false)) {
		t.Fatalf("planned hold_on_circuit_open = %v, want the state's false (no diff)", modResp.PlanValue)
	}

	update := func(plan, config RouteResourceModel, prior tfsdk.State) RouteResourceModel {
		t.Helper()
		f.lastPatch = nil
		resp := resource.UpdateResponse{State: nullStateOf(t, r)}
		r.Update(ctx, resource.UpdateRequest{Plan: planOf(t, r, &plan), Config: configOf(t, r, &config), State: prior}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Update: %v", resp.Diagnostics)
		}
		return modelOf[RouteResourceModel](t, resp.State)
	}

	plan := refreshed
	plan.Name = types.StringValue("renamed")
	plan.HoldOnCircuitOpen = modResp.PlanValue
	config := plan
	config.HoldOnCircuitOpen = types.BoolNull()
	got := update(plan, config, st)
	if v, ok := f.lastPatch["holdOnCircuitOpen"]; ok {
		t.Errorf("patch body carried holdOnCircuitOpen = %v though it is not configured", v)
	}
	if got.HoldOnCircuitOpen.ValueBool() {
		t.Error("hold_on_circuit_open = true after update, want false kept")
	}

	// Set to false in configuration, it is sent.
	config.HoldOnCircuitOpen = types.BoolValue(false)
	update(plan, config, st)
	if v, ok := f.lastPatch["holdOnCircuitOpen"]; !ok || v != false {
		t.Errorf("patch body holdOnCircuitOpen = %v (present %v), want false", v, ok)
	}

	// State from 0.6.x has no hold_on_circuit_open. Without a refresh (-refresh=false) the plan
	// holds that null, and the applied state must too; the next refresh fills it in.
	f.route["holdOnCircuitOpen"] = true
	old := refreshed
	old.HoldOnCircuitOpen = types.BoolNull()
	plan = old
	plan.Name = types.StringValue("renamed again")
	config = plan
	if got := update(plan, config, stateOf(t, r, &old)); !got.HoldOnCircuitOpen.IsNull() {
		t.Errorf("hold_on_circuit_open = %v, want the planned null", got.HoldOnCircuitOpen)
	}
}

func TestBuildUpdateRouteRequest_ClearsRemovedOptionals(t *testing.T) {
	ctx := context.Background()
	state := newRoutePlan()
	state.ID = types.StringValue("rt_1")
	state.FilterConditions = types.StringValue(`[{"field":"a","operator":"exists"}]`)
	state.TransformID = types.StringValue("tr_1")
	state.SchemaID = types.StringValue("sch_1")
	state.NotifyEmails = types.StringValue("ops@example.com")
	state.ExpectedResponse = types.StringValue(`{"status":200}`)
	state.FailoverDestinationIds = types.ListValueMust(types.StringType, []attr.Value{types.StringValue("dst_2")})

	plan := state
	plan.FilterConditions, plan.TransformID, plan.SchemaID = types.StringNull(), types.StringNull(), types.StringNull()
	plan.NotifyEmails, plan.ExpectedResponse = types.StringNull(), types.StringNull()
	plan.FailoverDestinationIds = types.ListNull(types.StringType)

	req, diags := buildUpdateRouteRequest(ctx, &plan, &state)
	if diags.HasError() {
		t.Fatal(diags)
	}
	body := toJSONMap(t, req)
	for key, want := range map[string]string{
		"filterConditions": "[]", "transformId": `""`, "schemaId": `""`, "notifyEmails": `""`,
		"expectedResponse": "null", "failoverDestinationIds": "[]",
	} {
		got, ok := body[key]
		b, _ := json.Marshal(got)
		if !ok || string(b) != want {
			t.Errorf("%s = %s (present %v), want %s", key, b, ok, want)
		}
	}
	// Update adds hold_on_circuit_open, and only when it is configured.
	if v, ok := body["holdOnCircuitOpen"]; ok {
		t.Errorf("holdOnCircuitOpen = %v, want it left to Update", v)
	}

	// Changing only filter_logic re-sends the conditions, since the API applies the logic only
	// together with them.
	plan = state
	plan.FilterLogic = types.StringValue("OR")
	req, _ = buildUpdateRouteRequest(ctx, &plan, &state)
	body = toJSONMap(t, req)
	if body["filterLogic"] != "OR" || body["filterConditions"] == nil {
		t.Errorf("filterLogic = %v, filterConditions = %v", body["filterLogic"], body["filterConditions"])
	}
}

// Webhook endpoints send a configured hold_on_circuit_open on create and update and read it back;
// unset, it is not sent and the endpoint's value is kept.
func TestWebhookEndpointHoldOnCircuitOpen(t *testing.T) {
	ep := client.WebhookEndpoint{
		ID: "ep_1", ApplicationID: "app_1", URL: "https://example.com/hook", TimeoutSeconds: 30,
		CircuitFailureThreshold: 5, CircuitSuccessThreshold: 2, CircuitCooldownSeconds: 60,
		SecretVersion: 1, CreatedAt: "2026-11-01T00:00:00Z", UpdatedAt: "2026-11-01T00:00:00Z",
	}
	hold := false
	ep.HoldOnCircuitOpen = &hold
	var lastCreate, lastPatch map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/api/organizations/org-1")
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPost && p == "/webhook-endpoints":
			lastCreate = map[string]any{}
			_ = json.Unmarshal(body, &lastCreate)
			reply := ep
			reply.Secret = strp("whsec_x")
			writeTestJSON(w, http.StatusCreated, map[string]any{"data": reply})
		case r.Method == http.MethodGet && p == "/webhook-endpoints/ep_1":
			writeTestJSON(w, http.StatusOK, map[string]any{"data": ep})
		case r.Method == http.MethodPatch && p == "/webhook-endpoints/ep_1":
			lastPatch = map[string]any{}
			_ = json.Unmarshal(body, &lastPatch)
			writeTestJSON(w, http.StatusOK, map[string]any{"data": ep})
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			writeTestJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		}
	}))
	r := &WebhookEndpointResource{client: testClient(t, srv)}

	plan := WebhookEndpointResourceModel{
		ID: types.StringUnknown(), ApplicationID: types.StringValue("app_1"), URL: types.StringValue("https://example.com/hook"),
		Description: types.StringNull(), Headers: types.StringNull(), TimeoutSeconds: types.Int64Value(30),
		RateLimitPerSecond: types.Int64Value(0), SuccessStatusCodes: types.StringNull(), BackoffType: types.StringValue("exponential"),
		RetryDelays: types.StringNull(), UseStaticIP: types.BoolValue(false), CircuitFailureThreshold: types.Int64Value(5),
		CircuitSuccessThreshold: types.Int64Value(2), CircuitCooldownSeconds: types.Int64Value(60),
		HoldOnCircuitOpen: types.BoolValue(false), IsDisabled: types.BoolValue(false), DisabledReason: types.StringNull(),
		Secret: types.StringUnknown(), SecretVersion: types.Int64Unknown(), CreatedAt: types.StringUnknown(), UpdatedAt: types.StringUnknown(),
	}
	createResp := resource.CreateResponse{State: nullStateOf(t, r)}
	r.Create(context.Background(), resource.CreateRequest{Plan: planOf(t, r, &plan), Config: configOf(t, r, &plan)}, &createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", createResp.Diagnostics)
	}
	if v, ok := lastCreate["holdOnCircuitOpen"]; !ok || v != false {
		t.Errorf("create body holdOnCircuitOpen = %v (present %v), want false", v, ok)
	}
	applied := modelOf[WebhookEndpointResourceModel](t, createResp.State)
	if applied.HoldOnCircuitOpen.ValueBool() {
		t.Error("hold_on_circuit_open = true after create, want false")
	}

	update := func(plan, config WebhookEndpointResourceModel) WebhookEndpointResourceModel {
		t.Helper()
		lastPatch = nil
		resp := resource.UpdateResponse{State: nullStateOf(t, r)}
		r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, r, &plan), Config: configOf(t, r, &config), State: createResp.State}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Update: %v", resp.Diagnostics)
		}
		return modelOf[WebhookEndpointResourceModel](t, resp.State)
	}

	plan = applied
	plan.HoldOnCircuitOpen = types.BoolValue(true)
	hold = true
	if got := update(plan, plan); !got.HoldOnCircuitOpen.ValueBool() {
		t.Error("hold_on_circuit_open = false after update, want true")
	}
	if lastPatch["holdOnCircuitOpen"] != true {
		t.Errorf("patch body holdOnCircuitOpen = %v, want true", lastPatch["holdOnCircuitOpen"])
	}

	// Unset in configuration, the plan holds the state's false (UseStateForUnknown), and nothing
	// is sent that would switch it back on.
	hold = false
	ep.Description = strp("renamed")
	plan = applied
	plan.Description = types.StringValue("renamed")
	config := plan
	config.HoldOnCircuitOpen = types.BoolNull()
	if got := update(plan, config); got.HoldOnCircuitOpen.ValueBool() {
		t.Error("hold_on_circuit_open = true after update, want false kept")
	}
	if v, ok := lastPatch["holdOnCircuitOpen"]; ok {
		t.Errorf("patch body carried holdOnCircuitOpen = %v though it is not configured", v)
	}
}

// updateRouteFrom applies plan over prior with the given configuration and returns the new state.
func updateRouteFrom(t *testing.T, r *RouteResource, prior tfsdk.State, plan, config RouteResourceModel) (tfsdk.State, RouteResourceModel) {
	t.Helper()
	resp := resource.UpdateResponse{State: nullStateOf(t, r)}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, r, &plan), Config: configOf(t, r, &config), State: prior}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", resp.Diagnostics)
	}
	return resp.State, modelOf[RouteResourceModel](t, resp.State)
}

func readRoute(t *testing.T, r *RouteResource, prior tfsdk.State) RouteResourceModel {
	t.Helper()
	resp := resource.ReadResponse{State: nullStateOf(t, r)}
	r.Read(context.Background(), resource.ReadRequest{State: prior}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	return modelOf[RouteResourceModel](t, resp.State)
}

// The API stores notify_emails as a list and answers it joined with ", ". A configured value that
// lists the same addresses in the same order keeps its text after create, refresh and update, so
// "a,b" shows no diff against the API's "a, b". A different list is real drift and reads back as
// the API's.
func TestRouteNotifyEmails_KeepsConfiguredSpacing(t *testing.T) {
	f := &fakeRouteAPI{t: t, route: baseRouteJSON()}
	r := &RouteResource{client: testClient(t, f.server())}

	plan := newRoutePlan()
	plan.NotifyEmails = types.StringValue("ops@example.com,dev@example.com")
	f.route["notifyEmails"] = "ops@example.com, dev@example.com"
	st, got := createRoute(t, r, plan)
	if got.NotifyEmails.ValueString() != "ops@example.com,dev@example.com" {
		t.Errorf("after create notify_emails = %q, want the configured text", got.NotifyEmails.ValueString())
	}
	// The body carries the entries trimmed and joined the way the API joins them.
	if f.lastCreate["notifyEmails"] != "ops@example.com, dev@example.com" {
		t.Errorf("create body notifyEmails = %v", f.lastCreate["notifyEmails"])
	}
	if got := readRoute(t, r, st); got.NotifyEmails.ValueString() != "ops@example.com,dev@example.com" {
		t.Errorf("after refresh notify_emails = %q, want the configured text", got.NotifyEmails.ValueString())
	}

	// Update to one address; the API answers it on its own.
	next := got
	next.NotifyEmails = types.StringValue(" ops@example.com ")
	f.route["notifyEmails"] = "ops@example.com"
	st, got = updateRouteFrom(t, r, st, next, next)
	if f.lastPatch["notifyEmails"] != "ops@example.com" {
		t.Errorf("patch body notifyEmails = %v, want the trimmed address", f.lastPatch["notifyEmails"])
	}
	if got.NotifyEmails.ValueString() != " ops@example.com " {
		t.Errorf("after update notify_emails = %q, want the configured text", got.NotifyEmails.ValueString())
	}

	// The API keeps only the first of an address listed twice in any letter case. The configured
	// text lists the same addresses once read that way, so it is kept rather than failing the apply.
	next = got
	next.NotifyEmails = types.StringValue("ops@example.com, dev@example.com, OPS@example.com")
	f.route["notifyEmails"] = "ops@example.com, dev@example.com"
	st, got = updateRouteFrom(t, r, st, next, next)
	if f.lastPatch["notifyEmails"] != "ops@example.com, dev@example.com" {
		t.Errorf("patch body notifyEmails = %v, want the repeat dropped", f.lastPatch["notifyEmails"])
	}
	if got.NotifyEmails.ValueString() != "ops@example.com, dev@example.com, OPS@example.com" {
		t.Errorf("after update with a repeat notify_emails = %q, want the configured text", got.NotifyEmails.ValueString())
	}

	// Changed outside Terraform: the refresh takes the API's list, so the next plan shows the diff.
	f.route["notifyEmails"] = "oncall@example.com, ops@example.com"
	if got := readRoute(t, r, st); got.NotifyEmails.ValueString() != "oncall@example.com, ops@example.com" {
		t.Errorf("after drift notify_emails = %q, want the API's list", got.NotifyEmails.ValueString())
	}
	// The same addresses in another order are drift too: notifications go out in the API's order.
	f.route["notifyEmails"] = "dev@example.com, ops@example.com"
	old := got
	old.NotifyEmails = types.StringValue("ops@example.com,dev@example.com")
	if got := readRoute(t, r, stateOf(t, r, &old)); got.NotifyEmails.ValueString() != "dev@example.com, ops@example.com" {
		t.Errorf("reordered notify_emails = %q, want the API's order", got.NotifyEmails.ValueString())
	}
}

// Removing notify_emails sends "" (the API's clear value) and reads back null. A configured "" lists
// no addresses: it is sent as "", and the API's null keeps the configured "" rather than failing the
// apply with an inconsistent result.
func TestRouteNotifyEmails_Clearing(t *testing.T) {
	f := &fakeRouteAPI{t: t, route: baseRouteJSON()}
	r := &RouteResource{client: testClient(t, f.server())}

	plan := newRoutePlan()
	plan.NotifyEmails = types.StringValue("ops@example.com")
	f.route["notifyEmails"] = "ops@example.com"
	st, applied := createRoute(t, r, plan)

	removed := applied
	removed.NotifyEmails = types.StringNull()
	f.route["notifyEmails"] = nil
	st, got := updateRouteFrom(t, r, st, removed, removed)
	if v, ok := f.lastPatch["notifyEmails"]; !ok || v != "" {
		t.Errorf("patch body notifyEmails = %v (present %v), want \"\"", v, ok)
	}
	if !got.NotifyEmails.IsNull() {
		t.Errorf("after removal notify_emails = %v, want null", got.NotifyEmails)
	}

	// With nothing left to clear, another apply sends nothing.
	updateRouteFrom(t, r, st, got, got)
	if v, ok := f.lastPatch["notifyEmails"]; ok {
		t.Errorf("patch body carried notifyEmails = %v with nothing to clear", v)
	}

	empty := newRoutePlan()
	empty.NotifyEmails = types.StringValue("")
	if _, got := createRoute(t, r, empty); got.NotifyEmails.IsNull() || got.NotifyEmails.ValueString() != "" {
		t.Errorf("configured \"\" read back as %v, want \"\"", got.NotifyEmails)
	}
	if v, ok := f.lastCreate["notifyEmails"]; !ok || v != "" {
		t.Errorf("create body notifyEmails = %v (present %v), want \"\"", v, ok)
	}
}

// An API from before the notify_emails fix answers the column as it is stored: its JSON text in a
// string, or the JSON array itself. Either reads as the same list instead of failing the refresh
// or reading the brackets in as addresses.
func TestRouteNotifyEmails_ReadsStoredArray(t *testing.T) {
	for name, stored := range map[string]any{
		"array":     []string{"ops@example.com", "dev@example.com"},
		"json text": `["ops@example.com", "dev@example.com"]`,
	} {
		t.Run(name, func(t *testing.T) { testRouteNotifyEmailsReadsStored(t, stored) })
	}
}

func testRouteNotifyEmailsReadsStored(t *testing.T, stored any) {
	route := baseRouteJSON()
	route["notifyEmails"] = stored
	f := &fakeRouteAPI{t: t, route: route}
	r := &RouteResource{client: testClient(t, f.server())}

	old := newRoutePlan()
	old.ID = types.StringValue("rt_1")
	old.CircuitCooldownSeconds, old.CircuitProbeSuccessThreshold, old.CircuitFailureThreshold = types.Int64Null(), types.Int64Null(), types.Int64Null()
	old.HoldOnCircuitOpen = types.BoolNull()
	old.CreatedAt, old.UpdatedAt = types.StringNull(), types.StringNull()
	if got := readRoute(t, r, stateOf(t, r, &old)); got.NotifyEmails.ValueString() != "ops@example.com, dev@example.com" {
		t.Errorf("notify_emails = %q, want the stored list joined", got.NotifyEmails.ValueString())
	}

	old.NotifyEmails = types.StringValue("ops@example.com,dev@example.com")
	if got := readRoute(t, r, stateOf(t, r, &old)); got.NotifyEmails.ValueString() != "ops@example.com,dev@example.com" {
		t.Errorf("notify_emails = %q, want the prior text kept", got.NotifyEmails.ValueString())
	}
}

// Each entry of notify_emails must be an email address at plan time; empty entries and spaces
// around the commas are allowed.
func TestRouteNotifyEmails_Validator(t *testing.T) {
	attr := schemaOf(t, &RouteResource{}).Attributes["notify_emails"].(schema.StringAttribute)
	if len(attr.Validators) == 0 {
		t.Fatal("notify_emails has no validator")
	}
	validate := func(v types.String) []string {
		var bad []string
		for _, val := range attr.Validators {
			resp := &validator.StringResponse{}
			val.ValidateString(context.Background(), validator.StringRequest{Path: path.Root("notify_emails"), ConfigValue: v}, resp)
			for _, d := range resp.Diagnostics.Errors() {
				bad = append(bad, d.Detail())
			}
		}
		return bad
	}
	for _, ok := range []string{
		"ops@example.com",
		"ops@example.com,dev@example.com",
		" ops@example.com ,  first.last+tag@sub.example.co.uk ",
		"ops@example.com,,dev@example.com,",
		"",
	} {
		if bad := validate(types.StringValue(ok)); len(bad) != 0 {
			t.Errorf("%q refused: %v", ok, bad)
		}
	}
	for value, want := range map[string]int{
		"ops":                             1,
		"ops@example":                     1,
		"ops@example.com;dev@example.com": 1,
		"ops@example.com dev@example.com": 1,
		"Ops <ops@example.com>":           1,
		"ops@@example.com":                1,
		"ops@example..com":                1,
		"a@example.com,bad,also-bad":      2,
	} {
		if bad := validate(types.StringValue(value)); len(bad) != want {
			t.Errorf("%q: %d errors %v, want %d", value, len(bad), bad, want)
		}
	}
	if bad := validate(types.StringUnknown()); len(bad) != 0 {
		t.Errorf("unknown value refused: %v", bad)
	}
}
