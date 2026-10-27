resource "aidial_model" "tf_aidial_integration_model_full" {
  bucket       = "platform"
  path         = "tf-aidial-ci-full-model-test"
  name         = "tf-aidial-ci-full-model-test"
  type         = "CHAT"
  display_name = "tf-dial Terraform Full Model Test"
  endpoint     = "https://example.invalid/v1/chat/completions"
  description  = "Temporary full-schema model test for terraform-provider-aidial."

  display_version = "1.0"
  icon_url        = "gpt3.svg"

  reference = "https://example.invalid/docs/tf-aidial-ci-full-model-test"
  author    = "tf-dial"

  user_roles = [
    "admin",
    "developer",
  ]

  dependencies = [
    "dependency-a",
  ]

  description_keywords = [
    "Integration Test",
    "Terraform",
  ]

  responses_endpoint = "https://example.invalid/v1/responses"

  input_attachment_types = [
    "application/pdf",
    "image/*",
  ]

  forward_auth_token = false
  max_retry_attempts = 5

  features = {
    accessible_by_per_request_key              = true
    allow_resume                               = true
    assistant_attachments_in_request_supported = true
    auto_caching_supported                     = true
    cache_supported                            = true
    configuration_endpoint                     = "https://example.invalid/config"
    consent_required                           = false
    content_parts_supported                    = true
    custom_temperature_supported               = true
    folder_attachments_supported               = true
    max_completion_tokens_supported            = true
    max_tokens_supported                       = true
    parallel_tool_calls_supported              = true
    rate_endpoint                              = "https://example.invalid/rate"
    reasoning_efforts                          = ["low", "medium", "high"]
    seed_supported                             = true
    support_comment_in_rate_response           = true
    system_prompt_supported                    = true
    temperature_supported                      = true
    tokenize_endpoint                          = "https://example.invalid/tokenize"
    tools_supported                            = true
    truncate_prompt_endpoint                   = "https://example.invalid/truncate"
    url_attachments_supported                  = true
  }

  limits = {
    max_prompt_tokens     = 200000
    max_completion_tokens = 128000
    max_total_tokens      = 300000
  }

  upstreams = [
    {
      endpoint = "https://example.invalid/v1/responses"
      weight   = 1
      tier     = 0
    },
  ]

  interceptors = []
}
