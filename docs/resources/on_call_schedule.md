---
page_title: "uptimewarden_on_call_schedule Resource"
description: |-
  Manages a weekly on-call rotation schedule.
---

# uptimewarden_on_call_schedule

Manages a weekly on-call rotation schedule. Each member is assigned to one or more days of the week. Schedules are referenced by escalation policy steps with `target_type = "schedule"`.

## Example Usage

```hcl
resource "uptimewarden_on_call_schedule" "primary" {
  name     = "Engineering On-Call"
  timezone = "Europe/Istanbul"
  members = [
    { user_id = 1, day_of_week = 0 }, # Alice — Sunday
    { user_id = 1, day_of_week = 1 }, # Alice — Monday
    { user_id = 1, day_of_week = 2 }, # Alice — Tuesday
    { user_id = 2, day_of_week = 3 }, # Bob   — Wednesday
    { user_id = 2, day_of_week = 4 }, # Bob   — Thursday
    { user_id = 2, day_of_week = 5 }, # Bob   — Friday
    { user_id = 3, day_of_week = 6 }, # Carol — Saturday
  ]
}
```

## Argument Reference

- `name` (Required) — Schedule name.
- `timezone` (Optional, Default: `UTC`) — IANA timezone (e.g. `Europe/Istanbul`, `America/New_York`).
- `members` (Optional) — On-call assignments. Maximum 7 (one per day).
  - `user_id` (Required) — Internal user ID.
  - `day_of_week` (Required) — Day of week: `0`=Sunday, `1`=Monday … `6`=Saturday.

## Attribute Reference

- `id` — Public ID (ULID).

## Import

```bash
terraform import uptimewarden_on_call_schedule.primary 01HWXXXXXXXXXXXXXXXXXXXXXXXXX
```
