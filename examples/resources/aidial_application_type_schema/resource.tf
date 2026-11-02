resource "aidial_application_type_schema" "quick" {
  bucket = "platform"
  path   = "quick"

  schema = jsonencode({
    "$id" = "https://dial.epam.com/application_type_schemas/quick"
    type  = "object"
    properties = {
      prompt = { type = "string" }
    }
  })
}
