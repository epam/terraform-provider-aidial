resource "aidial_model" "bedrock_anthropic_claude_haiku_4_5_20251001_v1_0_with_thinking" {
  bucket          = "platform"
  path            = "bedrock.anthropic.claude-haiku-4-5-20251001-v1:0-with-thinking"
  name            = "bedrock.anthropic.claude-haiku-4-5-20251001-v1:0-with-thinking"
  type            = "CHAT"
  display_name    = "Anthropic Claude Haiku 4.5"
  display_version = "20251001-v1:0 with Thinking (AWS Bedrock)"
  override_name   = "anthropic.claude-haiku-4-5-20251001-v1:0-with-thinking"
  icon_url        = "anthropic.svg"
  description     = "Claude Haiku 4.5 (with extended thinking) delivers near-frontier performance with the speed and cost to power real-time products and high-volume agentic workloads."
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
  endpoint           = ""
  responses_endpoint = ""
  interfaces         = {}
  upstreams          = []
  user_roles         = []
  default_headers    = {}
  interceptors       = []
}
 