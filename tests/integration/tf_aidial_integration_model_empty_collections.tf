resource "aidial_model" "tf_aidial_integration_model_empty_collections" {
  bucket       = "platform"
  path         = "tf-aidial-ci-test-model-empty-collections"
  name         = "tf-aidial-ci-test-model-empty-collections"
  type         = "CHAT"
  display_name = "tf-dial Terraform Test Model"
  endpoint     = "https://example.invalid/v1/chat/completions"

  user_roles             = []
  dependencies           = []
  description_keywords   = []
  input_attachment_types = []
  interceptors           = []
  upstreams              = []

  interfaces      = {}
  default_headers = {}
}