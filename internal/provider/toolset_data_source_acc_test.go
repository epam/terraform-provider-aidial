package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccToolsetDataSource_missing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{{
			Config: accProviderConfig() + `
data "aidial_toolset" "t" {
  bucket = "platform"
  path   = "tf-acc-does-not-exist"
}
`,
			ExpectError: regexp.MustCompile(`Toolset not found`),
		}},
	})
}
