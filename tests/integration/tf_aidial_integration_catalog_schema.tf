resource "aidial_catalog_schema" "tf_aidial_integration_catalog_schema" {
  bucket = "platform"
  path   = "tf-aidial-ci-test-catalog-schema"

  schema = jsonencode({
    "$id" = "https://dial.epam.com/catalog-schemas/tf-aidial-ci-test"
    type  = "object"
    properties = {
      provider            = { type = "string" }
      vendor              = { type = "string" }
      license             = { type = "string" }
      knowledgeCutoffDate = { type = "string" }
      parameters          = { type = "string" }
    }
  })
}

resource "aidial_model" "tf_aidial_integration_model_catalog" {
  bucket       = "platform"
  path         = "tf-aidial-ci-test-model-catalog"
  name         = "tf-aidial-ci-test-model-catalog"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model"
  description  = "Integration test for catalog_schema_id and catalog_properties."
  endpoint     = "https://example.invalid/v1/chat/completions"

  catalog_schema_id = aidial_catalog_schema.tf_aidial_integration_catalog_schema.schema_id
  catalog_properties = {
    provider            = "Provider"
    vendor              = "Vendor"
    license             = "License"
    knowledgeCutoffDate = "2026-08-17"
    parameters          = "100B"
  }
}

data "aidial_catalog_schema" "tf_aidial_integration_catalog_schema" {
  bucket = aidial_catalog_schema.tf_aidial_integration_catalog_schema.bucket
  path   = aidial_catalog_schema.tf_aidial_integration_catalog_schema.path
}
