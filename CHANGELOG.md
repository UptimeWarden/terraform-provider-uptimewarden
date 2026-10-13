# Changelog

## Unreleased

### Added

- `uptimewarden_monitor` covers the whole `/api/v1/monitors` contract: check
  settings (`method`, `custom_headers`, `request_body`,
  `expected_status_codes`, `expected_body`, `content_validation`,
  `follow_redirects`, `verify_ssl`, `enable_ssl`, basic auth, `bearer_token`,
  `hostname`, `port`, gRPC, `dns_config`, `agent_id`), regions and consensus
  (`verification_regions`, `min_confirming_regions`), thresholds
  (`retry_count`, `response_time_threshold`, SSL and domain expiry, security
  analysis), alert routing (`alert_types`, `notifications`, recipients,
  `voice_mode`, `escalation_policy_id`, `selected_webhook_ids`),
  `maintenance_windows`, heartbeat settings, `description`, `tags`, `groups`
  and `external_id`. Computed: `public_id`, `heartbeat_token` and
  `heartbeat_url` (sensitive), `managed_by_source`, `created_at`.
- Data sources `uptimewarden_regions`, `uptimewarden_escalation_policy` (by
  name) and `uptimewarden_webhook` (by name). `uptimewarden_monitor` data
  source returns every field.
- API errors show the API's code, message and details, e.g.
  `UptimeWarden API 400 VALIDATION_ERROR: Invalid monitoring region (Valid regions: …)`.

### Changed

- Changing `type` updates the monitor in place, except to or from
  `heartbeat`, which still replaces it.
- `url` is optional for `heartbeat` monitors (UptimeWarden sets it) and
  rejected there; it is still required for every other type.
- Optional attributes left out of the configuration keep the value
  UptimeWarden holds; updates send only changed attributes.
- `region` and `verification_regions` are checked by the API against the
  active regions (the provider only checks the code format), so a new region
  works without a provider release.
- `uptimewarden_monitor.timeout` no longer defaults to `30` in the plan. When
  unset it reads back the API's value. (A follow-up update still sets it if an
  older API ignores it on create.)
- `uptimewarden_monitor.region` no longer defaults to `gwc` and is only sent
  when configured. Codes: `ae` (Sydney), `brs` (São Paulo), `gwc` (Sweden
  Central), `sea` (East Asia), `wus3` (Phoenix); `uaen` is retired.
- When the API returns a different value for a configured attribute, the
  apply fails with an error naming the attribute and both values.
- The provider reads the API base URL from `UW_API_URL`, the variable the CLI
  and GitHub Actions use. `UW_BASE_URL` still works as an alias.

### Removed

- `uptimewarden_status_page`, `uptimewarden_escalation_policy`,
  `uptimewarden_on_call_schedule`, `uptimewarden_webhook_config` and
  `uptimewarden_api_token`. The UptimeWarden API does not accept API tokens on
  the endpoints these resources called, so every apply with them failed to
  authenticate. Status pages, escalation policies and on-call schedules are
  planned as resources once the API exposes them to API tokens.
