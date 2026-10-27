terraform {
  required_providers {
    aidial = {
      source = "epam/aidial"
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
