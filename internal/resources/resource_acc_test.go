package resources_test

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/hookbase/terraform-provider-hookbase/internal/client"
	"github.com/hookbase/terraform-provider-hookbase/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"hookbase": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// testAccProviderConfig returns the provider block that reads credentials from
// environment variables (HOOKBASE_API_KEY, HOOKBASE_API_URL, HOOKBASE_ORG_ID).
const testAccProviderConfig = `
provider "hookbase" {}
`

// --- Source ---

func TestAccSourceResource_basic(t *testing.T) {
	slug := fmt.Sprintf("tf-acc-src-%d", acctest.RandInt())

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and verify
			{
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_source" "test" {
  name          = "tf-acc-test-source"
  slug          = %q
  provider_type = "custom"
}
`, slug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_source.test", "id"),
					resource.TestCheckResourceAttr("hookbase_source.test", "name", "tf-acc-test-source"),
					resource.TestCheckResourceAttr("hookbase_source.test", "slug", slug),
					resource.TestCheckResourceAttr("hookbase_source.test", "provider_type", "custom"),
					resource.TestCheckResourceAttr("hookbase_source.test", "is_active", "true"),
					resource.TestCheckResourceAttrSet("hookbase_source.test", "signing_secret"),
					resource.TestCheckResourceAttrSet("hookbase_source.test", "ingest_url"),
					resource.TestCheckResourceAttrSet("hookbase_source.test", "created_at"),
				),
			},
			// Update name and verify
			{
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_source" "test" {
  name          = "tf-acc-test-source-updated"
  slug          = %q
  provider_type = "custom"
}
`, slug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hookbase_source.test", "name", "tf-acc-test-source-updated"),
					resource.TestCheckResourceAttr("hookbase_source.test", "slug", slug),
				),
			},
		},
	})
}

func TestAccSourceResource_import(t *testing.T) {
	slug := fmt.Sprintf("tf-acc-src-imp-%d", acctest.RandInt())

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_source" "test" {
  name          = "tf-acc-import-source"
  slug          = %q
  provider_type = "custom"
}
`, slug),
			},
			// Import and verify
			{
				ResourceName:            "hookbase_source.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"signing_secret"},
			},
		},
	})
}

// --- Transform ---

func TestAccTransformResource_basic(t *testing.T) {
	name := fmt.Sprintf("tf-acc-transform-%d", acctest.RandInt())

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_transform" "test" {
  name           = %q
  code           = "{ \"mapped\": $.input }"
  transform_type = "jsonata"
}
`, name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_transform.test", "id"),
					resource.TestCheckResourceAttr("hookbase_transform.test", "name", name),
					resource.TestCheckResourceAttr("hookbase_transform.test", "code", "{ \"mapped\": $.input }"),
					resource.TestCheckResourceAttr("hookbase_transform.test", "transform_type", "jsonata"),
					resource.TestCheckResourceAttr("hookbase_transform.test", "input_format", "json"),
					resource.TestCheckResourceAttr("hookbase_transform.test", "output_format", "json"),
					resource.TestCheckResourceAttrSet("hookbase_transform.test", "created_at"),
				),
			},
		},
	})
}

// --- Filter ---

func TestAccFilterResource_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig + `
resource "hookbase_filter" "test" {
  name  = "tf-acc-test-filter"
  logic = "AND"
  conditions = jsonencode([{
    field    = "type"
    operator = "equals"
    value    = "order.created"
  }])
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_filter.test", "id"),
					resource.TestCheckResourceAttr("hookbase_filter.test", "name", "tf-acc-test-filter"),
					resource.TestCheckResourceAttr("hookbase_filter.test", "logic", "AND"),
					resource.TestCheckResourceAttrSet("hookbase_filter.test", "conditions"),
					resource.TestCheckResourceAttrSet("hookbase_filter.test", "created_at"),
				),
			},
		},
	})
}

// --- Destination ---

func TestAccDestinationResource_basic(t *testing.T) {
	slug := fmt.Sprintf("tf-acc-dest-%d", acctest.RandInt())

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// An empty headers map stays {} instead of reading back as null.
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_destination" "test" {
  name    = "tf-acc-test-dest"
  slug    = %q
  url     = "https://httpbin.org/post"
  headers = {}
}
`, slug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "id"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "name", "tf-acc-test-dest"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "slug", slug),
					resource.TestCheckResourceAttr("hookbase_destination.test", "url", "https://httpbin.org/post"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "method", "POST"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "type", "http"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "is_active", "true"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "headers.%", "0"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "has_signing_secret", "true"),
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "signing_secret"),
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "created_at"),
				),
			},
			{
				// PATCH answers {success:true}; the update must still leave a complete state, with
				// nothing blanked: 0.6.x read the reply into state.
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_destination" "test" {
  name       = "tf-acc-test-dest-renamed"
  slug       = %q
  url        = "https://httpbin.org/anything"
  timeout_ms = 15000
  headers = {
    "X-Env" = "acc"
  }
}

data "hookbase_destination" "by_slug" {
  slug = hookbase_destination.test.slug
}
`, slug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "id"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "name", "tf-acc-test-dest-renamed"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "slug", slug),
					resource.TestCheckResourceAttr("hookbase_destination.test", "url", "https://httpbin.org/anything"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "method", "POST"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "type", "http"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "auth_type", "none"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "timeout_ms", "15000"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "is_active", "true"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "headers.%", "1"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "headers.X-Env", "acc"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "has_signing_secret", "true"),
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "signing_secret"),
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "created_at"),
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "updated_at"),
					resource.TestCheckResourceAttrPair("data.hookbase_destination.by_slug", "id", "hookbase_destination.test", "id"),
					resource.TestCheckResourceAttr("data.hookbase_destination.by_slug", "name", "tf-acc-test-dest-renamed"),
					resource.TestCheckResourceAttr("data.hookbase_destination.by_slug", "type", "http"),
					resource.TestCheckResourceAttr("data.hookbase_destination.by_slug", "has_signing_secret", "true"),
				),
			},
			{
				// Import reads signing_secret through reveal-secret, so the whole state matches.
				ResourceName:      "hookbase_destination.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// A bridge destination publishes each delivery as an outbound event. It takes a config and no url,
// and has no signing secret.
func TestAccDestinationResource_bridge(t *testing.T) {
	rnd := acctest.RandInt()
	slug := fmt.Sprintf("tf-acc-bridge-%d", rnd)
	evtName := fmt.Sprintf("test.acc.bridge.evt%d", rnd)
	extID := fmt.Sprintf("tf-acc-bridge-app-%d", rnd)

	config := func(name, bridgeConfig string) string {
		return testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_event_type" "test" {
  name = %q
}

resource "hookbase_webhook_application" "test" {
  name        = "tf-acc-bridge-app"
  external_id = %q
}

resource "hookbase_destination" "test" {
  name   = %q
  slug   = %q
  type   = "bridge"
  config = %s
}

data "hookbase_destination" "by_id" {
  id = hookbase_destination.test.id
}
`, evtName, extID, name, slug, bridgeConfig)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("tf-acc-bridge", `jsonencode({
    eventTypeSource = "fixed"
    eventType       = hookbase_event_type.test.name
    applicationId   = hookbase_webhook_application.test.id
  })`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "id"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "type", "bridge"),
					resource.TestCheckNoResourceAttr("hookbase_destination.test", "url"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "has_signing_secret", "false"),
					resource.TestCheckNoResourceAttr("hookbase_destination.test", "signing_secret"),
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "config"),
					resource.TestCheckResourceAttr("data.hookbase_destination.by_id", "type", "bridge"),
					resource.TestCheckResourceAttr("data.hookbase_destination.by_id", "has_signing_secret", "false"),
				),
			},
			{
				// A rename keeps the config; a config change is sent and read back.
				Config: config("tf-acc-bridge-renamed", `jsonencode({
    eventTypeSource = "field"
    eventTypeField  = "data.type"
    eventTypePrefix = "acc."
    applicationId   = hookbase_webhook_application.test.id
  })`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hookbase_destination.test", "name", "tf-acc-bridge-renamed"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "slug", slug),
					resource.TestCheckResourceAttr("hookbase_destination.test", "type", "bridge"),
					resource.TestCheckNoResourceAttr("hookbase_destination.test", "url"),
					resource.TestCheckResourceAttr("hookbase_destination.test", "has_signing_secret", "false"),
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "created_at"),
					resource.TestCheckResourceAttr("data.hookbase_destination.by_id", "name", "tf-acc-bridge-renamed"),
				),
			},
		},
	})
}

// --- Route ---

func TestAccRouteResource_basic(t *testing.T) {
	srcSlug := fmt.Sprintf("tf-acc-rt-src-%d", acctest.RandInt())
	destSlug := fmt.Sprintf("tf-acc-rt-dest-%d", acctest.RandInt())

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_source" "test" {
  name          = "tf-acc-route-source"
  slug          = %q
  provider_type = "custom"
}

resource "hookbase_destination" "test" {
  name = "tf-acc-route-dest"
  slug = %q
  url  = "https://httpbin.org/post"
}

resource "hookbase_route" "test" {
  name           = "tf-acc-test-route"
  source_id      = hookbase_source.test.id
  destination_id = hookbase_destination.test.id
}
`, srcSlug, destSlug),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_route.test", "id"),
					resource.TestCheckResourceAttr("hookbase_route.test", "name", "tf-acc-test-route"),
					resource.TestCheckResourceAttrPair("hookbase_route.test", "source_id", "hookbase_source.test", "id"),
					resource.TestCheckResourceAttrPair("hookbase_route.test", "destination_id", "hookbase_destination.test", "id"),
					resource.TestCheckResourceAttr("hookbase_route.test", "is_active", "true"),
					resource.TestCheckResourceAttr("hookbase_route.test", "hold_on_circuit_open", "true"),
					resource.TestCheckResourceAttrSet("hookbase_route.test", "created_at"),
				),
			},
		},
	})
}

// --- Event Type ---

func TestAccEventTypeResource_basic(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig + `
resource "hookbase_event_type" "test" {
  name        = "test.acc.event"
  description = "Acceptance test event type"
  category    = "testing"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_event_type.test", "id"),
					resource.TestCheckResourceAttr("hookbase_event_type.test", "name", "test.acc.event"),
					resource.TestCheckResourceAttr("hookbase_event_type.test", "description", "Acceptance test event type"),
					resource.TestCheckResourceAttr("hookbase_event_type.test", "category", "testing"),
					resource.TestCheckResourceAttr("hookbase_event_type.test", "is_enabled", "true"),
					resource.TestCheckResourceAttrSet("hookbase_event_type.test", "created_at"),
				),
			},
		},
	})
}

// --- Webhook Application ---

func TestAccWebhookApplicationResource_basic(t *testing.T) {
	extID := fmt.Sprintf("tf-acc-%d", acctest.RandInt())

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_webhook_application" "test" {
  name        = "tf-acc-test-app"
  external_id = %q
}
`, extID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_webhook_application.test", "id"),
					resource.TestCheckResourceAttr("hookbase_webhook_application.test", "name", "tf-acc-test-app"),
					resource.TestCheckResourceAttr("hookbase_webhook_application.test", "external_id", extID),
					resource.TestCheckResourceAttr("hookbase_webhook_application.test", "is_disabled", "false"),
					resource.TestCheckResourceAttrSet("hookbase_webhook_application.test", "created_at"),
				),
			},
		},
	})
}

// --- Webhook Endpoint ---

func TestAccWebhookEndpointResource_basic(t *testing.T) {
	extID := fmt.Sprintf("tf-acc-ep-%d", acctest.RandInt())

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_webhook_application" "test" {
  name        = "tf-acc-ep-app"
  external_id = %q
}

resource "hookbase_webhook_endpoint" "test" {
  application_id  = hookbase_webhook_application.test.id
  url             = "https://httpbin.org/post"
  description     = "Acceptance test endpoint"
  timeout_seconds = 15
}
`, extID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_webhook_endpoint.test", "id"),
					resource.TestCheckResourceAttrPair("hookbase_webhook_endpoint.test", "application_id", "hookbase_webhook_application.test", "id"),
					resource.TestCheckResourceAttr("hookbase_webhook_endpoint.test", "url", "https://httpbin.org/post"),
					resource.TestCheckResourceAttr("hookbase_webhook_endpoint.test", "description", "Acceptance test endpoint"),
					resource.TestCheckResourceAttr("hookbase_webhook_endpoint.test", "timeout_seconds", "15"),
					resource.TestCheckResourceAttr("hookbase_webhook_endpoint.test", "is_disabled", "false"),
					resource.TestCheckResourceAttr("hookbase_webhook_endpoint.test", "hold_on_circuit_open", "true"),
					resource.TestCheckResourceAttrSet("hookbase_webhook_endpoint.test", "secret"),
					resource.TestCheckResourceAttrSet("hookbase_webhook_endpoint.test", "secret_version"),
					resource.TestCheckResourceAttrSet("hookbase_webhook_endpoint.test", "created_at"),
				),
			},
		},
	})
}

// --- Webhook Subscription ---

func TestAccWebhookSubscriptionResource_basic(t *testing.T) {
	extID := fmt.Sprintf("tf-acc-sub-%d", acctest.RandInt())
	evtName := fmt.Sprintf("test.acc.sub.evt%d", acctest.RandInt())

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_event_type" "test" {
  name        = %q
  description = "Acceptance test event type for subscription"
}

resource "hookbase_webhook_application" "test" {
  name        = "tf-acc-sub-app"
  external_id = %q
}

resource "hookbase_webhook_endpoint" "test" {
  application_id = hookbase_webhook_application.test.id
  url            = "https://httpbin.org/post"
}

resource "hookbase_webhook_subscription" "test" {
  endpoint_id   = hookbase_webhook_endpoint.test.id
  event_type_id = hookbase_event_type.test.id
}
`, evtName, extID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hookbase_webhook_subscription.test", "id"),
					resource.TestCheckResourceAttrPair("hookbase_webhook_subscription.test", "endpoint_id", "hookbase_webhook_endpoint.test", "id"),
					resource.TestCheckResourceAttrPair("hookbase_webhook_subscription.test", "event_type_id", "hookbase_event_type.test", "id"),
					resource.TestCheckResourceAttr("hookbase_webhook_subscription.test", "is_enabled", "true"),
					resource.TestCheckResourceAttrSet("hookbase_webhook_subscription.test", "created_at"),
				),
			},
		},
	})
}

// --- 0.7.0: route updates, hold_on_circuit_open, notify_emails ---

// testAccRouteConfig is a source, a destination and a route whose body is routeBody.
func testAccRouteConfig(srcSlug, destSlug, routeBody string) string {
	return testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_source" "test" {
  name          = "tf-acc-route-source"
  slug          = %q
  provider_type = "custom"
}

resource "hookbase_destination" "test" {
  name = "tf-acc-route-dest"
  slug = %q
  url  = "https://httpbin.org/post"
}

resource "hookbase_route" "test" {
  source_id      = hookbase_source.test.id
  destination_id = hookbase_destination.test.id
%s
}

data "hookbase_route" "test" {
  id = hookbase_route.test.id
}
`, srcSlug, destSlug, routeBody)
}

// Route updates read the route back (PATCH answers {success:true}, which 0.6.x read into state as
// blanks), circuit settings go through circuit-config, removing filter_conditions clears them, and
// hold_on_circuit_open is sent when set and keeps the server's value when unset.
func TestAccRouteResource_update(t *testing.T) {
	rnd := acctest.RandInt()
	srcSlug := fmt.Sprintf("tf-acc-rtu-src-%d", rnd)
	destSlug := fmt.Sprintf("tf-acc-rtu-dest-%d", rnd)
	const rt, ds = "hookbase_route.test", "data.hookbase_route.test"
	wiring := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttrSet(rt, "id"),
		resource.TestCheckResourceAttrPair(rt, "source_id", "hookbase_source.test", "id"),
		resource.TestCheckResourceAttrPair(rt, "destination_id", "hookbase_destination.test", "id"),
		resource.TestCheckResourceAttrSet(rt, "created_at"),
		resource.TestCheckResourceAttrSet(rt, "updated_at"),
		resource.TestCheckResourceAttrPair(ds, "source_id", "hookbase_source.test", "id"),
		resource.TestCheckResourceAttrPair(ds, "destination_id", "hookbase_destination.test", "id"),
	)
	filter := `
  filter_conditions = jsonencode([{
    field    = "type"
    operator = "equals"
    value    = "order.created"
  }])`

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// hold_on_circuit_open unset: the API default.
				Config: testAccRouteConfig(srcSlug, destSlug, `
  name     = "tf-acc-rtu"
  priority = 2`+filter),
				Check: resource.ComposeAggregateTestCheckFunc(wiring,
					resource.TestCheckResourceAttr(rt, "name", "tf-acc-rtu"),
					resource.TestCheckResourceAttr(rt, "priority", "2"),
					resource.TestCheckResourceAttr(rt, "filter_logic", "AND"),
					resource.TestCheckResourceAttrSet(rt, "filter_conditions"),
					resource.TestCheckResourceAttr(rt, "hold_on_circuit_open", "true"),
					resource.TestCheckResourceAttr(ds, "hold_on_circuit_open", "true"),
				),
			},
			{
				Config: testAccRouteConfig(srcSlug, destSlug, `
  name                      = "tf-acc-rtu-renamed"
  priority                  = 2
  hold_on_circuit_open      = false
  circuit_cooldown_seconds  = 120
  circuit_failure_threshold = 10`+filter),
				Check: resource.ComposeAggregateTestCheckFunc(wiring,
					resource.TestCheckResourceAttr(rt, "name", "tf-acc-rtu-renamed"),
					resource.TestCheckResourceAttr(rt, "priority", "2"),
					resource.TestCheckResourceAttr(rt, "is_active", "true"),
					resource.TestCheckResourceAttr(rt, "filter_logic", "AND"),
					resource.TestCheckResourceAttrSet(rt, "filter_conditions"),
					resource.TestCheckResourceAttr(rt, "circuit_cooldown_seconds", "120"),
					resource.TestCheckResourceAttr(rt, "circuit_failure_threshold", "10"),
					resource.TestCheckResourceAttr(rt, "hold_on_circuit_open", "false"),
					resource.TestCheckResourceAttr(ds, "name", "tf-acc-rtu-renamed"),
					resource.TestCheckResourceAttr(ds, "hold_on_circuit_open", "false"),
				),
			},
			{
				// Removing filter_conditions clears them on the server; the post-apply plan would
				// show them coming back otherwise.
				Config: testAccRouteConfig(srcSlug, destSlug, `
  name                 = "tf-acc-rtu-renamed"
  priority             = 2
  hold_on_circuit_open = true`),
				Check: resource.ComposeAggregateTestCheckFunc(wiring,
					resource.TestCheckNoResourceAttr(rt, "filter_conditions"),
					resource.TestCheckResourceAttr(rt, "hold_on_circuit_open", "true"),
					resource.TestCheckResourceAttr(rt, "circuit_cooldown_seconds", "120"),
					resource.TestCheckResourceAttr(ds, "hold_on_circuit_open", "true"),
				),
			},
			{
				Config: testAccRouteConfig(srcSlug, destSlug, `
  name                 = "tf-acc-rtu-renamed"
  priority             = 2
  hold_on_circuit_open = false`),
				Check: resource.ComposeAggregateTestCheckFunc(wiring,
					resource.TestCheckResourceAttr(rt, "hold_on_circuit_open", "false"),
				),
			},
			{
				// Unset again, with another change: the route keeps false rather than taking the
				// API default.
				Config: testAccRouteConfig(srcSlug, destSlug, `
  name     = "tf-acc-rtu-last"
  priority = 5`),
				Check: resource.ComposeAggregateTestCheckFunc(wiring,
					resource.TestCheckResourceAttr(rt, "name", "tf-acc-rtu-last"),
					resource.TestCheckResourceAttr(rt, "priority", "5"),
					resource.TestCheckResourceAttr(rt, "hold_on_circuit_open", "false"),
					resource.TestCheckResourceAttr(rt, "circuit_cooldown_seconds", "120"),
					resource.TestCheckResourceAttr(rt, "circuit_failure_threshold", "10"),
					resource.TestCheckResourceAttr(ds, "hold_on_circuit_open", "false"),
				),
			},
		},
	})
}

// testAccCheckRouteNotifyEmails reads the route from the API and checks its notify list.
func testAccCheckRouteNotifyEmails(name string, want ...string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
		if !ok {
			return fmt.Errorf("%s not in state", name)
		}
		// The same API the provider talks to: HOOKBASE_API_URL, or the provider's default.
		apiURL := os.Getenv("HOOKBASE_API_URL")
		if apiURL == "" {
			apiURL = "https://api.hookbase.app"
		}
		c := client.New(apiURL, os.Getenv("HOOKBASE_ORG_ID"), os.Getenv("HOOKBASE_API_KEY"))
		route, err := c.GetRoute(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}
		if !slices.Equal([]string(route.NotifyEmails), want) {
			return fmt.Errorf("API notifyEmails = %q, want %q", route.NotifyEmails, want)
		}
		return nil
	}
}

// notify_emails is a comma-separated string. The API stores the list and returns it joined with
// ", "; state keeps the configured text when the list is the same, so the post-apply plan of each
// step is empty whatever the spacing. Needs the API's notify_emails fix: without it, every
// non-empty value fails with a 500.
func TestAccRouteResource_notifyEmails(t *testing.T) {
	rnd := acctest.RandInt()
	srcSlug := fmt.Sprintf("tf-acc-rtn-src-%d", rnd)
	destSlug := fmt.Sprintf("tf-acc-rtn-dest-%d", rnd)
	const rt = "hookbase_route.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Caught at plan time.
				Config: testAccRouteConfig(srcSlug, destSlug, `
  name          = "tf-acc-rtn"
  notify_emails = "ops@example.com;dev@example.com"`),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`is not an email address`),
			},
			{
				Config: testAccRouteConfig(srcSlug, destSlug, `
  name              = "tf-acc-rtn"
  notify_on_failure = true
  notify_emails     = "ops@example.com, dev@example.com"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rt, "notify_emails", "ops@example.com, dev@example.com"),
					testAccCheckRouteNotifyEmails(rt, "ops@example.com", "dev@example.com"),
				),
			},
			{
				// One address changed, no space after the comma: the API's "a, b" is no diff.
				Config: testAccRouteConfig(srcSlug, destSlug, `
  name              = "tf-acc-rtn"
  notify_on_failure = true
  notify_emails     = "ops@example.com,oncall@example.com"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(rt, "notify_emails", "ops@example.com,oncall@example.com"),
					testAccCheckRouteNotifyEmails(rt, "ops@example.com", "oncall@example.com"),
				),
			},
			{
				// Removed from configuration: cleared on the server.
				Config: testAccRouteConfig(srcSlug, destSlug, `
  name              = "tf-acc-rtn"
  notify_on_failure = true`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(rt, "notify_emails"),
					testAccCheckRouteNotifyEmails(rt),
				),
			},
		},
	})
}

// --- 0.7.0: webhook endpoint hold_on_circuit_open ---

func TestAccWebhookEndpointResource_holdOnCircuitOpen(t *testing.T) {
	extID := fmt.Sprintf("tf-acc-eph-%d", acctest.RandInt())
	const ep, ds = "hookbase_webhook_endpoint.test", "data.hookbase_webhook_endpoint.test"
	config := func(endpointBody string) string {
		return testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_webhook_application" "test" {
  name        = "tf-acc-eph-app"
  external_id = %q
}

resource "hookbase_webhook_endpoint" "test" {
  application_id = hookbase_webhook_application.test.id
  url            = "https://httpbin.org/post"
%s
}

data "hookbase_webhook_endpoint" "test" {
  id = hookbase_webhook_endpoint.test.id
}
`, extID, endpointBody)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config(`  hold_on_circuit_open = false`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ep, "hold_on_circuit_open", "false"),
					resource.TestCheckResourceAttr(ds, "hold_on_circuit_open", "false"),
				),
			},
			{
				Config: config(`  hold_on_circuit_open = true`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ep, "hold_on_circuit_open", "true"),
					resource.TestCheckResourceAttr(ds, "hold_on_circuit_open", "true"),
				),
			},
			{
				Config: config(`  hold_on_circuit_open = false`),
				Check:  resource.TestCheckResourceAttr(ep, "hold_on_circuit_open", "false"),
			},
			{
				// Unset, with another change: the endpoint keeps false rather than taking the API
				// default.
				Config: config(`  description = "unset hold"`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(ep, "description", "unset hold"),
					resource.TestCheckResourceAttr(ep, "hold_on_circuit_open", "false"),
					resource.TestCheckResourceAttrPair(ep, "application_id", "hookbase_webhook_application.test", "id"),
					resource.TestCheckResourceAttr(ds, "hold_on_circuit_open", "false"),
					resource.TestCheckResourceAttr(ds, "description", "unset hold"),
				),
			},
		},
	})
}

// --- 0.7.0: source catalog_provider and empty lists ---

func TestAccSourceResource_catalogAndEmptyLists(t *testing.T) {
	slug := fmt.Sprintf("tf-acc-src-cat-%d", acctest.RandInt())
	const src, ds = "hookbase_source.test", "data.hookbase_source.test"
	config := func(name string) string {
		return testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_source" "test" {
  name             = %q
  slug             = %q
  provider_type    = "github"
  catalog_provider = "github"
  ip_allowlist     = []
  ip_denylist      = []
  encrypt_fields   = []
  mask_fields      = []
  allowed_methods  = []
}

data "hookbase_source" "test" {
  slug = hookbase_source.test.slug
}
`, name, slug)
	}
	emptyLists := resource.ComposeAggregateTestCheckFunc(
		resource.TestCheckResourceAttr(src, "ip_allowlist.#", "0"),
		resource.TestCheckResourceAttr(src, "ip_denylist.#", "0"),
		resource.TestCheckResourceAttr(src, "encrypt_fields.#", "0"),
		resource.TestCheckResourceAttr(src, "mask_fields.#", "0"),
		resource.TestCheckResourceAttr(src, "allowed_methods.#", "0"),
	)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config("tf-acc-src-cat"),
				Check: resource.ComposeAggregateTestCheckFunc(emptyLists,
					resource.TestCheckResourceAttr(src, "provider_type", "github"),
					resource.TestCheckResourceAttr(src, "catalog_provider", "github"),
					resource.TestCheckResourceAttr(ds, "catalog_provider", "github"),
					resource.TestCheckResourceAttrPair(ds, "id", src, "id"),
				),
			},
			{
				Config: config("tf-acc-src-cat-renamed"),
				Check: resource.ComposeAggregateTestCheckFunc(emptyLists,
					resource.TestCheckResourceAttr(src, "name", "tf-acc-src-cat-renamed"),
					resource.TestCheckResourceAttr(src, "catalog_provider", "github"),
					resource.TestCheckResourceAttr(ds, "name", "tf-acc-src-cat-renamed"),
					resource.TestCheckResourceAttr(ds, "catalog_provider", "github"),
				),
			},
		},
	})
}
