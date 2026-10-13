# Terraform Provider for UptimeWarden

The official Terraform provider for [UptimeWarden](https://uptimewarden.online) — an AI-driven uptime monitoring platform.

Manage your UptimeWarden monitors as code alongside the rest of your
infrastructure: every monitor setting the dashboard has — checks, regions and
consensus, thresholds, alert routing, escalation, webhooks, maintenance
windows and heartbeats.

The provider authenticates with an UptimeWarden API token (`uptw_...`) on the
`/api/v1` API. Escalation policies and webhooks are read (by name) through data
sources; resources for status pages, escalation policies, webhooks and on-call
schedules are planned once the API accepts API tokens for writing them.

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

data "uptimewarden_escalation_policy" "p1" {
  name = "P1 on-call"
}

resource "uptimewarden_monitor" "api" {
  name                  = "Production API"
  type                  = "http"
  url                   = "https://api.acme.com/health"
  interval              = 60
  timeout               = 10
  expected_status_codes = "200-299"
  escalation_policy_id  = data.uptimewarden_escalation_policy.p1.id
}

resource "uptimewarden_monitor" "backup" {
  name     = "Nightly backup"
  type     = "heartbeat"
  interval = 86400
}
```

The full attribute list is in [docs/resources/monitor.md](./docs/resources/monitor.md).

## Resources

| Resource | Description |
|---|---|
| `uptimewarden_monitor` | HTTP, TCP, UDP, SSL, WebSocket, gRPC, MCP, Ping, DNS, and Heartbeat monitors with all their settings |

### Regions

The `region` attribute takes one of these codes:

| Code | Region |
|---|---|
| `ae` | `ae` (Sydney) |
| `brs` | `brs` (São Paulo) |
| `gwc` | `gwc` (Sweden Central) |
| `sea` | `sea` (East Asia) |
| `wus3` | `wus3` (Phoenix) |

Choosing a region needs a plan that allows it (Professional or Enterprise).
Leave `region` unset to let UptimeWarden place the monitor (`gwc`). The
`uptimewarden_regions` data source returns the active codes and your plan's
allowance.

## Data Sources

| Data Source | Description |
|---|---|
| `data.uptimewarden_monitor` | Look up an existing monitor by public ID |
| `data.uptimewarden_regions` | Active regions and your plan's region allowance |
| `data.uptimewarden_escalation_policy` | Look up an escalation policy by name |
| `data.uptimewarden_webhook` | Look up a webhook configuration by name |

## Provider Configuration

| Attribute | Description | Environment variable |
|---|---|---|
| `api_token` | UptimeWarden API token | `UW_API_TOKEN` |
| `base_url` | API base URL (default: `https://app.uptimewarden.online`) | `UW_API_URL` (alias: `UW_BASE_URL`) |

`UW_API_URL` is the same variable the CLI and the GitHub Actions read.

## Import

Monitors can be imported by their public ID (ULID):

```bash
terraform import uptimewarden_monitor.api 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
```

## Development

```bash
# Build
make build

# Run unit tests (no live API required)
make test

# Run acceptance tests (requires UW_API_TOKEN for a live account; UW_API_URL optional)
make testacc
```

To use a local build, add a `dev_overrides` block for
`uptimewarden/uptimewarden` to your `~/.terraformrc` that points at your
`$GOPATH/bin` (after `make install`).

Changes are listed in [CHANGELOG.md](./CHANGELOG.md). Releases are described
in [RELEASING.md](./RELEASING.md).

## License

Apache 2.0 — see [LICENSE](./LICENSE).
