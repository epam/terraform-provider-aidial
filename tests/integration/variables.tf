variable "aidial_base_url" {
  type = string
}

variable "aidial_api_key" {
  type      = string
  sensitive = true
}

variable "aidial_role_bucket" {
  type = string
}

variable "aidial_role_path" {
  type = string
}

variable "aidial_service_key" {
  type      = string
  sensitive = true
}

variable "aidial_admin_url" {
  type        = string
  default     = "https://example.com"
  description = "Base URL of the DIAL admin console (distinct from aidial_base_url, the API host), used only to compose convenience output links."
}