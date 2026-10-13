---
page_title: "UptimeWarden Provider"
description: |-
  The UptimeWarden provider manages monitors, status pages, escalation policies, on-call schedules, webhook configs, and API tokens.
---

# UptimeWarden Provider

The UptimeWarden provider lets you manage your monitoring infrastructure as code — same `terraform plan`, same review process, same state file as the rest of your infrastructure.

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

## Schema

### Required

- `api_token` (String, Sensitive) — UptimeWarden API token. Can also be set via the `UW_API_TOKEN` environment variable.

### Optional

- `base_url` (String) — API base URL. Defaults to `https://app.uptimewarden.online`. Can also be set via `UW_BASE_URL`.
