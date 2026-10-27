# Neither bucket nor path is set on a role.
resource "aidial_role" "tf_aidial_integration_role_bucket_path_omitted" {
  name = "tf-aidial-ci-${var.aidial_role_path}-defaults"
}

check "role_bucket_path_defaults" {
  assert {
    condition     = aidial_role.tf_aidial_integration_role_bucket_path_omitted.bucket == "platform"
    error_message = "bucket must default to \"platform\"."
  }
  assert {
    condition     = aidial_role.tf_aidial_integration_role_bucket_path_omitted.path == "tf-aidial-ci-${var.aidial_role_path}-defaults"
    error_message = "path must default to name."
  }
}
