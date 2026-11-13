# An application stored in DIAL Core resource storage, for example one managed
# elsewhere with the aidial_application resource.
data "aidial_application" "assistant" {
  bucket = "platform"
  path   = "my-application"
}

output "application_endpoint" {
  value = data.aidial_application.assistant.endpoint
}
