package resources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

// Schema-generic versions of the source test helpers, for the other resources' unit tests.

func schemaOf(t *testing.T, r resource.Resource) schema.Schema {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("schema: %v", resp.Diagnostics)
	}
	return resp.Schema
}

func nullStateOf(t *testing.T, r resource.Resource) tfsdk.State {
	t.Helper()
	s := schemaOf(t, r)
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
}

func planOf(t *testing.T, r resource.Resource, m any) tfsdk.Plan {
	t.Helper()
	s := schemaOf(t, r)
	plan := tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
	if d := plan.Set(context.Background(), m); d.HasError() {
		t.Fatalf("plan.Set: %v", d)
	}
	return plan
}

func stateOf(t *testing.T, r resource.Resource, m any) tfsdk.State {
	t.Helper()
	st := nullStateOf(t, r)
	if d := st.Set(context.Background(), m); d.HasError() {
		t.Fatalf("state.Set: %v", d)
	}
	return st
}

// modelOf reads a state back into a model and fails if any value is still unknown, which
// Terraform rejects after apply.
func modelOf[M any](t *testing.T, st tfsdk.State) M {
	t.Helper()
	var m M
	if d := st.Get(context.Background(), &m); d.HasError() {
		t.Fatalf("state.Get: %v", d)
	}
	if !st.Raw.IsFullyKnown() {
		t.Fatal("state has unknown values after the operation")
	}
	return m
}

// testClient points a client at srv, closing it when the test ends.
func testClient(t *testing.T, srv *httptest.Server) *client.Client {
	t.Helper()
	t.Cleanup(srv.Close)
	return &client.Client{BaseURL: srv.URL, OrganizationID: "org-1", APIKey: "key", HTTPClient: srv.Client()}
}

func writeTestJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// toJSONMap marshals v and reads it back as a generic map: the JSON body a request struct sends.
func toJSONMap(t *testing.T, v any) map[string]any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
