# TEMPORARY probe: model references a catalog schema that does not exist,
# and a second model omits catalog_schema_id entirely.
resource "aidial_model" "tf_aidial_integration_model_catalog_missing_schema" {
  bucket            = "platform"
  path              = "tf-aidial-ci-test-model-catalog-missing"
  name              = "tf-aidial-ci-test-model-catalog-missing"
  type              = "CHAT"
  endpoint          = "https://example.invalid/v1/chat/completions"
  catalog_schema_id = "https://dial.epam.com/catalog-schemas/does-not-exist"
  catalog_properties = {
    vendor = "Vendor"
  }
}

resource "aidial_model" "tf_aidial_integration_model_catalog_no_id" {
  bucket   = "platform"
  path     = "tf-aidial-ci-test-model-catalog-no-id"
  name     = "tf-aidial-ci-test-model-catalog-no-id"
  type     = "CHAT"
  endpoint = "https://example.invalid/v1/chat/completions"
}

# Schema whose path differs from its $id, referenced by an unrelated id above.
resource "aidial_catalog_schema" "tf_aidial_integration_catalog_probe" {
  path   = "tf-aidial-ci-test-catalog-probe"
  schema = jsonencode({ "$id" = "https://dial.epam.com/catalog-schemas/probe-other-id", type = "object" })
}
