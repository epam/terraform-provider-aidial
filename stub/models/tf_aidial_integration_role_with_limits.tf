# Role with cost limits and per-model limits
resource "aidial_role" "tf_aidial_integration_role_with_limits" {
  name = "tf-aidial-integration-role-with-limits"

  cost_limit = {
    week  = 10
    month = 50
  }

  limits = {
    "bedrock.anthropic.claude-haiku-4-5-20251001-v1:0"                = jsonencode({})
    "bedrock.anthropic.claude-haiku-4-5-20251001-v1:0-with-thinking"  = jsonencode({})
    "bedrock.anthropic.claude-opus-4-1-20250805-v1:0"                 = jsonencode({})
    "bedrock.anthropic.claude-opus-4-1-20250805-v1:0-with-thinking"   = jsonencode({})
    "bedrock.anthropic.claude-opus-4-5-20251101-v1:0"                 = jsonencode({})
    "bedrock.anthropic.claude-opus-4-5-20251101-v1:0-with-thinking"   = jsonencode({})
    "bedrock.anthropic.claude-opus-4-6-v1"                            = jsonencode({})
    "bedrock.anthropic.claude-opus-4-6-v1-with-thinking"              = jsonencode({})
    "bedrock.anthropic.claude-opus-5"                                 = jsonencode({})
    "bedrock.anthropic.claude-sonnet-4-20250514-v1:0-with-thinking"   = jsonencode({})
    "bedrock.anthropic.claude-sonnet-4-5-20250929-v1:0"               = jsonencode({})
    "bedrock.anthropic.claude-sonnet-4-5-20250929-v1:0-with-thinking" = jsonencode({})
    "bedrock.anthropic.claude-sonnet-4-6"                             = jsonencode({})
    "bedrock.anthropic.claude-sonnet-4-6-with-thinking"               = jsonencode({})
    "bedrock.anthropic.claude-sonnet-5"                               = jsonencode({})
    "bedrock.anthropic.claude-sonnet-5-without-thinking"              = jsonencode({})
    "bedrock.anthropic.claude-v3-haiku"                               = jsonencode({})
    "bedrock.deepseek.v3.2"                                           = jsonencode({})
    "bedrock.meta.llama4-maverick-17b-instruct-v1:0"                  = jsonencode({})
    "bedrock.meta.llama4-scout-17b-instruct-v1:0"                     = jsonencode({})
    "bedrock.minimax.minimax-m2.1"                                    = jsonencode({})
    "bedrock.minimax.minimax-m2.5"                                    = jsonencode({})
    "bedrock.nvidia.nemotron-nano-12b-v2"                             = jsonencode({})
    "bedrock.nvidia.nemotron-super-3-120b"                            = jsonencode({})
    "bedrock.qwen.qwen3-coder-480b-a35b-v1:0"                         = jsonencode({})
    "bedrock.stability.sd3-5-large-v1:0"                              = jsonencode({})
    "bedrock.stability.stable-image-core-v1:1"                        = jsonencode({})
    "bedrock.stability.stable-image-ultra-v1:1"                       = jsonencode({})
    "bedrock.xai.grok-4.6"                                            = jsonencode({})
    "bedrock.zai.glm-4.7"                                             = jsonencode({})
    "bedrock.zai.glm-4.7-flash"                                       = jsonencode({})
    "bedrock.zai.glm-5"                                               = jsonencode({})
    "openai.gpt-4" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-4.1-mini-2025-04-14" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-4.1-nano-2025-04-14" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-4o" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-4o-2024-11-20" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-4o-transcribe" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5-2025-08-07" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5-2025-08-07-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5-codex-2025-09-15" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5-codex-2025-09-15-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5-mini-2025-08-07" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5-mini-2025-08-07-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5-nano-2025-08-07" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5-nano-2025-08-07-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.1-2025-11-13" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.1-2025-11-13-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.1-codex-2025-11-13" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.1-codex-2025-11-13-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.1-codex-mini-2025-11-13" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.1-codex-mini-2025-11-13-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.2-2025-12-11" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.2-2025-12-11-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.2-codex-2026-01-14-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.3-codex-2026-02-24" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.3-codex-2026-02-24-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.4-2026-03-05" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.4-2026-03-05-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.4-mini-2026-03-17" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.4-mini-2026-03-17-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.4-nano-2026-03-17" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.4-nano-2026-03-17-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.5-2026-04-24" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.5-2026-04-24-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.6-luna-2026-07-09" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.6-luna-2026-07-09-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.6-sol-2026-07-09" = jsonencode({})
    "openai.gpt-5.6-sol-2026-07-09-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.6-terra-2026-07-09" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-5.6-terra-2026-07-09-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-chat-latest" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-chat-latest-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-image-1-mini-2025-10-06" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.gpt-image-1.5-2025-12-16" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.o1-2024-12-17" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.o1-2024-12-17-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.o3-2025-04-16" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.o3-2025-04-16-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.o3-mini-2025-01-31" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.o3-mini-2025-01-31-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.o4-mini-2025-04-16" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.o4-mini-2025-04-16-reasoning" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.text-embedding-3-large-1" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.text-embedding-3-small-1" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.text-embedding-ada-002" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.tts-001" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.tts-hd-001" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "openai.whisper-001" = jsonencode({
      minute = 64000000
      day    = 100000000
    })
    "vertex.claude-haiku-4-5@20251001"            = jsonencode({})
    "vertex.claude-opus-4-1@20250805"             = jsonencode({})
    "vertex.claude-opus-4-5@20251101"             = jsonencode({})
    "vertex.claude-opus-4-6@default"              = jsonencode({})
    "vertex.claude-opus-4-7@default"              = jsonencode({})
    "vertex.claude-opus-4-8@default"              = jsonencode({})
    "vertex.claude-opus-4@20250514"               = jsonencode({})
    "vertex.claude-opus-5@default"                = jsonencode({})
    "vertex.claude-sonnet-4-5@20250929"           = jsonencode({})
    "vertex.claude-sonnet-4-6@default"            = jsonencode({})
    "vertex.claude-sonnet-4@20250514"             = jsonencode({})
    "vertex.claude-sonnet-5@default"              = jsonencode({})
    "vertex.gemini-2.5-flash"                     = jsonencode({})
    "vertex.gemini-2.5-flash-google-search"       = jsonencode({})
    "vertex.gemini-2.5-flash-image"               = jsonencode({})
    "vertex.gemini-2.5-flash-lite"                = jsonencode({})
    "vertex.gemini-2.5-pro"                       = jsonencode({})
    "vertex.gemini-2.5-pro-google-search"         = jsonencode({})
    "vertex.gemini-3-flash-preview"               = jsonencode({})
    "vertex.gemini-3-flash-preview-google-search" = jsonencode({})
    "vertex.gemini-3-pro-image"                   = jsonencode({})
    "vertex.gemini-3-pro-image-preview"           = jsonencode({})
    "vertex.gemini-3.1-flash-image"               = jsonencode({})
    "vertex.gemini-3.1-flash-image-preview"       = jsonencode({})
    "vertex.gemini-3.1-flash-lite"                = jsonencode({})
    "vertex.gemini-3.1-pro-preview"               = jsonencode({})
    "vertex.gemini-3.1-pro-preview-google-search" = jsonencode({})
    "vertex.gemini-3.1-pro-preview-with-thinking" = jsonencode({})
    "vertex.gemini-3.5-flash"                     = jsonencode({})
    "vertex.gemini-3.5-flash-google-search"       = jsonencode({})
    "vertex.gemini-3.5-flash-lite"                = jsonencode({})
    "vertex.gemini-3.5-flash-lite-google-search"  = jsonencode({})
    "vertex.gemini-3.5-flash-with-thinking"       = jsonencode({})
    "vertex.gemini-3.6-flash"                     = jsonencode({})
    "vertex.gemini-3.6-flash-google-search"       = jsonencode({})
    "vertex.gemini-3.6-flash-with-thinking"       = jsonencode({})
    "vertex.gemini-3.7-flash"                     = jsonencode({})
    "vertex.gemini-3.7-flash-google-search"       = jsonencode({})
    "vertex.gemini-3.7-flash-with-thinking"       = jsonencode({})
    "vertex.gemini-3.8-flash"                     = jsonencode({})
    "vertex.gemini-3.8-flash-google-search"       = jsonencode({})
    "vertex.gemini-3.8-flash-with-thinking"       = jsonencode({})
    "vertex.gemini-embedding-2"                   = jsonencode({})
    "vertex.text-embedding-005"                   = jsonencode({})
    "vertex.veo-3.1-fast-generate-001"            = jsonencode({})
    "vertex.veo-3.1-generate-001"                 = jsonencode({})
    "bedrock.anthropic.claude-opus-4-8"           = jsonencode({})
  }
}
 