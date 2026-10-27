resource "aidial_model" "tf_aidial_integration_model_no_dependencies" {
  bucket       = "platform"
  path         = "tf-aidial-ci-test-model-no-dependencies"
  name         = "tf-aidial-ci-test-model-no-dependencies"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model"
  description  = "Temporary model created while testing terraform-provider-aidial"
  endpoint     = "https://example.invalid/v1/chat/completions"
  dependencies = []
}
