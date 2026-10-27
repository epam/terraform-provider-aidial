# DIAL Core treats null (not set) and empty ("" / []) as different values.
# Unset attributes must stay null, explicit empty values must stay empty.
# drift_check_null_vs_empty.sh changes this model outside Terraform and
# checks that the next plan detects it.
resource "aidial_model" "tf_aidial_integration_model_null_vs_empty" {
  bucket       = "platform"
  path         = "tf-aidial-ci-test-model-null-vs-empty"
  name         = "tf-aidial-ci-test-model-null-vs-empty"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model"
  endpoint     = "https://example.invalid/v1/chat/completions"

  # Explicitly empty: sent as "" and [].
  icon_url   = ""
  user_roles = []

  # Not set: description, display_version, input_attachment_types stay null.
}

check "model_null_vs_empty" {
  assert {
    condition     = aidial_model.tf_aidial_integration_model_null_vs_empty.description == null
    error_message = "description is not set and must stay null."
  }
  assert {
    condition     = aidial_model.tf_aidial_integration_model_null_vs_empty.display_version == null
    error_message = "display_version is not set and must stay null."
  }
  assert {
    condition     = aidial_model.tf_aidial_integration_model_null_vs_empty.input_attachment_types == null
    error_message = "input_attachment_types is not set and must stay null, not []."
  }
  assert {
    condition     = aidial_model.tf_aidial_integration_model_null_vs_empty.icon_url == ""
    error_message = "icon_url is set to \"\" and must stay \"\", not null."
  }
  assert {
    condition     = aidial_model.tf_aidial_integration_model_null_vs_empty.user_roles != null && length(aidial_model.tf_aidial_integration_model_null_vs_empty.user_roles) == 0
    error_message = "user_roles is set to [] and must stay [], not null."
  }
}
