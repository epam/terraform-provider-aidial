resource "aidial_translator" "example" {
  bucket   = "platform"
  path     = "openai-to-claude"
  base_url = "https://translator.example.invalid"

  # Interface types: openaiChatCompletions, openaiEmbeddings,
  # openaiResponses, anthropicMessages.
  in  = "openaiChatCompletions"
  out = "anthropicMessages"
}
