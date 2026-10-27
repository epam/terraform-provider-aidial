# bucket is set explicitly, path is omitted: path falls back to name.
resource "aidial_model" "tf_aidial_integration_model_path_omitted" {
  bucket       = var.aidial_role_bucket
  name         = "tf-aidial-ci-test-model-path-omitted"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model"
  endpoint     = "https://example.invalid/v1/chat/completions"
}

check "model_path_default" {
  assert {
    condition     = aidial_model.tf_aidial_integration_model_path_omitted.bucket == var.aidial_role_bucket
    error_message = "an explicit bucket must not be overridden by the default."
  }
  assert {
    condition     = aidial_model.tf_aidial_integration_model_path_omitted.path == "tf-aidial-ci-test-model-path-omitted"
    error_message = "path must default to name."
  }
}
