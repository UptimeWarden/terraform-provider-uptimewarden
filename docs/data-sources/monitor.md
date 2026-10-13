---
page_title: "uptimewarden_monitor Data Source"
description: |-
  Reads an existing UptimeWarden monitor by its public ID.
---

# data.uptimewarden_monitor

Reads an existing monitor by public ID. Use this data source to reference a monitor managed outside of Terraform — for example, one created via the UI, the CLI, or another Terraform workspace.

## Example Usage

```hcl
data "uptimewarden_monitor" "homepage" {
  id = "01HWXXXXXXXXXXXXXXXXXXXXXXXXX"
}

output "homepage_url" {
  value = data.uptimewarden_monitor.homepage.url
}
```

## Argument Reference

- `id` (Required) — Public ID of the monitor to look up.

## Attribute Reference

- `name` — Monitor name.
- `type` — Protocol type.
- `url` — Target URL or address.
- `interval` — Check interval in seconds.
- `timeout` — Request timeout in seconds.
- `region` — Primary check region.
- `is_active` — Whether the monitor is actively checking.
