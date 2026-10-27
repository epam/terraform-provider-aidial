resource "aidial_model" "tf_aidial_integration_model_no_user_roles" {
  bucket       = "platform"
  path         = "tf-aidial-ci-test-model-no-user-roles"
  name         = "tf-aidial-ci-test-model-no-user-roles"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model"
  description  = "Temporary model created while testing terraform-provider-aidial"
  endpoint     = "https://example.invalid/v1/chat/completions"
  user_roles   = []
}
