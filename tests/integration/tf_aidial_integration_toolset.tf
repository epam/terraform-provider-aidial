# Toolset in the platform bucket, with features, interfaces, defaults and an API key header.
resource "aidial_toolset" "tf_aidial_integration_toolset" {
  bucket      = "platform"
  path        = "tf-aidial-ci-test-toolset"
  description = "CI toolset"
  endpoint    = "http://toolset.example.invalid/mcp"
  transport   = "HTTP"

  allowed_tools = ["search", "fetch"]
  user_roles    = ["ci-role"]

  default_headers = {
    "X-Team" = "ci"
  }

  features = {
    tools_supported = true
  }

  auth_settings = {
    authentication_type = "API_KEY"
    api_key_header      = "X-Api-Key"
  }

  defaults = {
    timeout = 30
  }

  catalog_properties = {
    vendor = "Acme"
  }
}

data "aidial_toolset" "tf_aidial_integration_toolset" {
  bucket = aidial_toolset.tf_aidial_integration_toolset.bucket
  path   = aidial_toolset.tf_aidial_integration_toolset.path
}

# Bucket other than platform and a nested path, with a complete OAuth block (Core
# probes the endpoint of incomplete OAuth settings) and a client secret.
resource "aidial_toolset" "tf_aidial_integration_toolset_public" {
  bucket   = "public"
  path     = "tf-aidial-ci/nested/toolset"
  endpoint = "http://toolset.example.invalid/mcp"

  auth_settings = {
    authentication_type        = "OAUTH"
    authorization_endpoint     = "http://auth.example.invalid/authorize"
    token_endpoint             = "http://auth.example.invalid/token"
    token_endpoint_auth_method = "client_secret_post"
    client_id                  = "ci-client"
    client_secret              = var.aidial_service_key
    redirect_uri               = "http://redirect.example.invalid/callback"
    code_challenge_method      = "S256"
    dynamically_registered     = false
    resource                   = "http://toolset.example.invalid"
    scopes_supported           = ["read"]
  }
}

data "aidial_toolset" "tf_aidial_integration_toolset_public" {
  bucket = aidial_toolset.tf_aidial_integration_toolset_public.bucket
  path   = aidial_toolset.tf_aidial_integration_toolset_public.path
}
