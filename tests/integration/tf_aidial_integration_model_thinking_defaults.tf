resource "aidial_model" "tf_aidial_integration_model_thinking_defaults" {
  bucket          = "platform"
  path            = "tf-aidial-ci-test-model-thinking-defaults"
  name            = "tf-aidial-ci-test-model-thinking-defaults"
  type            = "CHAT"
  display_name    = "tf-dial Terraform Test Model"
  display_version = "thinking-defaults"
  override_name   = "tf-aidial-ci-test-model-thinking-defaults"
  icon_url        = "anthropic.svg"
  description     = "Integration test for a model with defaults.custom_fields (extended thinking) and upstreams = []."
  description_keywords = [
    "Text Generation",
    "Image Recognition",
    "PDF Supported",
    "TXT Supported",
    "MD Supported",
    "AWS",
    "Functions",
    "Development",
    "Reasoning",
  ]
  input_attachment_types = [
    "image/*",
    "text/plain",
    "text/markdown",
    "application/pdf",
  ]
  features = {
    cache_supported = true
    tools_supported = true
  }
  defaults = {
    max_tokens = 64000
    custom_fields = {
      configuration = {
        thinking = {
          type          = "enabled"
          budget_tokens = 16000
        }
      }
    }
  }
  limits = {
    max_total_tokens = 200000
  }
  pricing = {
    unit       = "token"
    prompt     = "0.0000008"
    completion = "0.000004"
    cacheRead  = "0.0000001"
    cacheWrite = "0.00000125"
  }
  endpoint           = "https://example.invalid/v1/chat/completions"
  responses_endpoint = ""
  interfaces         = {}
  upstreams          = []
  user_roles         = []
  default_headers    = {}
  interceptors       = []
}
 