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
