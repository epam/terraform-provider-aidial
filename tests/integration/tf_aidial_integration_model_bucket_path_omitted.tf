# Neither bucket nor path is set: bucket falls back to "platform" and path
# falls back to name.
resource "aidial_model" "tf_aidial_integration_model_bucket_path_omitted" {
  name         = "tf-aidial-ci-test-model-bucket-path-omitted"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model"
  endpoint     = "https://example.invalid/v1/chat/completions"
}

check "model_bucket_path_defaults" {
  assert {
    condition     = aidial_model.tf_aidial_integration_model_bucket_path_omitted.bucket == "platform"
    error_message = "bucket must default to \"platform\"."
  }
  assert {
    condition     = aidial_model.tf_aidial_integration_model_bucket_path_omitted.path == "tf-aidial-ci-test-model-bucket-path-omitted"
    error_message = "path must default to name."
  }
  assert {
    condition     = aidial_model.tf_aidial_integration_model_bucket_path_omitted.id == "platform/tf-aidial-ci-test-model-bucket-path-omitted"
    error_message = "id must be <bucket>/<path> built from the defaults."
  }
}
