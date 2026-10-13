package provider_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"

	"github.com/uptimewarden/terraform-provider-uptimewarden/internal/provider"
)

func testProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"uptimewarden": providerserver.NewProtocol6WithError(provider.New("test")()),
	}
}

func providerBlock(url string) string {
	return fmt.Sprintf(`
provider "uptimewarden" {
  api_token = "test-token"
  base_url  = %q
}
`, url)
}

const res = "uptimewarden_monitor.test"

// Every group of fields round-trips: create sends them under the API's
// names (the fake rejects unknown fields), state reads them back, and a
// second plan is empty.
func TestMonitor_FullHTTPMonitor(t *testing.T) {
	api, srv := newFakeAPI(t)
	config := providerBlock(srv.URL) + fmt.Sprintf(`
resource "uptimewarden_monitor" "test" {
  name        = "Checkout API"
  type        = "http"
  url         = "https://api.acme.com/checkout/health"
  description = "checkout"
  interval    = 60
  timeout     = 10
  tags        = ["prod", "checkout"]
  groups      = ["Payments"]
  external_id = "checkout-api"

  region                 = "gwc"
  verification_regions   = ["sea", "ae"]
  min_confirming_regions = 2

  method                = "POST"
  request_body          = "{}"
  custom_headers        = { "X-Probe" = "uptimewarden" }
  expected_status_codes = "200-299"
  expected_body         = "ok"
  follow_redirects      = false
  verify_ssl            = true
  basic_auth_username   = "probe"
  basic_auth_password   = "s3cret"
  bearer_token          = "tok"
  content_validation = {
    enabled          = true
    type             = "text"
    expected_content = "ok"
  }

  retry_count             = 2
  response_time_threshold = 800
  ssl_expiry_days         = 14
  ssl_alerts = {
    thirty_days  = false
    fifteen_days = true
    seven_days   = true
    one_day      = true
  }

  alert_types   = ["down", "recovery", "ssl"]
  notifications = {
    email = { down = true, recovery = true }
    sms   = { down = true }
  }
  email_addresses      = "oncall@acme.com"
  phone_numbers        = "+15550100"
  voice_mode           = "all"
  escalation_policy_id = %q
  selected_webhook_ids = [%q]

  maintenance_windows = [{
    type     = "recurring"
    day      = "sunday"
    start    = "02:00"
    end      = "03:00"
    timezone = "Europe/Istanbul"
  }]
}
`, policyID, hookID)

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(res, "id", res, "public_id"),
					resource.TestCheckResourceAttr(res, "managed_by_source", "terraform"),
					resource.TestCheckResourceAttr(res, "timeout", "10"),
					resource.TestCheckResourceAttr(res, "groups.#", "1"),
					resource.TestCheckResourceAttr(res, "verification_regions.1", "ae"),
					resource.TestCheckResourceAttr(res, "custom_headers.X-Probe", "uptimewarden"),
					resource.TestCheckResourceAttr(res, "content_validation.type", "text"),
					resource.TestCheckResourceAttr(res, "ssl_alerts.thirty_days", "false"),
					resource.TestCheckResourceAttr(res, "notifications.sms.down", "true"),
					resource.TestCheckResourceAttr(res, "escalation_policy_id", policyID),
					resource.TestCheckResourceAttr(res, "maintenance_windows.0.day", "sunday"),
					resource.TestCheckResourceAttr(res, "basic_auth_password", "s3cret"),
					// Unset fields read back the API's defaults.
					resource.TestCheckResourceAttr(res, "domain_expiry_threshold", "30"),
					resource.TestCheckResourceAttr(res, "is_active", "true"),
					resource.TestCheckNoResourceAttr(res, "heartbeat_url"),
				),
			},
			{Config: config, PlanOnly: true},
		},
	})

	post := api.posts[0]
	if post["managedBySource"] != "terraform" {
		t.Errorf("managedBySource: %v", post["managedBySource"])
	}
	if _, sent := post["groups"]; sent {
		t.Error("groups must not be sent on create (PATCH-only)")
	}
	if len(api.patch) == 0 || api.patch[0]["groups"] == nil {
		t.Errorf("groups must follow in a PATCH, got %v", api.patch)
	}
	if _, sent := post["domainExpiryThreshold"]; sent {
		t.Error("an unset attribute must not be sent")
	}
	dv := post["contentValidation"].(map[string]any)
	if _, sent := dv["schema"]; sent {
		t.Errorf("unset nested attribute sent: %v", dv)
	}
}

// Clearing values with the contract's empty forms keeps the plan clean even
// though the API answers null.
func TestMonitor_ClearAndUpdateOnlyChanged(t *testing.T) {
	api, srv := newFakeAPI(t)
	base := `
resource "uptimewarden_monitor" "test" {
  name                    = "API"
  type                    = "http"
  url                     = "https://api.acme.com"
  external_id             = "%s"
  response_time_threshold = %d
  escalation_policy_id    = "%s"
  selected_webhook_ids    = %s
}
`
	step1 := providerBlock(srv.URL) + fmt.Sprintf(base, "api", 500, policyID, fmt.Sprintf("[%q, %q]", hookID, hookID2))
	step2 := providerBlock(srv.URL) + fmt.Sprintf(base, "", 0, "", "[]")

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{Config: step1},
			{
				Config: step2,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "external_id", ""),
					resource.TestCheckResourceAttr(res, "response_time_threshold", "0"),
					resource.TestCheckResourceAttr(res, "escalation_policy_id", ""),
					resource.TestCheckResourceAttr(res, "selected_webhook_ids.#", "0"),
				),
			},
			{Config: step2, PlanOnly: true},
		},
	})
	last := api.patch[len(api.patch)-1]
	want := map[string]bool{"externalId": true, "responseTimeThreshold": true, "escalationPolicyId": true, "selectedWebhookIds": true}
	for k := range last {
		if !want[k] {
			t.Errorf("PATCH sent unchanged field %q: %v", k, last)
		}
	}
	if len(last) != len(want) {
		t.Errorf("PATCH body: %v", last)
	}
}

func TestMonitor_Heartbeat(t *testing.T) {
	_, srv := newFakeAPI(t)
	config := providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name                    = "Nightly backup"
  type                    = "heartbeat"
  interval                = 86400
  heartbeat_grace_seconds = 1800
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestMatchResourceAttr(res, "url", regexp.MustCompile(`^heartbeat://hbtok`)),
					resource.TestMatchResourceAttr(res, "heartbeat_url", regexp.MustCompile(`/api/heartbeat/hbtok\w+/ping$`)),
					resource.TestMatchResourceAttr(res, "heartbeat_token", regexp.MustCompile(`^hbtok`)),
					resource.TestCheckResourceAttr(res, "heartbeat_interval_seconds", "86400"),
					resource.TestCheckResourceAttr(res, "heartbeat_grace_seconds", "1800"),
				),
			},
			{Config: config, PlanOnly: true},
		},
	})
}

func TestMonitor_URLRules(t *testing.T) {
	_, srv := newFakeAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name = "No URL"
  type = "http"
}
`,
				ExpectError: errRe(`url is required for http monitors`),
			},
			{
				Config: providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name = "Cron"
  type = "heartbeat"
  url  = "https://example.com"
}
`,
				ExpectError: errRe(`url is set by UptimeWarden for heartbeat monitors`),
			},
		},
	})
}

// http → tcp is an in-place PATCH; tcp → heartbeat replaces the monitor.
func TestMonitor_TypeChanges(t *testing.T) {
	api, srv := newFakeAPI(t)
	cfg := func(typ, url string) string {
		u := ""
		if url != "" {
			u = fmt.Sprintf("url = %q", url)
		}
		return providerBlock(srv.URL) + fmt.Sprintf(`
resource "uptimewarden_monitor" "test" {
  name = "Changing"
  type = %q
  %s
}
`, typ, u)
	}
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{Config: cfg("http", "https://db.example.com")},
			{
				Config: cfg("tcp", "tcp://db.example.com:5432"),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(res, plancheck.ResourceActionUpdate),
				}},
				Check: resource.TestCheckResourceAttr(res, "type", "tcp"),
			},
			{
				Config: cfg("heartbeat", ""),
				ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{
					plancheck.ExpectResourceAction(res, plancheck.ResourceActionDestroyBeforeCreate),
				}},
				Check: resource.TestCheckResourceAttr(res, "type", "heartbeat"),
			},
		},
	})
	if len(api.dels) < 1 {
		t.Error("changing to heartbeat must delete the old monitor")
	}
}

func TestMonitor_NotAppliedIsReported(t *testing.T) {
	api, srv := newFakeAPI(t)
	api.ignore["retryCount"] = true
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name        = "Retry"
  type        = "http"
  url         = "https://example.com"
  retry_count = 5
}
`,
				ExpectError: errRe(`retry_count: configured 5, API returned 1`),
			},
		},
	})
}

func TestMonitor_APIValidationErrorIsShown(t *testing.T) {
	_, srv := newFakeAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name   = "Dubai"
  type   = "http"
  url    = "https://example.com"
  region = "uaen"
}
`,
				ExpectError: errRe(`VALIDATION_ERROR: Invalid monitoring region (Valid regions: ae, gwc, sea)`),
			},
		},
	})
}

func TestMonitor_ConfigValidators(t *testing.T) {
	_, srv := newFakeAPI(t)
	cases := map[string]string{
		`response_time_threshold = 20`:         `response_time_threshold`,
		`timeout = 400`:                        `timeout`,
		`voice_mode = "some"`:                  `voice_mode`,
		`escalation_policy_id = "P1 on-call"`:  `escalation policy public id`,
		`selected_webhook_ids = ["slack"]`:     `webhook public id`,
		`verification_regions = ["sea","sea"]`: `verification_regions`,
		`heartbeat_interval_seconds = 30`:      `heartbeat_interval_seconds`,
		`type = "ftp"`:                         `value must be one of`,
	}
	for line, want := range cases {
		t.Run(line, func(t *testing.T) {
			typ := `type = "http"`
			if line[:4] == "type" {
				typ = ""
			}
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: testProviderFactories(),
				Steps: []resource.TestStep{{
					Config: providerBlock(srv.URL) + fmt.Sprintf(`
resource "uptimewarden_monitor" "test" {
  name = "v"
  url  = "https://example.com"
  %s
  %s
}
`, typ, line),
					ExpectError: errRe(want),
				}},
			})
		})
	}
}

func TestMonitor_Import(t *testing.T) {
	_, srv := newFakeAPI(t)
	config := providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name                = "Imported"
  type                = "http"
  url                 = "https://example.com"
  basic_auth_username = "u"
  basic_auth_password = "p"
  dns_config = {
    record_type      = "A"
    expected_answers = ["1.2.3.4"]
  }
}
`
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{Config: config},
			{Config: config, ResourceName: res, ImportState: true, ImportStateVerify: true},
		},
	})
}

// errRe matches text in Terraform's output, which wraps long lines.
func errRe(text string) *regexp.Regexp {
	parts := regexp.MustCompile(`\s+`).Split(regexp.QuoteMeta(text), -1)
	return regexp.MustCompile(strings.Join(parts, `\s+`))
}
