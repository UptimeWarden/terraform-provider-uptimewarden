---
page_title: "uptimewarden_escalation_policy Resource"
description: |-
  Manages an on-call escalation policy.
---

# uptimewarden_escalation_policy

Manages an on-call escalation policy. A policy defines an ordered chain of targets (users or schedules) that receive pages when an incident is not acknowledged within the configured wait time.

## Example Usage

```hcl
resource "uptimewarden_on_call_schedule" "primary" {
  name = "Primary On-Call"
  members = [
    { user_id = 1, day_of_week = 1 },
    { user_id = 2, day_of_week = 4 },
  ]
}

resource "uptimewarden_escalation_policy" "p1" {
  name = "P1 On-Call"
  steps = [
    {
      step_order  = 0
      target_type = "schedule"
      target_id   = uptimewarden_on_call_schedule.primary.id
      wait_minutes = 10
      channels    = ["email", "push"]
    },
    {
      step_order  = 1
      target_type = "user"
      target_id   = 42  # fallback user internal ID
      wait_minutes = 5
      channels    = ["sms"]
    },
  ]
}
```

## Argument Reference

- `name` (Required) — Policy name.
- `steps` (Required) — Ordered escalation steps (1–10).
  - `step_order` (Required) — Step position (0–9).
  - `target_type` (Required) — `user` or `schedule`.
  - `target_id` (Required) — Internal ID of the user or on-call schedule.
  - `wait_minutes` (Optional, Default: `5`) — Minutes before escalating to next step (0–1440).
  - `channels` (Optional, Default: `["email","sms","push"]`) — Notification channels for this step.

## Attribute Reference

- `id` — Public ID (ULID).

## Import

```bash
terraform import uptimewarden_escalation_policy.p1 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
```
