data "aidial_bucket" "current" {}

resource "aidial_route" "example" {
  bucket = data.aidial_bucket.current.bucket
  path   = "my-route"
  name   = "my-route"

  methods = ["GET", "POST"]
  paths   = ["/my-route/.*"]

  upstreams = [
    {
      endpoint = "https://example.invalid/v1/chat/completions"
      tier     = 0
      weight   = 1
    },
  ]
}
