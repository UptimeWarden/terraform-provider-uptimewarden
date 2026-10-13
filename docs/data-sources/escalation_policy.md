---
page_title: "uptimewarden_escalation_policy Data Source"
description: |-
  Looks up one of your escalation policies by name.
---

# data.uptimewarden_escalation_policy

Looks up one of your escalation policies by name (`GET /api/v1/escalation-policies`)
so a monitor can reference it by public id. Policies are created in the
dashboard; a resource for them is planned once the API accepts API tokens for
writing them.

The name must match exactly one policy (case-sensitive); otherwise the read
fails and lists the names that exist.

## Example Usage

```hcl
data "uptimewarden_escalation_policy" "p1" {
  name = "P1 on-call"
}

resource "uptimewarden_monitor" "api" {
  name                 = "API"
  type                 = "http"
  url                  = "https://api.acme.com/health"
  escalation_policy_id = data.uptimewarden_escalation_policy.p1.id
}
```

## Argument Reference

- `name` (Required) — Policy name.

## Attribute Reference

- `id` (String) — Public id (ULID).
- `step_count` (Number) — Number of escalation steps.
- `created_at` (String)
- `updated_at` (String)
