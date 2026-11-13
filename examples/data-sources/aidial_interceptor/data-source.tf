# An interceptor stored in DIAL Core resource storage, for example one managed
# elsewhere with the aidial_interceptor resource.
data "aidial_interceptor" "filter" {
  bucket = "platform"
  path   = "my-interceptor"
}

output "interceptor_endpoint" {
  value = data.aidial_interceptor.filter.endpoint
}
