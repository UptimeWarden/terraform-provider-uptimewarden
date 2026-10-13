terraform {
  required_providers {
    uptimewarden = {
      source  = "uptimewarden/uptimewarden"
      version = "~> 0.1"
    }
  }
}

# Token comes from UW_API_TOKEN environment variable by default.
# Set base_url to override for staging or self-hosted environments.
provider "uptimewarden" {}
