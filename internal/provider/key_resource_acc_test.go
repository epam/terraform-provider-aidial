package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKey_basic(t *testing.T) {
	path := accPath(t)
	cfg := fmt.Sprintf(`%s
resource "aidial_key" "t" {
  path    = %q
  key     = "secret-%s"
  project = "tf-acc"
  role    = "default"
  roles   = ["default"]
}
`, accProviderConfig(), path, path)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.TestCheckResourceAttr("aidial_key.t", "project", "tf-acc")},
			// GET never returns the secret, so it cannot be verified after import.
			{ResourceName: "aidial_key.t", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"key"}},
		},
	})
}
