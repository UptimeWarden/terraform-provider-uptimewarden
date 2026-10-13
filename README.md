# Terraform Provider for UptimeWarden

The official Terraform provider for [UptimeWarden](https://uptimewarden.online) — an AI-driven uptime monitoring platform.

Manage monitors, status pages, escalation policies, on-call schedules, webhook configs, and API tokens as code alongside the rest of your infrastructure.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://go.dev/doc/install) >= 1.25 (for building from source)
- An UptimeWarden account with an API token (`UW_API_TOKEN`)

## Usage

> The first release to the Terraform Registry is coming soon. Until then,
> build the provider from source (see [Development](#development)).

```hcl
terraform {
  required_providers {
    uptimewarden = {
      source  = "uptimewarden/uptimewarden"
      version = "~> 0.1"
    }
  }
}

provider "uptimewarden" {
  # Token can also be set via UW_API_TOKEN environment variable.
  # api_token = var.uw_api_token
}

resource "uptimewarden_monitor" "api" {
  name     = "Production API"
  type     = "http"
  url      = "https://api.acme.com/health"
  interval = 60
  region   = "gwc"
}

resource "uptimewarden_status_page" "public" {
  name  = "Acme Status"
  title = "Acme System Status"
  slug  = "acme"
}
```

## Resources

| Resource | Description |
|---|---|
| `uptimewarden_monitor` | HTTP, TCP, UDP, SSL, WebSocket, gRPC, MCP, Ping, DNS, and Heartbeat monitors |
| `uptimewarden_webhook_config` | Outbound webhook endpoints for alert delivery |
| `uptimewarden_status_page` | Public or private status pages with optional custom domain |
| `uptimewarden_escalation_policy` | Ordered on-call escalation chains |
| `uptimewarden_on_call_schedule` | Weekly on-call rotation schedules |
| `uptimewarden_api_token` | API tokens with granular permission scopes |

## Data Sources

| Data Source | Description |
|---|---|
| `data.uptimewarden_monitor` | Look up an existing monitor by public ID |

## Provider Configuration

| Attribute | Description | Environment variable |
|---|---|---|
| `api_token` | UptimeWarden API token | `UW_API_TOKEN` |
| `base_url` | API base URL (default: `https://app.uptimewarden.online`) | `UW_BASE_URL` |

## Import

Most resources support `terraform import` by their public ID (ULID):

```bash
terraform import uptimewarden_monitor.api 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
terraform import uptimewarden_status_page.public 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
terraform import uptimewarden_webhook_config.pd 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
terraform import uptimewarden_escalation_policy.p1 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
terraform import uptimewarden_on_call_schedule.primary 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
```

> **Note:** `uptimewarden_api_token` does not support import because the plain token value is only available at creation time.

## Development

```bash
# Build
make build

# Run unit tests (no live API required)
make test

# Run acceptance tests (requires UW_API_TOKEN pointing at a live account)
make testacc
```

To use a local build, add a `dev_overrides` block for
`uptimewarden/uptimewarden` to your `~/.terraformrc` that points at your
`$GOPATH/bin` (after `make install`).

Releases are described in [RELEASING.md](./RELEASING.md).

## License

Apache 2.0 — see [LICENSE](./LICENSE).
