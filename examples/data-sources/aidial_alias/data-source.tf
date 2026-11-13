# An alias stored in DIAL Core resource storage, for example one managed
# elsewhere with the aidial_alias resource.
data "aidial_alias" "fast_chat" {
  bucket = "public"
  path   = "fast-chat"
}

output "alias_target" {
  value = data.aidial_alias.fast_chat.target_path
}
