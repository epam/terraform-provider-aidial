# An endpoint-based application.
resource "aidial_application" "example" {
  bucket       = "platform"
  path         = "my-application"
  display_name = "Support assistant"
  description  = "Answers questions about the product"
  endpoint     = "https://application.example.invalid/chat"

  forward_auth_token = true
  max_retry_attempts = 2
  interceptors       = ["my-interceptor"]

  default_headers = {
    "X-Team" = "platform"
  }
}

# A schema-based application: no endpoint, typed by an application type schema.
resource "aidial_application" "typed" {
  bucket                     = "platform"
  path                       = "my-typed-application"
  application_type_schema_id = aidial_application_type_schema.example.schema_id

  application_properties = {
    prompt = "You are a helpful assistant."
  }
}

# An application with MCP, its own routes and an OAuth external service.
resource "aidial_application" "full" {
  bucket   = "platform"
  path     = "my-full-application"
  endpoint = "https://application.example.invalid/chat"

  input_attachment_types = ["*/*"]
  description_keywords   = ["support"]
  dependencies           = ["my-model"]

  mcp = {
    endpoint  = "https://application.example.invalid/mcp"
    transport = "HTTP"
  }

  routes = {
    health = {
      paths = ["/health"]
      response = {
        status = 200
        body   = "ok"
      }
    }
  }

  allow_user_external_services = true
  external_services = {
    crm = {
      display_name = "CRM"
      auth_settings = {
        authentication_type    = "OAUTH"
        client_id              = "my-client"
        client_secret          = var.crm_client_secret # at least 8 characters, never read back from DIAL Core
        authorization_endpoint = "https://crm.example.invalid/authorize"
        token_endpoint         = "https://crm.example.invalid/token"
      }
    }
  }
}

variable "crm_client_secret" {
  type      = string
  sensitive = true
}
