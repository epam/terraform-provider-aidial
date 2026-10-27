variable "aidial_base_url" {
  type = string
}

variable "aidial_api_key" {
  type      = string
  sensitive = true
}

# Used by openai.openai_gpt_4_1_2025_04_14.tf. The stub DIAL Core never calls
# these endpoints, so placeholder values are enough.
variable "aidial_openai_host" {
  type    = string
  default = "http://openai-adapter.example"
}

variable "responses_endpoint" {
  type    = string
  default = "http://openai-adapter.example/openai/v1/responses"
}

# Every region listed here needs a key in cognitive_account_endpoints.
variable "model_regions" {
  type = map(list(string))
  default = {
    "gpt-4.1-2025-04-14" = ["eastus2"]
    "gpt-5.1-2025-11-13" = ["eastus2"]
  }
}

variable "cognitive_account_endpoints" {
  type = map(string)
  default = {
    eastus2 = "https://example-eastus2.openai.azure.com/"
  }
}
