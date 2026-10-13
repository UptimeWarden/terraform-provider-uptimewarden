---
page_title: "uptimewarden_webhook_config Resource"
description: |-
  Manages an outbound webhook notification endpoint.
---

# uptimewarden_webhook_config

Manages an outbound webhook configuration — an HTTP endpoint that receives alert notifications when monitors go down or recover.

## Example Usage

```hcl
resource "uptimewarden_webhook_config" "pagerduty" {
  type_id     = 1
  name        = "PagerDuty"
  url         = "https://events.pagerduty.com/integration/XXXXX/enqueue"
  http_method = "POST"
}

resource "uptimewarden_webhook_config" "slack" {
  type_id     = 1
  name        = "Slack #alerts"
  url         = var.slack_webhook_url
  http_method = "POST"
  retry_count = 3
  retry_delay = 30
}
```

## Argument Reference

- `type_id` (Required) — Webhook type ID. Use `1` for a generic HTTP webhook.
- `name` (Required) — Display name.
- `url` (Required) — Destination URL (http or https).
- `http_method` (Optional, Default: `POST`) — HTTP method. One of: `GET`, `POST`, `PUT`, `PATCH`, `DELETE`.
- `description` (Optional) — Optional description.
- `auth_method` (Optional) — Authentication method (e.g. `bearer`, `basic`).
- `timeout` (Optional, Default: `30`) — Request timeout in seconds.
- `retry_count` (Optional, Default: `3`) — Retry attempts on failure.
- `retry_delay` (Optional, Default: `60`) — Seconds between retries.
- `is_active` (Optional, Default: `true`) — Whether this config is active.

## Attribute Reference

- `id` — Public ID (ULID).

## Import

```bash
terraform import uptimewarden_webhook_config.pd 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
```
