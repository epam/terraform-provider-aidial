resource "aidial_model" "tf_aidial_integration_model_all_new_fields" {
  bucket       = "platform"
  path         = "tf-aidial-ci-test-model-all-new-fields"
  name         = "tf-aidial-ci-test-model-all-new-fields"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model"
  description  = "Integration test for model fields"
  endpoint     = "https://example.invalid/v1/chat/completions"

  user_roles             = []
  dependencies           = []
  description_keywords   = []
  input_attachment_types = []
  interceptors           = []
  fields_hashing_order   = ["prefix.body.tools", "prefix.body.messages"]

  features = {
    skills_supported = true
  }

  override_name = "anthropic.tf-aidial-ci-claude-v3-haiku"

  interfaces = {
    anthropicMessages = {
      base_url = "http://bedrock-adapter.example"
    }
  }

  default_headers = {}

  defaults = {
    tools = [
      {
        type = "static_function"

        static_function = {
          name = "web_search"
        }
      }
    ]
  }
}