terraform {
  required_providers {
    uptimewarden = {
      source  = "uptimewarden/uptimewarden"
      version = "~> 0.1"
    }
  }
}

# Token comes from UW_API_TOKEN environment variable by default.
# Set base_url (or UW_API_URL) to point at another API, such as staging.
provider "uptimewarden" {}
