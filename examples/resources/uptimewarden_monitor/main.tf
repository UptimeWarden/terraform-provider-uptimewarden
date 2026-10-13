resource "uptimewarden_monitor" "homepage" {
  name     = "Homepage"
  type     = "http"
  url      = "https://acme.com"
  interval = 60
}

resource "uptimewarden_monitor" "api_health" {
  name     = "API Health"
  type     = "http"
  url      = "https://api.acme.com/health"
  interval = 60
  timeout  = 10
  region   = "sea" # East Asia; choosing a region needs Professional or Enterprise
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

# Read an existing monitor (created outside Terraform)
data "uptimewarden_monitor" "existing" {
  id = "01HWXXXXXXXXXXXXXXXXXXXXXX"
}

output "existing_monitor_url" {
  value = data.uptimewarden_monitor.existing.url
}
