package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
)

func TestAccModel_basic(t *testing.T) {
	path := accPath(t)
	cfg := func(desc string) string {
		return accProviderConfig() + fmt.Sprintf(`
resource "aidial_model" "t" {
  bucket = "platform"
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

// bucket and path are the whole identity: changing only name updates in place.
func TestAccModel_nameChangeUpdatesInPlace(t *testing.T) {
	path := accPath(t)
	cfg := func(name string) string {
		return accProviderConfig() + fmt.Sprintf(`
resource "aidial_model" "t" {
  bucket       = "platform"
  path         = %q
  name         = %q
  type         = "CHAT"
  display_name = "acc"
  endpoint     = "https://example.invalid/v1/chat/completions"
}
`, path, name)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg(path)},
			{
				Config: cfg(path + "-renamed"),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("aidial_model.t", plancheck.ResourceActionUpdate)},
				},
			},
		},
	})
}

// name is optional: DIAL Core derives it from path, so omitting it must plan cleanly
// and adding it later is an in-place update.
func TestAccModel_nameOmitted(t *testing.T) {
	path := accPath(t)
	base := accProviderConfig() + fmt.Sprintf(`
resource "aidial_model" "t" {
  bucket       = "platform"
  path         = %q
  type         = "CHAT"
  display_name = "acc"
  endpoint     = "https://example.invalid/v1/chat/completions"
`, path)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: base + "}\n", Check: resource.TestCheckResourceAttrPair("aidial_model.t", "name", "aidial_model.t", "api_name")},
			{Config: base + "}\n", PlanOnly: true},
			{
				Config: base + "  name = \"custom\"\n}\n",
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction("aidial_model.t", plancheck.ResourceActionUpdate)},
				},
				Check: resource.TestCheckResourceAttr("aidial_model.t", "name", "custom"),
			},
		},
	})
}

// Flat and conditional pricing rates survive a round trip through Core without drift.
func TestAccModel_conditionalPricing(t *testing.T) {
	path := accPath(t)
	cfg := func(prompt string) string {
		return accProviderConfig() + fmt.Sprintf(`
resource "aidial_model" "t" {
  bucket   = "platform"
  path     = %q
  type     = "CHAT"
  endpoint = "https://example.invalid/v1/chat/completions"
  pricing = {
    unit       = "token"
    prompt     = %s
    completion = { rate = "0.000008" }
    cache_read = {
      test     = { field = "promptTokens", operator = ">", value = "100000" }
      if_true  = "0.00000005"
      if_false = "0.00000001"
    }
  }
}
`, path, prompt)
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: cfg(`{ rate = "0.000002" }`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("aidial_model.t", "pricing.prompt.rate", "0.000002"),
					resource.TestCheckResourceAttr("aidial_model.t", "pricing.completion.rate", "0.000008"),
					resource.TestCheckResourceAttr("aidial_model.t", "pricing.cache_read.test.operator", ">"),
					resource.TestCheckResourceAttr("aidial_model.t", "pricing.cache_read.test.value", "100000"),
					resource.TestCheckResourceAttr("aidial_model.t", "pricing.cache_read.if_true", "0.00000005"),
					resource.TestCheckResourceAttr("aidial_model.t", "pricing.cache_read.if_false", "0.00000001"),
				),
			},
			accImportStep("aidial_model.t"),
			{
				Config: cfg(`{
      test     = { field = "promptTokens", operator = ">", value = "100000" }
      if_true  = "0.0000005"
      if_false = "0.0000001"
    }`),
				Check: resource.TestCheckResourceAttr("aidial_model.t", "pricing.prompt.if_true", "0.0000005"),
			},
		},
	})
}
