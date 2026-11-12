terraform {
  required_providers {
    aidial = {
      source = "epam/aidial"
    }
  }
}

# Authenticates only with an OIDC ID token; no api_key is set anywhere.
provider "aidial" {
  base_url = var.aidial_base_url
  id_token = var.aidial_id_token
}
