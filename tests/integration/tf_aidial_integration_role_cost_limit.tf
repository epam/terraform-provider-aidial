# Role with cost limits and a per-model limit, share omitted: DIAL Core may
# read back "share": {}, which must not turn the unset attribute into {}.
resource "aidial_role" "tf_aidial_integration_role_cost_limit" {
  bucket = var.aidial_role_bucket
  path   = "${var.aidial_role_path}-cost-limit"
  name   = "roles/${var.aidial_role_bucket}/${var.aidial_role_path}-cost-limit"

  cost_limit = {
    minute = 123
    day    = 12.2
    week   = 22.8
  }

  limits = {
    "models/${aidial_model.tf_aidial_integration_model_full.bucket}/${aidial_model.tf_aidial_integration_model_full.path}" = jsonencode({
      day = 10
    })
  }
}

check "role_cost_limit" {
  assert {
    condition = (
      aidial_role.tf_aidial_integration_role_cost_limit.cost_limit.minute == 123 &&
      aidial_role.tf_aidial_integration_role_cost_limit.cost_limit.day == 12.2 &&
      aidial_role.tf_aidial_integration_role_cost_limit.cost_limit.week == 22.8
    )
    error_message = "cost limits must round-trip unchanged."
  }
  assert {
    condition     = aidial_role.tf_aidial_integration_role_cost_limit.cost_limit.month == null
    error_message = "cost_limit.month was not set and must stay null."
  }
  assert {
    condition     = aidial_role.tf_aidial_integration_role_cost_limit.share == null
    error_message = "share was omitted and must stay null."
  }
  assert {
    condition     = aidial_role.tf_aidial_integration_role_cost_limit.status != ""
    error_message = "status must be populated (read from _metadata)."
  }
}
