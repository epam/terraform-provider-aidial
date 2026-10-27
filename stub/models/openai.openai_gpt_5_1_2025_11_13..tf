resource "aidial_model" "openai_gpt_5_1_2025_11_13" {
  bucket          = "platform"
  path            = "openai.gpt-5.1-2025-11-13"
  name            = "openai.gpt-5.1-2025-11-13"
  type            = "CHAT"
  display_name    = "GPT-5.1_char_test"
  display_version = "openai.gpt-5.1-2025-11-13"
  icon_url        = "gpt3.svg"
  description     = "GPT‑5.1 is ideal for tasks that require long-form reasoning and synthesis, such as analyzing complex documents or generating detailed reports, and for interactive workflows like spreadsheet editing, structured content creation, and coding assistance. It also excels in customizable communication, enabling tailored responses for support, education, and executive summaries across diverse domains."
  description_keywords = [
    "Continue Supported",
    "Image Recognition",
  ]
  input_attachment_types = [
    "text/json",
    "text/plain",
  ]
  max_input_attachments = 30
  features = {
    system_prompt_supported                    = true
    tools_supported                            = false
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
  pricing = {
    unit       = "token"
    prompt     = "0.00000125"
    completion = "0.00001"
  }
  max_retry_attempts = 5
  endpoint           = "${var.aidial_openai_host}/openai/deployments/gpt-5.1-2025-11-13/chat/completions"
  override_name      = "gpt-5.1-2025-11-13"
  responses_endpoint = var.responses_endpoint
  interfaces         = {}
  upstreams = [
    for idx, region in var.model_regions["gpt-5.1-2025-11-13"] : {
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
 