data "aidial_bucket" "current" {}

resource "aidial_model" "example" {
  bucket          = data.aidial_bucket.current.bucket
  path            = "my-model"
  name            = "my-model"
  type            = "CHAT"
  display_name    = "My Model"
  display_version = "latest"
  description     = "Example chat model served through an OpenAI-compatible adapter."
  override_name   = "gpt-example"

  catalog_schema_id = "https://dial.epam.com/catalog-schemas/model"
  catalog_properties = {
    provider            = "Provider"
    vendor              = "Vendor"
    license             = "License"
    knowledgeCutoffDate = "2026-08-17"
    parameters          = "100B"
  }

  endpoint = "https://adapter.example.com/openai/deployments/gpt-example/chat/completions"

  input_attachment_types = ["image/*", "application/pdf"]
  description_keywords   = ["Text Generation", "Image Recognition"]
  user_roles             = ["default", "user"]

  features = {
    system_prompt_supported = true
    tools_supported         = true
    temperature_supported   = true
  }

  limits = {
    max_prompt_tokens     = 200000
    max_completion_tokens = 32000
  }

  pricing = {
    unit       = "token"
    prompt     = "0.000002"
    completion = "0.000008"
  }

  upstreams = [
    {
      id       = "primary"
      endpoint = "https://adapter-eu.example.com/openai/deployments/gpt-example/chat/completions"
      weight   = 1
      tier     = 0
    },
    {
      id       = "secondary"
      endpoint = "https://adapter-us.example.com/openai/deployments/gpt-example/chat/completions"
      weight   = 1
      tier     = 1
    },
  ]
}
