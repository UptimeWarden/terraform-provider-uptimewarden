---
page_title: "UptimeWarden Provider"
description: |-
  The UptimeWarden provider manages UptimeWarden monitors and reads regions, escalation policies and webhooks.
---

# UptimeWarden Provider

The UptimeWarden provider manages your UptimeWarden monitors as code: the same
`terraform plan`, review process and state file as the rest of your
infrastructure.

It authenticates with an UptimeWarden API token (`uptw_...`) on the `/api/v1`
API and offers:

- the `uptimewarden_monitor` resource, with every monitor setting the dashboard has;
- data sources `uptimewarden_monitor`, `uptimewarden_regions`,
  `uptimewarden_escalation_policy` and `uptimewarden_webhook` (by name).

Resources for status pages, escalation policies, webhooks and on-call
schedules are planned once the API accepts API tokens for writing them; until
then create those in the dashboard and reference policies and webhooks from
monitors.

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
