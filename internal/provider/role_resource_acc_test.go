package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRole_basic(t *testing.T) {
	path := accPath(t)
	cfg := func(day int) string {
		return accProviderConfig() + fmt.Sprintf(`
resource "aidial_role" "t" {
  path = %q
  name = "roles/platform/%s"
  limits = {
    "models/platform/acc" = jsonencode({ day = %d })
  }
}
`, path, path, day)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg(1), Check: resource.TestCheckResourceAttrSet("aidial_role.t", "limits.models/platform/acc")},
			// Import has no configured JSON/name to preserve: Core returns the limits with defaults filled in.
			{ResourceName: "aidial_role.t", ImportState: true, ImportStateVerify: true, ImportStateVerifyIgnore: []string{"limits.%", "limits.models/platform/acc", "name"}},
			{Config: cfg(2), Check: resource.TestCheckResourceAttrSet("aidial_role.t", "limits.models/platform/acc")},
		},
	})
}

// A resource deleted outside Terraform is absent (GET 404), so the next plan recreates it.
func TestAccRole_deletedOutsideTerraform(t *testing.T) {
	path := accPath(t)
	cfg := accProviderConfig() + fmt.Sprintf(`
resource "aidial_role" "t" {
  path = %q
  name = "roles/platform/%s"
}
`, path, path)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg},
			{
				PreConfig:          func() { accDelete(t, "/v1/roles/platform/"+path) },
				Config:             cfg,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
