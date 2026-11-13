package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func accToolsetConfig(bucket, path, description string) string {
	return accProviderConfig() + fmt.Sprintf(`
resource "aidial_toolset" "t" {
  bucket      = %[1]q
  path        = %[2]q
  description = %[3]q
  endpoint    = "http://example.invalid/mcp"
  transport   = "HTTP"

  allowed_tools = ["search", "fetch"]

  auth_settings = {
    authentication_type = "API_KEY"
    api_key_header      = "X-Api-Key"
  }

  catalog_properties = {
    vendor = "Acme"
  }
}

data "aidial_toolset" "t" {
  bucket = aidial_toolset.t.bucket
  path   = aidial_toolset.t.path
}
`, bucket, path, description)
}

func TestAccToolset_lifecycle(t *testing.T) {
	path := accPath(t)
	const res, ds = "aidial_toolset.t", "data.aidial_toolset.t"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accToolsetConfig("platform", path, "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "id", "platform/"+path),
					resource.TestCheckResourceAttr(res, "description", "first"),
					resource.TestCheckResourceAttr(res, "transport", "HTTP"),
					resource.TestCheckResourceAttr(res, "allowed_tools.#", "2"),
					resource.TestCheckResourceAttr(res, "auth_settings.authentication_type", "API_KEY"),
					resource.TestCheckResourceAttrSet(res, "reference"),
					resource.TestCheckResourceAttr(res, "max_retry_attempts", "1"),
					resource.TestCheckResourceAttrPair(ds, "id", res, "id"),
					resource.TestCheckResourceAttrPair(ds, "reference", res, "reference"),
					resource.TestCheckResourceAttrPair(ds, "endpoint", res, "endpoint"),
					resource.TestCheckResourceAttr(ds, "auth_settings.api_key_header", "X-Api-Key"),
					resource.TestCheckNoResourceAttr(ds, "auth_settings.client_secret"),
				),
			},
			{
				// An in-place update must keep Core's generated reference.
				Config: accToolsetConfig("platform", path, "second"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "description", "second"),
					resource.TestCheckResourceAttrPair(ds, "reference", res, "reference"),
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

// DIAL Core accepts buckets other than platform for toolsets, and reads a
// path as nested folders.
func TestAccToolset_publicBucketAndNestedPath(t *testing.T) {
	path := "folder/" + accPath(t)
	const res = "aidial_toolset.t"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accToolsetConfig("public", path, "nested"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "id", "public/"+path),
					resource.TestCheckResourceAttr(res, "bucket", "public"),
					// Core reports the author for toolsets outside the platform bucket.
					resource.TestCheckResourceAttrSet(res, "author"),
				),
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

// OAuth settings use the public bucket and a complete OAuth block: otherwise Core probes
// the endpoint when saving, and Core skips that probe for OAuth settings that
// are fully specified.
func TestAccToolset_oauthSecretIsKeptInState(t *testing.T) {
	path := accPath(t)
	const res = "aidial_toolset.t"
	config := accProviderConfig() + fmt.Sprintf(`
resource "aidial_toolset" "t" {
  bucket   = "public"
  path     = %q
  endpoint = "http://example.invalid/mcp"

  auth_settings = {
    authentication_type        = "OAUTH"
    authorization_endpoint     = "http://auth.example.invalid/authorize"
    token_endpoint             = "http://auth.example.invalid/token"
    token_endpoint_auth_method = "client_secret_post"
    client_id                  = "client-id"
    client_secret              = "a-long-client-secret"
    redirect_uri               = "http://redirect.example.invalid/callback"
    code_challenge_method      = "S256"
    dynamically_registered     = false
    resource                   = "http://resource.example.invalid"
    scopes_supported           = ["read"]
  }
}
`, path)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "auth_settings.client_secret", "a-long-client-secret"),
					resource.TestCheckResourceAttr(res, "auth_settings.client_id", "client-id"),
				),
			},
			{
				// A refresh must not lose the secret Core never returns.
				Config:   config,
				PlanOnly: true,
			},
			{
				ResourceName:            res,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"auth_settings.client_secret"},
				ImportStateIdFunc:       func(s *terraform.State) (string, error) { return "public/" + path, nil },
			},
		},
	})
}

func TestAccToolset_outOfBandDeletionIsRecreated(t *testing.T) {
	path := accPath(t)
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: accToolsetConfig("platform", path, "x")},
			{
				PreConfig: func() { accDelete(t, "/v1/toolsets/platform/"+path) },
				Config:    accToolsetConfig("platform", path, "x"),
				Check:     resource.TestCheckResourceAttr("aidial_toolset.t", "id", "platform/"+path),
			},
		},
	})
}
