package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func accInterceptorConfig(path, description string) string {
	return accProviderConfig() + fmt.Sprintf(`
resource "aidial_interceptor" "i" {
  bucket       = "platform"
  path         = %[1]q
  description  = %[2]q
  endpoint     = "http://example.invalid/intercept"
  display_name = "Acc interceptor"
  user_roles   = ["admin"]

  default_headers = {
    "X-Team" = "acc"
  }

  features = {
    tools_supported = true
  }

  defaults = {
    timeout = 30
  }
}

data "aidial_interceptor" "i" {
  bucket = aidial_interceptor.i.bucket
  path   = aidial_interceptor.i.path
}
`, path, description)
}

func TestAccInterceptor_lifecycle(t *testing.T) {
	path := accPath(t)
	const res, ds = "aidial_interceptor.i", "data.aidial_interceptor.i"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accInterceptorConfig(path, "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "id", "platform/"+path),
					resource.TestCheckResourceAttr(res, "description", "first"),
					resource.TestCheckResourceAttr(res, "endpoint", "http://example.invalid/intercept"),
					resource.TestCheckResourceAttr(res, "forward_auth_token", "false"),
					resource.TestCheckResourceAttr(res, "max_retry_attempts", "1"),
					resource.TestCheckResourceAttr(res, "user_roles.#", "1"),
					resource.TestCheckResourceAttr(res, "features.tools_supported", "true"),
					resource.TestCheckResourceAttrPair(ds, "id", res, "id"),
					resource.TestCheckResourceAttrPair(ds, "endpoint", res, "endpoint"),
					resource.TestCheckResourceAttrPair(ds, "display_name", res, "display_name"),
					resource.TestCheckResourceAttr(ds, "default_headers.X-Team", "acc"),
				),
			},
			{
				Config: accInterceptorConfig(path, "second"),
				Check:  resource.TestCheckResourceAttr(res, "description", "second"),
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

func TestAccInterceptor_minimal(t *testing.T) {
	path := accPath(t)
	const res = "aidial_interceptor.i"
	config := accProviderConfig() + fmt.Sprintf(`
resource "aidial_interceptor" "i" {
  bucket   = "platform"
  path     = %q
  endpoint = "http://example.invalid/intercept"
}
`, path)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "forward_auth_token", "false"),
					resource.TestCheckResourceAttr(res, "max_retry_attempts", "1"),
					resource.TestCheckNoResourceAttr(res, "user_roles.#"),
				),
			},
			{Config: config, PlanOnly: true}, // a re-plan shows no change
		},
	})
}
