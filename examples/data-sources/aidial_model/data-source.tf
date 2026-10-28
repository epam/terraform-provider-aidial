# A model stored in DIAL Core resource storage (for example, one created with
# the aidial_model resource in another workspace). bucket defaults to
# "platform" and path defaults to name.
data "aidial_model" "gpt4o" {
  name = "gpt-4o"
}

output "gpt4o_endpoint" {
  value = data.aidial_model.gpt4o.endpoint
}
