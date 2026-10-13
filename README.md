# Terraform Provider for UptimeWarden

The official Terraform provider for [UptimeWarden](https://uptimewarden.online) — an AI-driven uptime monitoring platform.

Manage your UptimeWarden monitors as code alongside the rest of your infrastructure.

The provider authenticates with an UptimeWarden API token (`uptw_...`). The API
accepts these tokens on its `/api/v1` monitor endpoints, so today the provider
covers monitors. Status pages, escalation policies and on-call schedules are
planned as resources once the API exposes them to API tokens.

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
  timeout  = 10
}
```

## Resources

| Resource | Description |
|---|---|
| `uptimewarden_monitor` | HTTP, TCP, UDP, SSL, WebSocket, gRPC, MCP, Ping, DNS, and Heartbeat monitors |

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
Leave `region` unset to let UptimeWarden place the monitor (`gwc`).

## Data Sources

| Data Source | Description |
|---|---|
| `data.uptimewarden_monitor` | Look up an existing monitor by public ID |

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
