# created_at and updated_at are sent as written and read back unchanged.
resource "aidial_model" "tf_aidial_integration_model_timestamps" {
  bucket       = "platform"
  path         = "tf-aidial-ci-test-model-timestamps"
  name         = "tf-aidial-ci-test-model-timestamps"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model Timestamps"
  endpoint     = "https://example.invalid/v1/chat/completions"
  created_at   = 1630454400000
  updated_at   = 1630454400000
}
