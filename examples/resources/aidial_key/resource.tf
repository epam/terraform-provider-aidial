variable "aidial_key_value" {
  type      = string
  sensitive = true
}

resource "aidial_key" "example" {
  bucket = "platform"
  path   = "my-key"

  key     = var.aidial_key_value
  project = "my-project"
  role    = "developer"
  roles   = ["developer"]
  secured = true
}
