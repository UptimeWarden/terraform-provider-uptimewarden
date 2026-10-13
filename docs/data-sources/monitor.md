---
page_title: "uptimewarden_monitor Data Source"
description: |-
  Reads an existing UptimeWarden monitor by its public ID.
---

# data.uptimewarden_monitor

Reads an existing monitor by public ID, with every field the API returns. Use
it to reference a monitor managed outside this configuration: one created in
the dashboard, with the CLI, or in another Terraform workspace.

## Example Usage

```hcl
data "uptimewarden_monitor" "homepage" {
  id = "01HWXXXXXXXXXXXXXXXXXXXXXXX"
}

output "homepage_url" {
  value = data.uptimewarden_monitor.homepage.url
}
```

## Argument Reference

- `id` (Required) — Public ID of the monitor to look up.

## Attribute Reference

Every attribute of the [`uptimewarden_monitor` resource](../resources/monitor.md),
read-only, plus:

- `last_checked` (String, Computed) — Time of the last check. API field: `lastChecked`.
- `heartbeat_last_ping_at` (String, Computed) — Heartbeat monitors: time of the last ping. API field: `heartbeatLastPingAt`.
