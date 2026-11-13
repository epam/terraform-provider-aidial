# Application the aliases point at.
resource "aidial_application" "tf_aidial_integration_alias_target" {
  bucket   = "platform"
  path     = "tf-aidial-ci-test-alias-target"
  endpoint = "http://application.example.invalid/chat"
}

# Alias in the public bucket, shown as a model, with every optional attribute.
resource "aidial_alias" "tf_aidial_integration_alias" {
  bucket      = "public"
  path        = "tf-aidial-ci-test-alias"
  present_as  = "model"
  target_path = "applications/${aidial_application.tf_aidial_integration_alias_target.path}"

  display_name         = "CI alias"
  description          = "CI alias"
  display_version      = "1.0"
  icon_url             = "http://icon.example.invalid/alias.png"
  intro                = "Hello from the CI alias"
  user_roles           = ["ci-role"]
  description_keywords = ["ci", "alias"]
}

data "aidial_alias" "tf_aidial_integration_alias" {
  bucket = aidial_alias.tf_aidial_integration_alias.bucket
  path   = aidial_alias.tf_aidial_integration_alias.path
}

# Minimal alias in the platform bucket: only what Core requires.
resource "aidial_alias" "tf_aidial_integration_alias_minimal" {
  bucket      = "platform"
  path        = "tf-aidial-ci-test-alias-minimal"
  present_as  = "application"
  target_path = "models/tf-aidial-ci-test-model"
}
