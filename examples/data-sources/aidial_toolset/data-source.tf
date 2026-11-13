# A toolset stored in DIAL Core resource storage, for example one managed
# elsewhere with the aidial_toolset resource.
data "aidial_toolset" "search" {
  bucket = "platform"
  path   = "my-toolset"
}

output "toolset_endpoint" {
  value = data.aidial_toolset.search.endpoint
}
