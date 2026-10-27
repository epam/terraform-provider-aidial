resource "aidial_key" "tf_aidial_integration_key" {
  bucket = var.aidial_role_bucket
  path   = "${var.aidial_role_path}-key"

  # DIAL Core rejects a key value that already exists, so make it unique per
  # run: concurrent pipelines (or a leftover key from a failed run that never
  # reached destroy) would otherwise collide on the same fixed secret.
  key     = "${var.aidial_service_key}-${var.aidial_role_path}"
  project = "tf-dial"
  role    = "developer"
  roles   = ["developer"]
  secured = true

}
