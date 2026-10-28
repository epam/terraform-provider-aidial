resource "aidial_catalog_schema" "model" {
  path = "model"

  schema = jsonencode({
    "$id" = "https://dial.epam.com/catalog-schemas/model"
    type  = "object"
    properties = {
      vendor     = { type = "string" }
      parameters = { type = "string" }
    }
  })
}
