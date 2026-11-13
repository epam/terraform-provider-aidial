resource "aidial_toolset" "example" {
  bucket      = "platform"
  path        = "my-toolset"
  description = "Search and fetch tools served over MCP"
  endpoint    = "https://mcp.example.invalid/mcp"
  transport   = "HTTP"

  # Expose only some of the tools the server offers.
  allowed_tools = ["search", "fetch"]

  auth_settings = {
    authentication_type = "API_KEY"
    api_key_header      = "X-Api-Key"
  }
}

# Other buckets, such as "public", and nested paths work as well. For OAuth,
# provide the full block; the client secret stays out of the plan output.
resource "aidial_toolset" "oauth" {
  bucket   = "public"
  path     = "team/oauth-toolset"
  endpoint = "https://mcp.example.invalid/mcp"

  auth_settings = {
    authentication_type        = "OAUTH"
    authorization_endpoint     = "https://auth.example.invalid/authorize"
    token_endpoint             = "https://auth.example.invalid/token"
    token_endpoint_auth_method = "client_secret_post"
    client_id                  = "my-client"
    client_secret              = var.toolset_client_secret
    redirect_uri               = "https://dial.example.invalid/oauth/callback"
    code_challenge_method      = "S256"
    dynamically_registered     = false
    resource                   = "https://mcp.example.invalid"
    scopes_supported           = ["read"]
  }
}

variable "toolset_client_secret" {
  type      = string
  sensitive = true
}
