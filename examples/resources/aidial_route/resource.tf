resource "aidial_route" "example" {
  bucket = "platform"
  path   = "my-route"

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
