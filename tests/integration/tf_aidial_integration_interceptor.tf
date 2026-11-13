# Interceptor in the platform bucket, with features, default headers and free-form defaults.
resource "aidial_interceptor" "tf_aidial_integration_interceptor" {
  bucket             = "platform"
  path               = "tf-aidial-ci-test-interceptor"
  description        = "CI interceptor"
  display_name       = "CI interceptor"
  endpoint           = "http://interceptor.example.invalid/intercept"
  user_roles         = ["ci-role"]
  max_retry_attempts = 2

  default_headers = {
    "X-Team" = "ci"
  }

  features = {
    tools_supported = true
  }

  defaults = {
    timeout = 30
  }

  catalog_properties = {
    vendor = "Acme"
  }
}

data "aidial_interceptor" "tf_aidial_integration_interceptor" {
  bucket = aidial_interceptor.tf_aidial_integration_interceptor.bucket
  path   = aidial_interceptor.tf_aidial_integration_interceptor.path
}

# Minimal interceptor: everything but the endpoint is left to Core's defaults.
resource "aidial_interceptor" "tf_aidial_integration_interceptor_minimal" {
  bucket   = "platform"
  path     = "tf-aidial-ci-test-interceptor-minimal"
  endpoint = "http://interceptor.example.invalid/intercept"
}
