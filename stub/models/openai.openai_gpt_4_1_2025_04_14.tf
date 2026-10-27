resource "aidial_model" "openai_gpt_4_1_2025_04_14" {
  bucket          = "platform"
  path            = "openai.gpt-4.1-2025-04-14"
  name            = "openai.gpt-4.1-2025-04-14"
  type            = "CHAT"
  display_name    = "GPT-4.1"
  display_version = "openai.gpt-4.1-2025-04-14"
  icon_url        = "gpt4.svg"
  description     = "GPT-4.1 is a flagship model for complex tasks. It is well suited for problem solving across domains."
  description_keywords = [
    "Text Generation",
    "Continue Supported",
    "Image Recognition",
  ]
  input_attachment_types = [
    "*/*",
  ]
  features = {
    system_prompt_supported                    = true
    tools_supported                            = true
    seed_supported                             = false
    url_attachments_supported                  = false
    folder_attachments_supported               = false
    allow_resume                               = true
    accessible_by_per_request_key              = true
    content_parts_supported                    = false
    temperature_supported                      = true
    parallel_tool_calls_supported              = true
    assistant_attachments_in_request_supported = false
    max_tokens_supported                       = true
    custom_temperature_supported               = true
    skills_supported                           = false
  }
  limits = {
    max_total_tokens      = 1047576
    max_completion_tokens = 32768
  }
  pricing = {
    unit       = "token"
    prompt     = "0.000002"
    completion = "0.000008"
  }
  tokenizer_model    = "gpt-4-0314"
  max_retry_attempts = 5
  endpoint           = "${var.aidial_openai_host}/openai/deployments/gpt-4.1-2025-04-14/chat/completions"
  override_name      = "gpt-4.1-2025-04-14"
  responses_endpoint = var.responses_endpoint
  interfaces         = {}
  upstreams = [
    for idx, region in var.model_regions["gpt-4.1-2025-04-14"] : {
      id       = tostring(idx + 1)
      endpoint = "${var.cognitive_account_endpoints[region]}openai/v1/responses"
      weight   = 1
      tier     = 0
    }
  ]
  user_roles = [
    "admin",
    "default",
    "full",
    "hitlimit",
    "user",
  ]
  default_headers    = {}
  interceptors       = []
  forward_auth_token = false
}
 