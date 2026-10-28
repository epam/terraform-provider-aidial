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
