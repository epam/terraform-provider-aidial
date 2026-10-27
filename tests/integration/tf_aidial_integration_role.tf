resource "aidial_role" "tf_aidial_integration_role" {
  bucket = var.aidial_role_bucket
  path   = var.aidial_role_path
  name   = "roles/${var.aidial_role_bucket}/${var.aidial_role_path}"

  limits = {
    "models/${aidial_model.tf_aidial_integration_model_full.bucket}/${aidial_model.tf_aidial_integration_model_full.path}" = jsonencode({
      minute      = 1
      day         = 1
      week        = 1
      month       = 1
      requestHour = 1
      requestDay  = 1
    })
  }
}
