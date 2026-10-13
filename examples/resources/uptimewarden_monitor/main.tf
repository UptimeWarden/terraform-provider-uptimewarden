resource "uptimewarden_monitor" "homepage" {
  name     = "Homepage"
  type     = "http"
  url      = "https://acme.com"
  interval = 60
  region   = "gwc"
}

resource "uptimewarden_monitor" "api_health" {
  name     = "API Health"
  type     = "http"
  url      = "https://api.acme.com/health"
  interval = 30
  timeout  = 10
  region   = "sea"
}

resource "uptimewarden_monitor" "db_port" {
  name     = "Database port"
  type     = "tcp"
  url      = "tcp://db.example.com:5432"
  interval = 60
}

resource "uptimewarden_monitor" "grpc_service" {
  name     = "gRPC auth service"
  type     = "grpc"
  url      = "grpc://auth.example.com:50051"
  interval = 60
}

resource "uptimewarden_monitor" "daily_backup" {
  name     = "Nightly backup job"
  type     = "heartbeat"
  url      = "heartbeat://daily-backup"
  interval = 86400
}

# Outbound webhook — receives alert notifications
resource "uptimewarden_webhook_config" "pagerduty" {
  type_id     = 1
  name        = "PagerDuty Integration"
  url         = "https://events.pagerduty.com/integration/XXXXX/enqueue"
  http_method = "POST"
}

# Status page
resource "uptimewarden_status_page" "public" {
  name  = "Acme Status"
  title = "Acme System Status"
  slug  = "acme-status"
}

# On-call schedule: Alice Mon–Wed, Bob Thu–Sun
resource "uptimewarden_on_call_schedule" "primary" {
  name     = "Primary On-Call"
  timezone = "Europe/Istanbul"
  members = [
    { user_id = 1, day_of_week = 1 }, # Alice — Monday
    { user_id = 1, day_of_week = 2 }, # Alice — Tuesday
    { user_id = 1, day_of_week = 3 }, # Alice — Wednesday
    { user_id = 2, day_of_week = 4 }, # Bob   — Thursday
    { user_id = 2, day_of_week = 5 }, # Bob   — Friday
  ]
}

# Escalation policy: page the schedule, then a fallback user after 10 min
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
      target_id   = 3 # fallback user ID
      wait_minutes = 5
      channels    = ["sms", "push"]
    },
  ]
}

# API token for CI/CD pipelines
resource "uptimewarden_api_token" "ci" {
  name        = "CI/CD Pipeline"
  permissions = ["monitors:read", "incidents:read"]
}

output "ci_token" {
  value     = uptimewarden_api_token.ci.token
  sensitive = true
}

# Read an existing monitor (created outside Terraform)
data "uptimewarden_monitor" "existing" {
  id = "01HWXXXXXXXXXXXXXXXXXXXXXX"
}

output "existing_monitor_url" {
  value = data.uptimewarden_monitor.existing.url
}
