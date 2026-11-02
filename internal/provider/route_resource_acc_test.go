package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRoute_basic(t *testing.T) {
	path := accPath(t)
	cfg := func(weight int) string {
		return accProviderConfig() + fmt.Sprintf(`
resource "aidial_route" "t" {
  bucket = "platform"
  path    = %q
  name    = "routes/platform/%s"
  methods = ["GET"]
  paths   = ["/%s/.*"]
  upstreams = [
    {
      endpoint = "https://example.invalid/v1/chat/completions"
      tier     = 0
      weight   = %d
    }
  ]
}
`, path, path, path, weight)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg(1)},
			accImportStep("aidial_route.t"),
			{Config: cfg(2)},
		},
	})
}

func TestAccRoute_nameOmitted(t *testing.T) {
	path := accPath(t)
	cfg := accProviderConfig() + fmt.Sprintf(`
resource "aidial_route" "t" {
  bucket  = "platform"
  path    = %q
  methods = ["GET"]
  paths   = ["/%s/.*"]
  upstreams = [
    {
      endpoint = "https://example.invalid/v1/chat/completions"
      tier     = 0
      weight   = 1
    }
  ]
}
`, path, path)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.TestCheckResourceAttrPair("aidial_route.t", "name", "aidial_route.t", "api_name")},
			{Config: cfg, PlanOnly: true},
		},
	})
}
