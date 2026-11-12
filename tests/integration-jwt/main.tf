# Smallest manifest that proves create, read and destroy work under
# "Authorization: Bearer". The full lifecycle runs in tests/integration.
resource "aidial_role" "jwt_check" {
  bucket = var.aidial_role_bucket
  path   = var.aidial_role_path
}

data "aidial_role" "jwt_check" {
  bucket = aidial_role.jwt_check.bucket
  path   = aidial_role.jwt_check.path
}

output "jwt_check_role_id" {
  value = data.aidial_role.jwt_check.id
}
