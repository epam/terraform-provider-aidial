package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func accAliasConfig(bucket, path, displayName string) string {
	return accProviderConfig() + fmt.Sprintf(`
resource "aidial_application" "target" {
  bucket   = "platform"
  path     = "%[2]s-app"
  endpoint = "http://example.invalid/chat"
}

resource "aidial_alias" "a" {
  bucket      = %[1]q
  path        = %[2]q
  present_as  = "model"
  target_path = "applications/${aidial_application.target.path}"

  display_name         = %[3]q
  description          = "Acc alias"
  intro                = "hello"
  user_roles           = ["admin"]
  description_keywords = ["acc", "alias"]
}

data "aidial_alias" "a" {
  bucket = aidial_alias.a.bucket
  path   = aidial_alias.a.path
}
`, bucket, path, displayName)
}

func TestAccAlias_lifecycle(t *testing.T) {
	path := accPath(t)
	const res, ds = "aidial_alias.a", "data.aidial_alias.a"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accAliasConfig("public", path, "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "id", "public/"+path),
					resource.TestCheckResourceAttr(res, "present_as", "model"),
					resource.TestCheckResourceAttr(res, "target_path", "applications/"+path+"-app"),
					resource.TestCheckResourceAttr(res, "display_name", "first"),
					resource.TestCheckResourceAttr(res, "user_roles.#", "1"),
					resource.TestCheckResourceAttr(res, "description_keywords.#", "2"),
					resource.TestCheckResourceAttrPair(ds, "id", res, "id"),
					resource.TestCheckResourceAttrPair(ds, "target_path", res, "target_path"),
					resource.TestCheckResourceAttrPair(ds, "intro", res, "intro"),
				),
			},
			{
				Config: accAliasConfig("public", path, "second"),
				Check:  resource.TestCheckResourceAttr(res, "display_name", "second"),
			},
			{
				ResourceName:      res,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) { return "public/" + path, nil },
			},
		},
	})
}

func TestAccAlias_platformBucket(t *testing.T) {
	path := accPath(t)
	const res = "aidial_alias.a"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accAliasConfig("platform", path, "p"),
				Check:  resource.TestCheckResourceAttr(res, "id", "platform/"+path),
			},
		},
	})
}

func TestAccAlias_minimal(t *testing.T) {
	path := accPath(t)
	const res = "aidial_alias.a"
	config := accProviderConfig() + fmt.Sprintf(`
resource "aidial_alias" "a" {
  bucket      = "public"
  path        = %q
  present_as  = "application"
  target_path = "models/some-model"
}
`, path)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "present_as", "application"),
					resource.TestCheckNoResourceAttr(res, "user_roles.#"),
					resource.TestCheckNoResourceAttr(res, "description_keywords.#"),
				),
			},
			{Config: config, PlanOnly: true}, // a re-plan shows no change
		},
	})
}

func TestAccAlias_invalidCombinationFailsAtPlan(t *testing.T) {
	path := accPath(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accProviderConfig() + fmt.Sprintf(`
resource "aidial_alias" "a" {
  bucket      = "public"
  path        = %q
  present_as  = "toolset"
  target_path = "applications/x"
}
`, path),
				ExpectError: regexp.MustCompile(`must point at a toolset`),
			},
		},
	})
}
