data "aidial_bucket" "current" {}

resource "aidial_role" "example" {
  bucket = data.aidial_bucket.current.bucket
  path   = "my-role"
  name   = "roles/${data.aidial_bucket.current.bucket}/my-role"

  description  = "Example role"
  display_name = "Example Role"

  cost_limit = {
    day   = 10
    week  = 50
    month = 200
  }

  topics = ["Example"]

  limits = {
    sample = jsonencode({ minute = 100, day = 1000 })
  }

  share = {
    sample = jsonencode({
      invitation_ttl     = 3600
      max_accepted_users = 10
    })
  }
}
