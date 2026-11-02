# A model stored in DIAL Core resource storage (for example, one created with
# the aidial_model resource in another workspace). bucket and path are the
# identity it was created with.
data "aidial_model" "gpt4o" {
  name   = "gpt-4o"
  bucket = "platform"
  path   = "gpt-4o"
}

output "gpt4o_endpoint" {
  value = data.aidial_model.gpt4o.endpoint
}
