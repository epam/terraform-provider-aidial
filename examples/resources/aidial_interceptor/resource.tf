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
