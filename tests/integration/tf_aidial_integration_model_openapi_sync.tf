# Fields added to sync aidial_model and aidial_route with the DIAL Core OpenAPI.
resource "aidial_model" "tf_aidial_integration_model_openapi_sync" {
  bucket   = "platform"
  path     = "tf-aidial-ci-test-model-openapi-sync"
  name     = "tf-aidial-ci-test-model-openapi-sync"
  type     = "CHAT"
  endpoint = "https://example.invalid/v1/chat/completions"

  base_url = "http://adapter.example/"

  responses_defaults = {
    max_output_tokens = 100
  }

  interfaces = {
    openaiResponses = {
      mode            = "passthrough"
      override_paths  = { "/v1/responses" = "/responses" }
      default_headers = { "X-Test" = "1" }
    }
    anthropicMessages = {
      base_url = "http://bedrock-adapter.example"
    }
  }

  upstreams = [
    {
      endpoint  = "https://example.invalid/v1/chat/completions"
      base_url  = "https://example.invalid"
      auth_type = "BEARER"
      weight    = 1
      tier      = 0
    },
  ]
}

resource "aidial_model" "tf_aidial_integration_model_embedding" {
  bucket   = "platform"
  path     = "tf-aidial-ci-test-model-embedding"
  name     = "tf-aidial-ci-test-model-embedding"
  type     = "EMBEDDING"
  endpoint = "https://example.invalid/v1/embeddings"

  embedding_dimensions = 1536
}

resource "aidial_route" "tf_aidial_integration_route_upstream_fields" {
  bucket = "platform"
  path   = "tf-aidial-ci-test-route-upstream-fields"
  paths  = ["/tf-aidial-ci-upstream-fields"]

  upstreams = [
    {
      endpoint  = "https://example.invalid/route"
      base_url  = "https://example.invalid"
      auth_type = "API_KEY"
      weight    = 1
      tier      = 0
    },
  ]
}
