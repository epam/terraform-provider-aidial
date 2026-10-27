resource "aidial_model" "tf_aidial_integration_model_no_input_attachment_types" {
  bucket                 = "platform"
  path                   = "tf-aidial-ci-test-model-no-input-attachment-types"
  name                   = "tf-aidial-ci-test-model-no-input-attachment-types"
  type                   = "CHAT"
  display_name           = "tf-dial Terraform Test Model."
  description            = "Temporary model created while testing terraform-provider-aidial"
  endpoint               = "https://example.invalid/v1/chat/completions"
  input_attachment_types = []
}
