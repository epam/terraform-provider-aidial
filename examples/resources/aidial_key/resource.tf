data "aidial_bucket" "current" {}

variable "aidial_key_value" {
  type      = string
  sensitive = true
}

resource "aidial_key" "example" {
  bucket = data.aidial_bucket.current.bucket
  path   = "my-key"

  key     = var.aidial_key_value
  project = "my-project"
  role    = "developer"
  roles   = ["developer"]
  secured = true
}
