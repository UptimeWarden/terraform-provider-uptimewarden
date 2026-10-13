---
page_title: "uptimewarden_webhook Data Source"
description: |-
  Looks up one of your webhook configurations by name.
---

# data.uptimewarden_webhook

Looks up one of your webhook configurations by name (`GET /api/v1/webhooks`)
so a monitor can reference it in `selected_webhook_ids`. Webhooks are created
in the dashboard; a resource for them is planned once the API accepts API
tokens for writing them.

The name must match exactly one webhook (case-sensitive); otherwise the read
fails and lists the names that exist.

## Example Usage

```hcl
data "uptimewarden_webhook" "slack" {
  name = "Slack #ops"
}

resource "uptimewarden_monitor" "api" {
  name                 = "API"
  type                 = "http"
  url                  = "https://api.acme.com/health"
  selected_webhook_ids = [data.uptimewarden_webhook.slack.id]
}
```

## Argument Reference

- `name` (Required) — Webhook name.

## Attribute Reference

- `id` (String) — Public id (ULID).
- `description` (String) — May be null.
- `message_format` (String) — May be null.
- `is_active` (Boolean)
- `enabled_alert_types` (List of String)
- `enabled_monitor_types` (List of String)
- `created_at` (String)
