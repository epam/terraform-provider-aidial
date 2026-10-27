resource "aidial_model" "tf_aidial_integration_model_no_interceptors" {
  bucket       = "platform"
  path         = "tf-aidial-ci-test-model-no-interceptors"
  name         = "tf-aidial-ci-test-model-no-interceptors"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model"
  description  = "Temporary model created while testing terraform-provider-aidial"
  endpoint     = "https://example.invalid/v1/chat/completions"
  interceptors = []
}
