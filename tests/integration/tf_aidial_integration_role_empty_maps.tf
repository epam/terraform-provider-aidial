# Role with explicitly empty limits and share: DIAL Core omits empty objects,
# so the configured {} must be kept instead of becoming null.
resource "aidial_role" "tf_aidial_integration_role_empty_maps" {
  bucket = var.aidial_role_bucket
  path   = "${var.aidial_role_path}-empty-maps"
  name   = "roles/${var.aidial_role_bucket}/${var.aidial_role_path}-empty-maps"

  limits = {}
  share  = {}
}
