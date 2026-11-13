# An alias that is invoked as "fast-chat" and points at an application.
resource "aidial_alias" "example" {
  bucket      = "public"
  path        = "fast-chat"
  present_as  = "model"
  target_path = "applications/my-application"

  display_name         = "Fast chat"
  description          = "Short name for the chat application"
  user_roles           = ["admin"]
  description_keywords = ["chat", "fast"]
}
