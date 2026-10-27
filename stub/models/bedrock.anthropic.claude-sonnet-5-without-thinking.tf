resource "aidial_model" "bedrock_anthropic_claude_sonnet_5_without_thinking" {
  bucket          = "platform"
  path            = "bedrock.anthropic.claude-sonnet-5-without-thinking"
  name            = "bedrock.anthropic.claude-sonnet-5-without-thinking"
  type            = "CHAT"
  display_name    = "Anthropic Claude Sonnet 5"
  display_version = "bedrock.anthropic.claude-sonnet-5-without-thinking"
  icon_url        = "anthropic.svg"
  description     = "Claude Sonnet 5 is Anthropic's most capable Sonnet model yet, built for coding, agents, and professional work at scale. It brings near-Opus intelligence to the model teams run every day, with the same balance of capability, cost, and speed teams already rely on Sonnet for. Agents: Claude Sonnet 5 can operate as both a lead agent and subagent in production pipelines, with reliable tool use and the cost profile to run high-volume agentic workloads at scale. Coding: Claude Sonnet 5 is built for everyday development work: building features, refactoring, and debugging across complex codebases without losing quality. Enterprise workflows: Claude Sonnet 5 reduces the rounds of editing needed to reach production-ready documents, spreadsheets, and presentations, making it a reliable everyday model for office tasks. Financial analysis: Claude Sonnet 5 pairs general intelligence improvements with stronger analysis and spreadsheet work, fitting high-volume financial workflows where precision and iteration speed matter. Computer use: Claude Sonnet 5 is an efficient computer-use model, making browser automation cost-effective enough to deploy across business tools at scale."
  description_keywords = [
    "Text Generation",
    "Reasoning",
    "TXT Supported",
    "Development",
    "Image Recognition",
    "Functions",
    "PDF Supported",
    "AWS",
  ]
  input_attachment_types = [
    "image/*",
    "text/plain",
    "text/markdown",
    "application/pdf",
  ]
  features = {
    tools_supported         = true
    system_prompt_supported = true
  }
  defaults = {
    max_tokens = 64000
    custom_fields = {
      configuration = {
        thinking = {
          type = "disabled"
        }
      }
    }
  }
  limits = {
    max_total_tokens = 200000
  }
  pricing = {
    unit       = "token"
    prompt     = "0.000002"
    completion = "0.00001"
  }
  endpoint           = "http://bedrock-adapter.example.svc.cluster.local.:80/openai/deployments/global.anthropic.claude-sonnet-5/chat/completions"
  override_name      = "global.anthropic.claude-sonnet-5"
  responses_endpoint = ""
  interfaces         = {}
  upstreams = [
    {
      id = "1"
      extra_data = {
        region              = "us-east-1"
        compatible_model_id = "anthropic.claude-sonnet-4-6"
      }
      weight = 1
      tier   = 0
    },
  ]
  user_roles = [
    "admin",
    "default",
    "full",
    "hitlimit",
    "user",
  ]
  default_headers = {}
  interceptors    = []
}
