package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func accTranslatorConfig(path, baseURL, out string) string {
	return accProviderConfig() + fmt.Sprintf(`
resource "aidial_translator" "t" {
  bucket   = "platform"
  path     = %[1]q
  base_url = %[2]q
  in       = "openaiChatCompletions"
  out      = %[3]q
}

data "aidial_translator" "t" {
  bucket = aidial_translator.t.bucket
  path   = aidial_translator.t.path
}
`, path, baseURL, out)
}

func TestAccTranslator_lifecycle(t *testing.T) {
	path := accPath(t)
	const res, ds = "aidial_translator.t", "data.aidial_translator.t"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accTranslatorConfig(path, "http://example.invalid:8000", "anthropicMessages"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "id", "platform/"+path),
					resource.TestCheckResourceAttr(res, "base_url", "http://example.invalid:8000"),
					resource.TestCheckResourceAttr(res, "in", "openaiChatCompletions"),
					resource.TestCheckResourceAttr(res, "out", "anthropicMessages"),
					resource.TestCheckResourceAttrPair(ds, "id", res, "id"),
					resource.TestCheckResourceAttrPair(ds, "base_url", res, "base_url"),
					resource.TestCheckResourceAttrPair(ds, "in", res, "in"),
					resource.TestCheckResourceAttrPair(ds, "out", res, "out"),
				),
			},
			{
				Config: accTranslatorConfig(path, "http://other.invalid:9000", "openaiResponses"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "base_url", "http://other.invalid:9000"),
					resource.TestCheckResourceAttr(res, "out", "openaiResponses"),
				),
			},
			{
				ResourceName:      res,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) { return "platform/" + path, nil },
			},
		},
	})
}
