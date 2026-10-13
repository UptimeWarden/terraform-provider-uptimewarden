data "uptimewarden_regions" "all" {}

output "region_codes" {
  value = data.uptimewarden_regions.all.codes
}
