---
page_title: "UptimeWarden Provider"
description: |-
  The UptimeWarden provider manages UptimeWarden monitors.
---

# UptimeWarden Provider

The UptimeWarden provider manages your UptimeWarden monitors as code: the same
`terraform plan`, review process and state file as the rest of your
infrastructure.

It authenticates with an UptimeWarden API token (`uptw_...`). The API accepts
these tokens on its `/api/v1` monitor endpoints, so the provider offers the
`uptimewarden_monitor` resource and data source. Status pages, escalation
policies and on-call schedules are planned as resources once the API exposes
them to API tokens.

## Example Usage

```hcl
terraform {
  required_providers {
    uptimewarden = {
      source  = "uptimewarden/uptimewarden"
      version = "~> 0.1"
    }
  }
}

provider "uptimewarden" {}
```

Set your API token via environment variable:

```bash
export UW_API_TOKEN="uptw_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

Or inline (not recommended for production):

```hcl
provider "uptimewarden" {
  api_token = "uptw_xxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

The token needs the `monitors:read`, `monitors:create` and `monitors:delete`
permissions (the Monitor Manager group).

## Schema

### Optional

- `api_token` (String, Sensitive) — UptimeWarden API token. Required, either here or via the `UW_API_TOKEN` environment variable.
- `base_url` (String) — API base URL. Defaults to `https://app.uptimewarden.online`. Can also be set via `UW_API_URL` (the same variable the CLI and GitHub Actions read). `UW_BASE_URL` is still accepted as an alias; `UW_API_URL` wins when both are set.
