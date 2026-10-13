---
page_title: "uptimewarden_regions Data Source"
description: |-
  The active monitoring regions and your plan's region allowance.
---

# data.uptimewarden_regions

Lists the active monitoring regions (`GET /api/v1/regions`) and what your plan
allows: whether you can choose a monitor's region, how many regions a monitor
may vote across, and the default consensus.

## Example Usage

```hcl
data "uptimewarden_regions" "all" {}

output "region_codes" {
  value = data.uptimewarden_regions.all.codes
}

resource "uptimewarden_monitor" "api" {
  name = "API"
  type = "http"
  url  = "https://api.acme.com/health"

  # Only choose a region when the plan allows it.
  region = data.uptimewarden_regions.all.plan.region_selection ? "sea" : null
}
```

## Attribute Reference

- `codes` (List of String) — Active region codes, e.g. `["ae", "brs", "gwc", "sea", "wus3"]`.
- `default` (String) — The region a monitor gets when none is chosen.
- `regions` (List of Object) — Active regions:
  - `code` (String) — e.g. `gwc`.
  - `name` (String) — e.g. `Sweden Central`.
  - `description` (String) — Area, e.g. `Europe`. May be null.
  - `is_default` (Boolean)
- `plan` (Object) — Your plan's allowance; `-1` means unlimited. Null when the API does not report it.
  - `region_selection` (Boolean) — Whether you can choose a monitor's `region`.
  - `max_regions` (Number) — Regions your monitors may use in total.
  - `max_verification_regions` (Number) — Regions (home plus verification) one monitor may vote across, so `verification_regions` holds at most this minus one.
  - `default_confirming_regions` (Number) — The default `min_confirming_regions`.
  - `allow_consensus_override` (Boolean) — Whether `min_confirming_regions` may differ from the default.
