# Endpoint-based application in the platform bucket, with features, default headers and free-form defaults.
resource "aidial_application" "tf_aidial_integration_application" {
  bucket             = "platform"
  path               = "tf-aidial-ci-test-application"
  description        = "CI application"
  display_name       = "CI application"
  endpoint           = "http://application.example.invalid/chat"
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

data "aidial_application" "tf_aidial_integration_application" {
  bucket = aidial_application.tf_aidial_integration_application.bucket
  path   = aidial_application.tf_aidial_integration_application.path
}

# Minimal application: everything but the endpoint is left to Core's defaults.
resource "aidial_application" "tf_aidial_integration_application_minimal" {
  bucket   = "platform"
  path     = "tf-aidial-ci-test-application-minimal"
  endpoint = "http://application.example.invalid/chat"
}

# Application in the public bucket, in a folder.
resource "aidial_application" "tf_aidial_integration_application_public" {
  bucket   = "public"
  path     = "tf-aidial-ci-test/application-public"
  endpoint = "http://application.example.invalid/chat"
}

# Schema-based application: no endpoint, typed by an application type schema.
resource "aidial_application" "tf_aidial_integration_application_schema" {
  bucket                     = "platform"
  path                       = "tf-aidial-ci-test-application-schema"
  application_type_schema_id = aidial_application_type_schema.tf_aidial_integration_application_type_schema.schema_id

  application_properties = {
    prompt = "hello"
  }
}
