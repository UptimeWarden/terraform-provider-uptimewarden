---
page_title: "uptimewarden_monitor Resource"
description: |-
  Manages an UptimeWarden monitor: checks, regions, thresholds and alert routing.
---

# uptimewarden_monitor

Manages an UptimeWarden monitor through the `/api/v1/monitors` API with an API
token: what it checks (HTTP/HTTPS, TCP, UDP, SSL, WebSocket, gRPC, MCP,
Ping/ICMP, DNS or Heartbeat), from which regions, its thresholds and who is
alerted.

Each attribute is the snake_case form of the API field with the same meaning
(`expected_status_codes` is `expectedStatusCodes`). Every request goes through
the same validation, defaults and plan limits as the dashboard; a refused
request fails the apply with the API's own message (for plan limits, your
plan's sentence).

## Example Usage

```hcl
data "uptimewarden_escalation_policy" "p1" {
  name = "P1 on-call"
}

data "uptimewarden_webhook" "slack" {
  name = "Slack #ops"
}

resource "uptimewarden_monitor" "checkout" {
  name        = "Checkout API"
  type        = "http"
  url         = "https://api.acme.com/checkout/health"
  interval    = 60
  timeout     = 10
  external_id = "checkout-api"
  tags        = ["prod", "payments"]

  # Where it is checked from
  region                 = "gwc"
  verification_regions   = ["sea"]
  min_confirming_regions = 2

  # HTTP settings
  method                = "GET"
  custom_headers        = { "X-Probe" = "uptimewarden" }
  expected_status_codes = "200-299"
  expected_body         = "ok"
  follow_redirects      = true
  verify_ssl            = true

  # Thresholds
  response_time_threshold = 800
  ssl_expiry_days         = 14
  ssl_alerts = {
    thirty_days  = false
    fifteen_days = true
    seven_days   = true
    one_day      = true
  }

  # Alert routing
  alert_types = ["down", "recovery"]
  notifications = {
    email = { down = true, recovery = true }
  }
  email_addresses      = "oncall@acme.com"
  escalation_policy_id = data.uptimewarden_escalation_policy.p1.id
  selected_webhook_ids = [data.uptimewarden_webhook.slack.id]

  maintenance_windows = [{
    type     = "recurring"
    day      = "sunday"
    start    = "02:00"
    end      = "03:00"
    timezone = "Europe/Istanbul"
  }]
}

resource "uptimewarden_monitor" "dns" {
  name = "MX records"
  type = "dns"
  url  = "acme.com"
  dns_config = {
    record_type      = "MX"
    expected_answers = ["mx1.acme.com"]
  }
}

resource "uptimewarden_monitor" "backup" {
  name                    = "Nightly backup"
  type                    = "heartbeat"
  interval                = 86400
  heartbeat_grace_seconds = 1800
}

output "backup_ping_url" {
  value     = uptimewarden_monitor.backup.heartbeat_url
  sensitive = true
}
```

## How attributes behave

- **Optional attributes left out** keep whatever UptimeWarden holds: its
  default on create (for example `interval` 300, `timeout` 30, `voice_mode`
  `sequential`), and the stored value later. Removing an attribute from the
  configuration therefore does not reset it; set the value you want instead.
- **Clearing:** `escalation_policy_id = ""` detaches the policy,
  `selected_webhook_ids = []` detaches all webhooks,
  `response_time_threshold = 0` turns the slow-response alert off and
  `external_id = ""` clears the handle.
- **Updates** send only the attributes whose value changed.
- **Type changes:** between two non-heartbeat types the monitor is updated in
  place (its history stays). Changing to or from `heartbeat` destroys the
  monitor and creates a new one.
- **Heartbeat monitors:** do not set `url` (UptimeWarden sets it to
  `heartbeat://<token>`, so it contains the token, like `heartbeat_token` and
  `heartbeat_url`). Ping `heartbeat_url` when the job succeeds; replace
  `/ping` with `/fail` to report a failure right away.
- **Secrets:** `basic_auth_password`, `bearer_token`, `heartbeat_token` and
  `heartbeat_url` are sensitive. The API returns them only to callers who can
  edit the monitor; when it does not, the value already in state is kept.
- **Groups** are set by the API on update only, so on create the provider sets
  them with one update right after.
- If the API accepts a request but returns a different value for an attribute
  you configured, the apply fails with an error that names the attribute and
  both values; state records what the API returned.
- Nested objects (`content_validation`, `ssl_alerts`, `dns_config`,
  `maintenance_windows`) are attributes: write them with `=`.

## Argument Reference

- `public_id` (String, Computed) — Public ID of the monitor (ULID). Same value as `id`. API field: `publicId`.
- `name` (String, Required) — Human-readable name. API field: `name`.
- `type` (String, Required) — Protocol type. One of: `http`, `tcp`, `udp`, `ssl`, `websocket`, `mcp`, `grpc`, `ping`, `dns`, `heartbeat`. Changing between two non-heartbeat types updates the monitor in place; changing to or from `heartbeat` destroys it and creates a new one. API field: `type`.
- `url` (String, Optional) — Target URL or address. Required for every type except `heartbeat`, where UptimeWarden sets it to `heartbeat://<token>` (do not set it). API field: `url`.
- `description` (String, Optional) — Free-text description. API field: `description`.
- `interval` (Number, Optional) — Check interval in seconds. At least 30 and at least your plan's minimum. API default: 300. API field: `interval`.
- `timeout` (Number, Optional) — Request timeout in seconds (5–300). API default: 30. API field: `timeout`.
- `is_active` (Boolean, Optional) — Whether the monitor is checking. API default: `true`. API field: `isActive`.
- `tags` (Set of String, Optional) — Tags. API field: `tags`.
- `groups` (Set of String, Optional) — Monitor groups (by name). The API sets groups on update only, so on create the provider sets them right after. API field: `groups`.
- `external_id` (String, Optional) — Your stable handle for the monitor (at most 255 characters). `""` clears it. API field: `externalId`.
- `managed_by_source` (String, Computed) — Who created the monitor: `terraform` for monitors this provider created, `manual`, `cli`, … API field: `managedBySource`.
- `region` (String, Optional) — Home region code: `ae` (Sydney), `brs` (São Paulo), `gwc` (Sweden Central), `sea` (East Asia), `wus3` (Phoenix). Choosing a region needs region selection on your plan; when unset the API places the monitor (`gwc`). The `uptimewarden_regions` data source lists the active codes and your plan's allowance. API field: `region`.
- `verification_regions` (List of String, Optional) — Regions that recheck a failure before it is alerted: active codes, no repeats, not the home region, at most your plan's verification regions minus one. When unset the API picks them from your plan. API field: `verificationRegions`.
- `min_confirming_regions` (Number, Optional) — How many regions must see the failure to confirm it. A value other than your plan's default needs a plan that allows it. API field: `minConfirmingRegions`.
- `agent_id` (String, Optional) — Run the checks from one of your own agents (its id). `""` or `server` means UptimeWarden's regions. API field: `agentId`.
- `method` (String, Optional) — HTTP method. API field: `method`.
- `request_body` (String, Optional) — HTTP request body. API field: `requestBody`.
- `expected_status_codes` (String, Optional) — Accepted HTTP status codes, e.g. `200-299` or `200,201`. API field: `expectedStatusCodes`.
- `expected_body` (String, Optional) — Text the response body must contain. API field: `expectedBody`.
- `custom_headers` (Map of String, Optional) — Extra request headers. API field: `customHeaders`.
- `content_validation` (Object, Optional) — Response content validation. API field: `contentValidation`.
  - `enabled` (Boolean, Optional) — Whether content validation runs. API field: `enabled`.
  - `type` (String, Optional) — `json`, `xml`, `text` or `regex`. API field: `type`.
  - `schema` (String, Optional) — JSON Schema or XML Schema. API field: `schema`.
  - `expected_content` (String, Optional) — Expected text or regex. API field: `expectedContent`.
  - `case_sensitive` (Boolean, Optional) — Case-sensitive text/regex match. API field: `caseSensitive`.
- `follow_redirects` (Boolean, Optional) — Follow HTTP redirects. API field: `followRedirects`.
- `verify_ssl` (Boolean, Optional) — Verify the TLS certificate. API field: `verifySsl`.
- `enable_ssl` (Boolean, Optional) — Use TLS (TCP and similar checks). API field: `enableSsl`.
- `basic_auth_username` (String, Optional) — HTTP basic auth user name. API field: `basicAuthUsername`.
- `basic_auth_password` (String, Optional, Sensitive) — HTTP basic auth password. API field: `basicAuthPassword`.
- `bearer_token` (String, Optional, Sensitive) — Bearer token sent in the Authorization header. API field: `bearerToken`.
- `hostname` (String, Optional) — Host name (TCP, UDP, ping and similar checks). API field: `hostname`.
- `port` (Number, Optional) — Port (1–65535). API field: `port`.
- `grpc_proto_file` (String, Optional) — gRPC proto file. API field: `grpcProtoFile`.
- `grpc_method` (String, Optional) — gRPC method. API field: `grpcMethod`.
- `dns_config` (Object, Optional) — DNS check settings (type `dns`). API field: `dnsConfig`.
  - `record_type` (String, Optional) — `A`, `AAAA`, `CNAME`, `MX`, `NS` or `TXT`. API field: `recordType`.
  - `resolver` (String, Optional) — Resolver to query; the check's default resolver when unset. API field: `resolver`.
  - `expected_answers` (List of String, Optional) — Values that must all appear in the answer. API field: `expectedAnswers`.
- `retry_count` (Number, Optional) — Retries before a failure counts. API field: `retryCount`.
- `response_time_threshold` (Number, Optional) — Slow-response alert threshold in milliseconds (50–5000). `0` turns it off. API field: `responseTimeThreshold`.
- `ssl_expiry_days` (Number, Optional) — Days before certificate expiry to alert. API default: 30. API field: `sslExpiryDays`.
- `ssl_alerts` (Object, Optional) — Which certificate-expiry reminders to send. API field: `sslAlerts`.
  - `thirty_days` (Boolean, Optional) — 30 days before expiry. API field: `thirtyDays`.
  - `fifteen_days` (Boolean, Optional) — 15 days before expiry. API field: `fifteenDays`.
  - `seven_days` (Boolean, Optional) — 7 days before expiry. API field: `sevenDays`.
  - `one_day` (Boolean, Optional) — 1 day before expiry. API field: `oneDay`.
- `domain_monitoring_enabled` (Boolean, Optional) — Watch the domain's registration expiry. A plan feature with a count limit. API field: `domainMonitoringEnabled`.
- `domain_expiry_threshold` (Number, Optional) — Days before domain expiry to alert. API field: `domainExpiryThreshold`.
- `security_analysis_enabled` (Boolean, Optional) — Run security analysis. A plan feature. API field: `securityAnalysisEnabled`.
- `security_analysis_mode` (String, Optional) — Security analysis mode. API field: `securityAnalysisMode`.
- `alert_types` (Set of String, Optional) — Alert types to send, e.g. `down`, `recovery`. API field: `alertTypes`.
- `notifications` (Map of Map of Boolean, Optional) — Channel × alert type switches, e.g. `{ email = { down = true, recovery = true } }`. API field: `notifications`.
- `email_addresses` (String, Optional) — Comma-separated e-mail recipients. API field: `emailAddresses`.
- `phone_numbers` (String, Optional) — Comma-separated SMS recipients. API field: `phoneNumbers`.
- `whatsapp_numbers` (String, Optional) — Comma-separated WhatsApp recipients. API field: `whatsappNumbers`.
- `voice_numbers` (String, Optional) — Comma-separated voice-call recipients. API field: `voiceNumbers`.
- `voice_mode` (String, Optional) — `sequential` (one number at a time, stops when someone answers) or `all`. API default: `sequential`. API field: `voiceMode`.
- `escalation_policy_id` (String, Optional) — Public id of one of your escalation policies (see the `uptimewarden_escalation_policy` data source). `""` detaches it. On-call escalation is a plan feature. API field: `escalationPolicyId`.
- `selected_webhook_ids` (Set of String, Optional) — Public ids of your webhooks (see the `uptimewarden_webhook` data source). `[]` detaches all. API field: `selectedWebhookIds`.
- `heartbeat_interval_seconds` (Number, Optional) — Heartbeat monitors: expected seconds between pings (60–2592000). Defaults to `interval`. API field: `heartbeatIntervalSeconds`.
- `heartbeat_grace_seconds` (Number, Optional) — Heartbeat monitors: slack after a missed ping before alerting (0–2592000). API default: 60. API field: `heartbeatGraceSeconds`.
- `heartbeat_token` (String, Computed, Sensitive) — Heartbeat monitors: the ping token. API field: `heartbeatToken`.
- `heartbeat_url` (String, Computed, Sensitive) — Heartbeat monitors: the success ping URL. Replace `/ping` with `/fail` to report a failure. API field: `heartbeatUrl`.
- `maintenance_windows` (List of Object, Optional) — Maintenance windows. Windows opened by deploys (GitHub Actions, CLI) show up here too; a one-off window that has ended is dropped by the API. API field: `maintenanceWindows`.
  - `id` (String, Optional) — Window id (set by the deploy endpoints). API field: `id`.
  - `type` (String, Optional) — `recurring` or `once`. API field: `type`.
  - `day` (String, Optional) — Recurring windows: weekday, e.g. `monday`. API field: `day`.
  - `start` (String, Required) — Start: `HH:MM` (recurring) or ISO 8601 (once). API field: `start`.
  - `end` (String, Required) — End: `HH:MM` (recurring) or ISO 8601 (once). API field: `end`.
  - `timezone` (String, Optional) — IANA time zone, e.g. `Europe/Istanbul`. API field: `timezone`.
  - `source` (String, Optional) — Who opened the window. API field: `source`.
  - `reason` (String, Optional) — Why. API field: `reason`.
- `created_at` (String, Computed) — Creation time (RFC 3339). API field: `createdAt`.

`id` (String) is the monitor's public ID, the same value as `public_id`.

## Import

```bash
terraform import uptimewarden_monitor.checkout 01J9ZP4M3K8Q5R6S7T8V9W0X1Y
```
