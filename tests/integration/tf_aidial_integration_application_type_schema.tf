resource "aidial_application_type_schema" "tf_aidial_integration_application_type_schema" {
  bucket = "platform"
  path   = "tf-aidial-ci-test-application-type-schema"

  schema = jsonencode({
    "$id" = "https://dial.epam.com/application_type_schemas/tf-aidial-ci-test"
    type  = "object"
    properties = {
      prompt = { type = "string" }
      temperature = {
        type    = "number"
        default = 0.5
      }
    }
  })
}

data "aidial_application_type_schema" "tf_aidial_integration_application_type_schema" {
  bucket = aidial_application_type_schema.tf_aidial_integration_application_type_schema.bucket
  path   = aidial_application_type_schema.tf_aidial_integration_application_type_schema.path
}
