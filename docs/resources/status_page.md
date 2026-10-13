---
page_title: "uptimewarden_status_page Resource"
description: |-
  Manages a public or private status page.
---

# uptimewarden_status_page

Manages a public or private UptimeWarden status page. Custom domain support requires Professional or Enterprise plan.

## Example Usage

```hcl
resource "uptimewarden_status_page" "public" {
  name  = "Acme Status"
  title = "Acme System Status"
  slug  = "acme"
}

resource "uptimewarden_status_page" "custom_domain" {
  name               = "Acme Status"
  title              = "Acme System Status"
  slug               = "acme"
  domain             = "status.acme.com"
  allow_subscriptions = true
  hide_powered_by    = true
}
```

## Argument Reference

- `name` (Required) — Internal name (not shown publicly).
- `title` (Required) — Public-facing title displayed on the page.
- `slug` (Required) — URL slug. Must be unique.
- `description` (Optional) — Optional description text.
- `domain` (Optional) — Custom domain (e.g. `status.acme.com`). Requires Professional+.
- `logo_url` (Optional) — Logo image URL.
- `favicon_url` (Optional) — Favicon URL.
- `is_public` (Optional, Default: `true`) — Whether the page is publicly accessible.
- `allow_subscriptions` (Optional, Default: `true`) — Let visitors subscribe to email updates.
- `show_history` (Optional, Default: `true`) — Show incident history.
- `history_days` (Optional, Default: `90`) — Days of history to display.
- `show_response_time` (Optional, Default: `true`) — Show response time chart.
- `show_uptime` (Optional, Default: `true`) — Show uptime percentage.
- `hide_powered_by` (Optional, Default: `false`) — Hide the "Powered by UptimeWarden" footer.

## Attribute Reference

- `id` — Public ID (ULID).

## Import

```bash
terraform import uptimewarden_status_page.public 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
```
