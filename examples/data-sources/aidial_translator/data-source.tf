# A translator stored in DIAL Core resource storage, for example one managed
# elsewhere with the aidial_translator resource.
data "aidial_translator" "claude" {
  bucket = "platform"
  path   = "openai-to-claude"
}

output "translator_base_url" {
  value = data.aidial_translator.claude.base_url
}
