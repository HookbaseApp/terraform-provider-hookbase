package resources

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
)

// clientResources builds every resource around c.
func clientResources(c *client.Client) map[string]resource.Resource {
	return map[string]resource.Resource{
		"destination":          &DestinationResource{client: c},
		"event_type":           &EventTypeResource{client: c},
		"filter":               &FilterResource{client: c},
		"route":                &RouteResource{client: c},
		"source":               &SourceResource{client: c},
		"transform":            &TransformResource{client: c},
		"webhook_application":  &WebhookApplicationResource{client: c},
		"webhook_endpoint":     &WebhookEndpointResource{client: c},
		"webhook_subscription": &WebhookSubscriptionResource{client: c},
	}
}

// idOnlyState is a state holding only an id, all else null: what an import starts from.
func idOnlyState(t *testing.T, r resource.Resource, id string) tfsdk.State {
	t.Helper()
	s := schemaOf(t, r)
	typ := s.Type().TerraformType(context.Background()).(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for name, at := range typ.AttributeTypes {
		vals[name] = tftypes.NewValue(at, nil)
	}
	vals["id"] = tftypes.NewValue(tftypes.String, id)
	return tfsdk.State{Schema: s, Raw: tftypes.NewValue(typ, vals)}
}

// statusAPI answers every request with status and records the method and path of each.
func statusAPI(t *testing.T, status int) (*client.Client, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var calls []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		mu.Unlock()
		writeTestJSON(w, status, map[string]string{"error": http.StatusText(status)})
	}))
	return testClient(t, srv), func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), calls...)
	}
}

// An object deleted outside Terraform answers 404. Read drops it from state, so the next plan
// creates it again instead of failing every refresh, and Delete counts it as deleted.
func TestNotFound_ReadRemovesAndDeleteSucceeds(t *testing.T) {
	c, calls := statusAPI(t, http.StatusNotFound)
	for name, r := range clientResources(c) {
		t.Run(name, func(t *testing.T) {
			before := len(calls())
			readResp := resource.ReadResponse{State: idOnlyState(t, r, "gone-1")}
			r.Read(context.Background(), resource.ReadRequest{State: idOnlyState(t, r, "gone-1")}, &readResp)
			if readResp.Diagnostics.HasError() {
				t.Errorf("Read: %v", readResp.Diagnostics)
			}
			if !readResp.State.Raw.IsNull() {
				t.Error("Read kept the object in state after a 404")
			}
			got := calls()[before:]
			if len(got) != 1 || !strings.HasPrefix(got[0], "GET ") || !strings.HasSuffix(got[0], "/gone-1") {
				t.Errorf("Read sent %v, want one GET of the object", got)
			}

			before = len(calls())
			delResp := resource.DeleteResponse{State: idOnlyState(t, r, "gone-1")}
			r.Delete(context.Background(), resource.DeleteRequest{State: idOnlyState(t, r, "gone-1")}, &delResp)
			if delResp.Diagnostics.HasError() {
				t.Errorf("Delete: %v", delResp.Diagnostics)
			}
			got = calls()[before:]
			if len(got) != 1 || !strings.HasPrefix(got[0], "DELETE ") {
				t.Errorf("Delete sent %v, want one DELETE", got)
			}
		})
	}
}

// Only a 404 means gone: any other error still fails Read and Delete and leaves the state alone.
func TestNotFound_OtherErrorsStillFail(t *testing.T) {
	c, _ := statusAPI(t, http.StatusForbidden)
	for name, r := range clientResources(c) {
		t.Run(name, func(t *testing.T) {
			readResp := resource.ReadResponse{State: idOnlyState(t, r, "rt-1")}
			r.Read(context.Background(), resource.ReadRequest{State: idOnlyState(t, r, "rt-1")}, &readResp)
			if !readResp.Diagnostics.HasError() {
				t.Error("Read: a 403 did not fail")
			}
			if readResp.State.Raw.IsNull() {
				t.Error("Read: a 403 removed the object from state")
			}

			delResp := resource.DeleteResponse{State: idOnlyState(t, r, "rt-1")}
			r.Delete(context.Background(), resource.DeleteRequest{State: idOnlyState(t, r, "rt-1")}, &delResp)
			if !delResp.Diagnostics.HasError() {
				t.Error("Delete: a 403 did not fail")
			}
		})
	}
}
