---
page_title: "uptimewarden_monitor Resource"
description: |-
  Manages an UptimeWarden monitor.
---

# uptimewarden_monitor

Manages an UptimeWarden monitor through the `/api/v1/monitors` API with an API
token. Supports the protocol types HTTP/HTTPS, TCP, UDP, SSL, WebSocket, gRPC,
MCP, Ping/ICMP, DNS and Heartbeat.

## Example Usage

```hcl
resource "uptimewarden_monitor" "api" {
  name     = "Production API"
  type     = "http"
  url      = "https://api.acme.com/health"
  interval = 60
  timeout  = 10
  region   = "gwc"
}

resource "uptimewarden_monitor" "db" {
  name     = "Database port"
  type     = "tcp"
  url      = "tcp://db.example.com:5432"
  interval = 60
}

resource "uptimewarden_monitor" "grpc" {
  name     = "Auth service"
  type     = "grpc"
  url      = "grpc://auth.example.com:50051"
  interval = 60
}
```

## Argument Reference

- `name` (Required) — Human-readable name.
- `type` (Required, Forces replacement) — Protocol type. One of: `http`, `tcp`, `udp`, `ssl`, `websocket`, `grpc`, `mcp`, `ping`, `dns`, `heartbeat`. The API cannot change the type of an existing monitor, so a change destroys the monitor and creates a new one (its history stays with the old monitor).
- `url` (Required) — Target URL or address.
- `interval` (Optional, Default: `300`) — Check interval in seconds (30–86400). Your plan sets the shortest interval it allows.
- `timeout` (Optional, Computed) — Request timeout in seconds (5–300). When unset, the API default (`30`) applies and is read back into state. The API applies a timeout on update but not yet on create, so a configured value other than `30` is set with an update right after the create.
- `region` (Optional, Computed) — Home region that runs the checks. One of: `ae` (Sydney), `brs` (São Paulo), `gwc` (Sweden Central), `sea` (East Asia), `wus3` (Phoenix). Choosing a region needs a plan that allows it (Professional or Enterprise). When unset, UptimeWarden places the monitor automatically (`gwc`) and the attribute reads back what the API reports.
- `is_active` (Optional, Default: `true`) — Whether the monitor is actively checking.

State always records what the API reports. If the API accepts a request but
returns a different value for an attribute you configured, the apply fails with
an error that names the attribute and both values.

## Attribute Reference

- `id` — Public ID of the monitor (ULID).

## Import

```bash
terraform import uptimewarden_monitor.api 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
```
