package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
				Config: testAccProviderConfig + fmt.Sprintf(`
resource "hookbase_destination" "test" {
  name = "tf-acc-test-dest"
  slug = %q
  url  = "https://httpbin.org/post"
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
					resource.TestCheckResourceAttrSet("hookbase_destination.test", "created_at"),
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
