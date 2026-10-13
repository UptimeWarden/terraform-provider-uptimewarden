---
page_title: "uptimewarden_monitor Resource"
description: |-
  Manages an UptimeWarden monitor.
---

# uptimewarden_monitor

Manages an UptimeWarden monitor. Supports all 10 protocol types: HTTP/HTTPS, TCP, UDP, SSL, WebSocket, gRPC, MCP (AI/LLM endpoints), Ping/ICMP, DNS, and Heartbeat (cron/job inverse monitoring).

## Example Usage

```hcl
resource "uptimewarden_monitor" "api" {
  name     = "Production API"
  type     = "http"
  url      = "https://api.acme.com/health"
  interval = 60
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

resource "uptimewarden_monitor" "backup" {
  name     = "Nightly backup job"
  type     = "heartbeat"
  url      = "heartbeat://nightly-backup"
  interval = 86400
}
```

## Argument Reference

- `name` (Required) — Human-readable name.
- `type` (Required, Forces replacement) — Protocol type. One of: `http`, `tcp`, `udp`, `ssl`, `websocket`, `grpc`, `mcp`, `ping`, `dns`, `heartbeat`.
- `url` (Required) — Target URL or address.
- `interval` (Optional, Default: `300`) — Check interval in seconds (30–86400).
- `timeout` (Optional, Default: `30`) — Request timeout in seconds (5–300).
- `region` (Optional, Default: `gwc`) — Primary check region. One of: `gwc` (Frankfurt), `wus3` (Phoenix), `sea` (Singapore), `brs` (São Paulo), `ae` (Dubai), `uaen` (UAE North).
- `is_active` (Optional, Default: `true`) — Whether the monitor is actively checking.

## Attribute Reference

- `id` — Public ID of the monitor (ULID).

## Import

```bash
terraform import uptimewarden_monitor.api 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
```
