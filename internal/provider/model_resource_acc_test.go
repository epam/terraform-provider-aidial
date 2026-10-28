package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccModel_basic(t *testing.T) {
	path := accPath(t)
	cfg := func(desc string) string {
		return accProviderConfig() + fmt.Sprintf(`
resource "aidial_model" "t" {
  path         = %q
  name         = %q
  type         = "CHAT"
  display_name = "acc"
  description  = %q
  endpoint     = "https://example.invalid/v1/chat/completions"
}
`, path, path, desc)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg("one"), Check: resource.TestCheckResourceAttr("aidial_model.t", "description", "one")},
			accImportStep("aidial_model.t"),
			{Config: cfg("two"), Check: resource.TestCheckResourceAttr("aidial_model.t", "description", "two")},
		},
	})
}
