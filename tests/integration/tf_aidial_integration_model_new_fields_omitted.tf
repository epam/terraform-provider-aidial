resource "aidial_model" "tf_aidial_integration_model_new_fields_omitted" {
  bucket       = "platform"
  path         = "tf-aidial-ci-test-model-new-fields-omitted"
  name         = "tf-aidial-ci-test-model-new-fields-omitted"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model"
  endpoint     = "https://example.invalid/v1/chat/completions"
}