# Changelog

## Unreleased

### Removed

- `uptimewarden_status_page`, `uptimewarden_escalation_policy`,
  `uptimewarden_on_call_schedule`, `uptimewarden_webhook_config` and
  `uptimewarden_api_token`. The UptimeWarden API does not accept API tokens on
  the endpoints these resources called, so every apply with them failed to
  authenticate. Status pages, escalation policies and on-call schedules are
  planned as resources once the API exposes them to API tokens.

### Changed

- `uptimewarden_monitor.timeout` no longer defaults to `30` in the plan. When
  unset it reads back the API's value. A configured value is set with an
  update right after the create, because the API's create path does not apply
  it yet; this removes "Provider produced inconsistent result after apply".
- `uptimewarden_monitor.region` no longer defaults to `gwc` and is only sent
  when configured, so plans without region selection work. It accepts `ae`
  (Sydney), `brs` (São Paulo), `gwc` (Sweden Central), `sea` (East Asia) and
  `wus3` (Phoenix); `uaen` is retired.
- When the API returns a different value for a configured attribute, the
  apply fails with an error naming the attribute and both values.
- The provider reads the API base URL from `UW_API_URL`, the variable the CLI
  and GitHub Actions use. `UW_BASE_URL` still works as an alias.
