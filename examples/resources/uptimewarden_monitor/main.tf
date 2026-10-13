data "uptimewarden_escalation_policy" "p1" {
  name = "P1 on-call"
}

data "uptimewarden_webhook" "slack" {
  name = "Slack #ops"
}

resource "uptimewarden_monitor" "api_health" {
  name        = "API Health"
  type        = "http"
  url         = "https://api.acme.com/health"
  interval    = 60
  timeout     = 10
  external_id = "api-health"

  expected_status_codes   = "200-299"
  response_time_threshold = 800

  alert_types          = ["down", "recovery"]
  email_addresses      = "oncall@acme.com"
  escalation_policy_id = data.uptimewarden_escalation_policy.p1.id
  selected_webhook_ids = [data.uptimewarden_webhook.slack.id]
}

resource "uptimewarden_monitor" "db_port" {
  name     = "Database port"
  type     = "tcp"
  url      = "tcp://db.example.com:5432"
  interval = 60
}

resource "uptimewarden_monitor" "mx" {
  name = "MX records"
  type = "dns"
  url  = "acme.com"
  dns_config = {
    record_type      = "MX"
    expected_answers = ["mx1.acme.com"]
  }
}

resource "uptimewarden_monitor" "nightly_backup" {
  name                    = "Nightly backup job"
  type                    = "heartbeat"
  interval                = 86400
  heartbeat_grace_seconds = 1800
}

output "backup_ping_url" {
  value     = uptimewarden_monitor.nightly_backup.heartbeat_url
  sensitive = true
}
