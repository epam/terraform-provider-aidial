package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccFileToolsetDataSource_missing(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{{
			Config: accProviderConfig() + `
data "aidial_file_toolset" "t" {
  name = "tf-acc-does-not-exist"
}
`,
			ExpectError: regexp.MustCompile(`File-sourced toolset not found`),
		}},
	})
}
