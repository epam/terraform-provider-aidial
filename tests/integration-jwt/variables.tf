variable "aidial_base_url" {
  type = string
}

variable "aidial_id_token" {
  type      = string
  sensitive = true
}

variable "aidial_role_bucket" {
  type = string
}

variable "aidial_role_path" {
  type = string
}
