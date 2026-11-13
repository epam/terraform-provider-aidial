# Translator in the platform bucket: converts OpenAI chat completions to Anthropic messages.
resource "aidial_translator" "tf_aidial_integration_translator" {
  bucket   = "platform"
  path     = "tf-aidial-ci-test-translator"
  base_url = "http://translator.example.invalid:8000"
  in       = "openaiChatCompletions"
  out      = "anthropicMessages"
}

data "aidial_translator" "tf_aidial_integration_translator" {
  bucket = aidial_translator.tf_aidial_integration_translator.bucket
  path   = aidial_translator.tf_aidial_integration_translator.path
}
