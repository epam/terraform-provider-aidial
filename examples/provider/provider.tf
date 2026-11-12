terraform {
  required_providers {
    aidial = {
      source  = "epam/aidial"
      version = "~> 0.2"
    }
  }
}

provider "aidial" {
  base_url = "https://dial-core.example.com"
  api_key  = var.aidial_api_key
}

variable "aidial_api_key" {
  type      = string
  sensitive = true
}

# In CI/CD, authenticate with an OIDC ID token instead of an API key.
# Use one credential per provider block; both may also come from the
# AIDIAL_API_KEY / AIDIAL_ID_TOKEN environment variables.
provider "aidial" {
  alias    = "ci"
  base_url = "https://dial-core.example.com"
  id_token = var.aidial_id_token
}

variable "aidial_id_token" {
  type      = string
  sensitive = true
}
