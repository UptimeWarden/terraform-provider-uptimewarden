package provider_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestDataSource_Regions(t *testing.T) {
	_, srv := newFakeAPI(t)
	const ds = "data.uptimewarden_regions.all"
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{{
			Config: providerBlock(srv.URL) + `data "uptimewarden_regions" "all" {}`,
			Check: resource.ComposeAggregateTestCheckFunc(
				resource.TestCheckResourceAttr(ds, "codes.#", "3"),
				resource.TestCheckResourceAttr(ds, "codes.1", "gwc"),
				resource.TestCheckResourceAttr(ds, "default", "gwc"),
				resource.TestCheckResourceAttr(ds, "regions.1.name", "Sweden Central"),
				resource.TestCheckNoResourceAttr(ds, "regions.2.description"),
				resource.TestCheckResourceAttr(ds, "plan.region_selection", "true"),
				resource.TestCheckResourceAttr(ds, "plan.max_verification_regions", "-1"),
				resource.TestCheckResourceAttr(ds, "plan.default_confirming_regions", "3"),
			),
		}},
	})
}

func TestDataSource_EscalationPolicyAndWebhookByName(t *testing.T) {
	_, srv := newFakeAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock(srv.URL) + `
data "uptimewarden_escalation_policy" "p1" { name = "P1 on-call" }
data "uptimewarden_webhook" "slack" { name = "Slack #ops" }

resource "uptimewarden_monitor" "test" {
  name                 = "Routed"
  type                 = "http"
  url                  = "https://example.com"
  escalation_policy_id = data.uptimewarden_escalation_policy.p1.id
  selected_webhook_ids = [data.uptimewarden_webhook.slack.id]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.uptimewarden_escalation_policy.p1", "id", policyID),
					resource.TestCheckResourceAttr("data.uptimewarden_escalation_policy.p1", "step_count", "2"),
					resource.TestCheckResourceAttr("data.uptimewarden_webhook.slack", "id", hookID),
					resource.TestCheckResourceAttr("data.uptimewarden_webhook.slack", "message_format", "slack"),
					resource.TestCheckNoResourceAttr("data.uptimewarden_webhook.slack", "description"),
					resource.TestCheckResourceAttr(res, "escalation_policy_id", policyID),
					resource.TestCheckTypeSetElemAttr(res, "selected_webhook_ids.*", hookID),
				),
			},
			{
				Config:      providerBlock(srv.URL) + `data "uptimewarden_webhook" "x" { name = "Teams" }`,
				ExpectError: errRe(`No webhook named "Teams". Available: "Slack #ops", "PagerDuty".`),
			},
			{
				Config:      providerBlock(srv.URL) + `data "uptimewarden_escalation_policy" "x" { name = "p1 on-call" }`,
				ExpectError: errRe(`No escalation policy named "p1 on-call".`),
			},
		},
	})
}

func TestDataSource_Monitor(t *testing.T) {
	_, srv := newFakeAPI(t)
	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name = "Cron"
  type = "heartbeat"
}
`,
			},
			{
				Config: providerBlock(srv.URL) + `
resource "uptimewarden_monitor" "test" {
  name = "Cron"
  type = "heartbeat"
}
data "uptimewarden_monitor" "read" {
  id = uptimewarden_monitor.test.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.uptimewarden_monitor.read", "heartbeat_url", res, "heartbeat_url"),
					resource.TestCheckResourceAttr("data.uptimewarden_monitor.read", "ssl_alerts.one_day", "true"),
					resource.TestCheckResourceAttr("data.uptimewarden_monitor.read", "notifications.email.down", "true"),
					resource.TestCheckNoResourceAttr("data.uptimewarden_monitor.read", "last_checked"),
				),
			},
		},
	})
}
