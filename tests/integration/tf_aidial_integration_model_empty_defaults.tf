resource "aidial_model" "tf_aidial_integration_model_empty_defaults" {
  bucket   = "platform"
  path     = "tf-aidial-ci-test-model-empty-defaults"
  name     = "tf-aidial-ci-test-model-empty-defaults"
  type     = "CHAT"
  endpoint = "https://example.invalid/v1/chat/completions"

  defaults = {}
}