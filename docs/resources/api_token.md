---
page_title: "uptimewarden_api_token Resource"
description: |-
  Creates a UptimeWarden API token with granular permissions.
---

# uptimewarden_api_token

Creates a UptimeWarden API token with granular permission scopes. The plain token value is returned **only at creation** and stored as a sensitive value in Terraform state. Destroying and re-creating this resource generates a new token.

> **Security note:** Store the `token` output in a secrets manager (e.g. AWS Secrets Manager, HashiCorp Vault) rather than in Terraform output files.

## Example Usage

```hcl
resource "uptimewarden_api_token" "ci" {
  name        = "CI/CD Pipeline"
  permissions = ["monitors:read", "incidents:read"]
}

output "ci_token" {
  value     = uptimewarden_api_token.ci.token
  sensitive = true
}
```

### Common permission sets

```hcl
# Monitor viewer (read-only)
permissions = ["monitors:read", "monitors:results:read"]

# Incident responder
permissions = ["incidents:read", "incidents:update"]

# Full access
permissions = ["monitors:read", "monitors:create", "monitors:delete",
               "incidents:read", "incidents:update"]
```

## Argument Reference

- `name` (Required) — Token name.
- `permissions` (Optional) — Permission strings. See the [API token documentation](https://docs.uptimewarden.online/developers/api/tokens) for the full list.
- `expires_at` (Optional) — ISO 8601 expiry timestamp (e.g. `2027-01-01T00:00:00Z`). Omit for no expiry.
- `is_active` (Optional, Default: `true`) — Whether the token is active.

## Attribute Reference

- `id` — Public ID (ULID).
- `prefix` — First 12 characters of the token (safe to log).
- `token` — Full plain-text token (sensitive). Available only immediately after creation.

## Import

`uptimewarden_api_token` does **not** support `terraform import`. The plain token value cannot be recovered after creation. Create a new token via Terraform instead.
