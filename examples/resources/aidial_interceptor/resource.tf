resource "aidial_interceptor" "example" {
  bucket       = "platform"
  path         = "my-interceptor"
  display_name = "Content filter"
  description  = "Checks chat completions before they reach the model"
  endpoint     = "https://interceptor.example.invalid/intercept"

  forward_auth_token = true
  max_retry_attempts = 2

  default_headers = {
    "X-Team" = "platform"
  }
}

# An interceptor that also sets the optional catalog and attachment fields.
resource "aidial_interceptor" "detailed" {
  bucket   = "platform"
  path     = "my-detailed-interceptor"
  endpoint = "https://interceptor.example.invalid/intercept"

  description_keywords   = ["content", "filter"]
  dependencies           = ["my-model"]
  input_attachment_types = ["image/png"]
  max_input_attachments  = 2
}
