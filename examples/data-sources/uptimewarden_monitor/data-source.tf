data "uptimewarden_monitor" "existing" {
  id = "01HWXXXXXXXXXXXXXXXXXXXXXXX"
}

output "existing_monitor_url" {
  value = data.uptimewarden_monitor.existing.url
}
