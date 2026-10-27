resource "aidial_model" "tf_aidial_integration_model_no_description_keywords" {
  bucket               = "platform"
  path                 = "tf-aidial-ci-test-model-no-description-keywords"
  name                 = "tf-aidial-ci-test-model-no-description-keywords"
  type                 = "CHAT"
  display_name         = "tf-dial Terraform Test Model"
  description          = "Temporary model created while testing terraform-provider-aidial"
  endpoint             = "https://example.invalid/v1/chat/completions"
  description_keywords = []
}
