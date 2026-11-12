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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

// fakeRouteAPI answers the route endpoints in the API's shapes: GET returns filterConditions as
// its stored JSON text, PATCH /routes/:id and PATCH /routes/:id/circuit-config answer
// {success:true}.
type fakeRouteAPI struct {
	t           *testing.T
	route       map[string]any // what GET answers under "route"
	calls       []string
	lastCreate  map[string]any
	lastPatch   map[string]any
	lastCircuit map[string]any
}

func (f *fakeRouteAPI) server() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/api/organizations/org-1")
		f.calls = append(f.calls, r.Method+" "+p)
		body, _ := io.ReadAll(r.Body)
		switch {
		case r.Method == http.MethodPost && p == "/routes":
			_ = json.Unmarshal(body, &f.lastCreate)
			writeTestJSON(w, http.StatusCreated, map[string]any{"route": map[string]any{"id": "rt_1"}})
		case r.Method == http.MethodGet && p == "/routes/rt_1":
			writeTestJSON(w, http.StatusOK, map[string]any{"route": f.route})
		case r.Method == http.MethodPatch && p == "/routes/rt_1":
			f.lastPatch = map[string]any{}
			_ = json.Unmarshal(body, &f.lastPatch)
			writeTestJSON(w, http.StatusOK, map[string]any{"success": true})
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
		HoldOnCircuitOpen:            types.BoolValue(true),
		CreatedAt:                    types.StringUnknown(),
		UpdatedAt:                    types.StringUnknown(),
	}
}

func createRoute(t *testing.T, r *RouteResource, plan RouteResourceModel) (tfsdk.State, RouteResourceModel) {
	t.Helper()
	resp := resource.CreateResponse{State: nullStateOf(t, r)}
	r.Create(context.Background(), resource.CreateRequest{Plan: planOf(t, r, &plan)}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Create: %v", resp.Diagnostics)
	}
	return resp.State, modelOf[RouteResourceModel](t, resp.State)
}

// The create body always carries holdOnCircuitOpen, and inline filter conditions, which the API
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

	// Unset, the default true is sent explicitly.
	f.route = baseRouteJSON()
	createRoute(t, r, newRoutePlan())
	if v := f.lastCreate["holdOnCircuitOpen"]; v != true {
		t.Errorf("create body holdOnCircuitOpen = %v, want true", v)
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

	update := func(plan RouteResourceModel) RouteResourceModel {
		t.Helper()
		f.lastPatch, f.lastCircuit = nil, nil
		resp := resource.UpdateResponse{State: nullStateOf(t, r)}
		r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, r, &plan), State: st}, &resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("Update: %v", resp.Diagnostics)
		}
		return modelOf[RouteResourceModel](t, resp.State)
	}

	plan := applied
	plan.Name = types.StringValue("renamed")
	plan.HoldOnCircuitOpen = types.BoolValue(false)
	f.route["name"], f.route["holdOnCircuitOpen"] = "renamed", false
	got := update(plan)
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
	f.route["circuitCooldownSeconds"] = 120
	got = update(plan)
	if f.lastCircuit == nil || f.lastCircuit["circuitCooldownSeconds"] != float64(120) || len(f.lastCircuit) != 1 {
		t.Errorf("circuit-config body = %v, want only circuitCooldownSeconds 120", f.lastCircuit)
	}
	if got.CircuitCooldownSeconds.ValueInt64() != 120 {
		t.Errorf("circuit_cooldown_seconds = %v", got.CircuitCooldownSeconds)
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
		"expectedResponse": "null", "failoverDestinationIds": "[]", "holdOnCircuitOpen": "true",
	} {
		got, ok := body[key]
		b, _ := json.Marshal(got)
		if !ok || string(b) != want {
			t.Errorf("%s = %s (present %v), want %s", key, b, ok, want)
		}
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

// Webhook endpoints send hold_on_circuit_open on create and update and read it back.
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
	r.Create(context.Background(), resource.CreateRequest{Plan: planOf(t, r, &plan)}, &createResp)
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

	plan = applied
	plan.HoldOnCircuitOpen = types.BoolValue(true)
	hold = true
	updateResp := resource.UpdateResponse{State: nullStateOf(t, r)}
	r.Update(context.Background(), resource.UpdateRequest{Plan: planOf(t, r, &plan), State: createResp.State}, &updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update: %v", updateResp.Diagnostics)
	}
	if lastPatch["holdOnCircuitOpen"] != true {
		t.Errorf("patch body holdOnCircuitOpen = %v, want true", lastPatch["holdOnCircuitOpen"])
	}
	if got := modelOf[WebhookEndpointResourceModel](t, updateResp.State); !got.HoldOnCircuitOpen.ValueBool() {
		t.Error("hold_on_circuit_open = false after update, want true")
	}
}
