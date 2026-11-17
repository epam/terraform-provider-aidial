package provider

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func accApplicationConfig(path, description string) string {
	return accProviderConfig() + fmt.Sprintf(`
resource "aidial_application" "i" {
  bucket       = "platform"
  path         = %[1]q
  description  = %[2]q
  endpoint     = "http://example.invalid/intercept"
  display_name = "Acc application"
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

data "aidial_application" "i" {
  bucket = aidial_application.i.bucket
  path   = aidial_application.i.path
}
`, path, description)
}

func TestAccApplication_lifecycle(t *testing.T) {
	path := accPath(t)
	const res, ds = "aidial_application.i", "data.aidial_application.i"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accApplicationConfig(path, "first"),
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
				Config: accApplicationConfig(path, "second"),
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

func TestAccApplication_minimal(t *testing.T) {
	path := accPath(t)
	const res = "aidial_application.i"
	config := accProviderConfig() + fmt.Sprintf(`
resource "aidial_application" "i" {
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

func TestAccApplication_schemaBased(t *testing.T) {
	path := accPath(t)
	const res = "aidial_application.i"
	config := accProviderConfig() + fmt.Sprintf(`
resource "aidial_application_type_schema" "t" {
  bucket = "platform"
  path   = %[1]q

  schema = jsonencode({
    "$id" = "https://dial.epam.com/application_type_schemas/%[1]s"
    type  = "object"
    properties = {
      prompt = { type = "string" }
    }
  })
}

resource "aidial_application" "i" {
  bucket                     = "platform"
  path                       = %[1]q
  application_type_schema_id = aidial_application_type_schema.t.schema_id
  application_properties = {
    prompt = "hello"
  }
  interceptors = ["acc-interceptor"]
}
`, path)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "application_properties.prompt", "hello"),
					resource.TestCheckResourceAttr(res, "interceptors.#", "1"),
					resource.TestCheckNoResourceAttr(res, "endpoint"),
				),
			},
			{Config: config, PlanOnly: true},
		},
	})
}

func TestAccApplication_publicBucket(t *testing.T) {
	path := "tf-acc/" + accPath(t)
	const res = "aidial_application.i"
	config := accProviderConfig() + fmt.Sprintf(`
resource "aidial_application" "i" {
  bucket   = "public"
  path     = %q
  endpoint = "http://example.invalid/chat"
}
`, path)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: config, Check: resource.TestCheckResourceAttr(res, "id", "public/"+path)},
			{Config: config, PlanOnly: true},
			{
				ResourceName:      res,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) { return "public/" + path, nil },
			},
		},
	})
}

func TestAccApplication_endpointXorSchema(t *testing.T) {
	cfg := func(body string) string {
		return accProviderConfig() + "resource \"aidial_application\" \"i\" {\n  bucket = \"platform\"\n  path   = \"tf-acc-xor\"\n" + body + "}\n"
	}
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg(""), PlanOnly: true, ExpectError: regexp.MustCompile(`nothing to run`)},
			{
				Config:      cfg("  endpoint = \"http://e\"\n  application_type_schema_id = \"https://s/x\"\n"),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`cannot be set together`),
			},
		},
	})
}

func accApplicationFullConfig(path, secret string) string {
	return accProviderConfig() + fmt.Sprintf(`
resource "aidial_application" "i" {
  bucket   = "platform"
  path     = %[1]q
  endpoint = "http://example.invalid/chat"

  input_attachment_types       = ["*/*"]
  description_keywords         = ["acc", "nested"]
  dependencies                 = ["acc-model"]
  max_input_attachments        = 3
  allow_user_external_services = true
  intro                        = "hello"
  responses_endpoint           = "http://example.invalid/responses"
  responses_defaults = {
    temperature = 0.5
  }

  mcp = {
    endpoint  = "http://example.invalid/mcp"
    transport = "HTTP"
    mcp_apps = {
      domain_override = "apps.example.invalid"
    }
  }

  routes = {
    static = {
      paths = ["/acc/static"]
      response = {
        status = 200
        body   = "ok"
      }
    }
    proxied = {
      paths   = ["/acc/proxied"]
      methods = ["GET"]
      upstreams = [{
        endpoint = "http://example.invalid/up"
        key      = "upstream-key"
        tier     = 0
        weight   = 1
      }]
    }
  }

  external_services = {
    oauth = {
      display_name = "OAuth service"
      auth_settings = {
        authentication_type    = "OAUTH"
        client_id              = "acc-client"
        client_secret          = %[2]q
        authorization_endpoint = "http://example.invalid/authorize"
        token_endpoint         = "http://example.invalid/token"
        scopes_supported       = ["read"]
      }
    }
  }
}

data "aidial_application" "i" {
  bucket = aidial_application.i.bucket
  path   = aidial_application.i.path
}
`, path, secret)
}

// TestAccApplication_fullFields applies mcp, routes and external_services
// against a real Core, then re-plans to prove Core's defaults cause no churn.
func TestAccApplication_fullFields(t *testing.T) {
	path := accPath(t)
	const res, ds = "aidial_application.i", "data.aidial_application.i"
	config := accApplicationFullConfig(path, "secret-12345")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "input_attachment_types.#", "1"),
					resource.TestCheckResourceAttr(res, "description_keywords.#", "2"),
					resource.TestCheckResourceAttr(res, "dependencies.#", "1"),
					resource.TestCheckResourceAttr(res, "max_input_attachments", "3"),
					resource.TestCheckResourceAttr(res, "allow_user_external_services", "true"),
					resource.TestCheckResourceAttr(res, "mcp.transport", "HTTP"),
					resource.TestCheckResourceAttr(res, "mcp.config_delivery", "META"),
					resource.TestCheckResourceAttr(res, "mcp.forward_per_request_key", "true"),
					resource.TestCheckResourceAttr(res, "mcp.mcp_apps.domain_override", "apps.example.invalid"),
					resource.TestCheckResourceAttr(res, "routes.%", "2"),
					resource.TestCheckResourceAttr(res, "routes.static.response.body", "ok"),
					resource.TestCheckResourceAttr(res, "routes.proxied.upstreams.0.key", "upstream-key"),
					resource.TestCheckResourceAttr(res, "external_services.oauth.auth_settings.client_secret", "secret-12345"),
					resource.TestCheckResourceAttr(ds, "mcp.endpoint", "http://example.invalid/mcp"),
					resource.TestCheckResourceAttr(ds, "routes.%", "2"),
					resource.TestCheckNoResourceAttr(ds, "external_services.oauth.auth_settings.client_secret"),
				),
			},
			{Config: config, PlanOnly: true}, // Core's defaults and the retained secrets cause no diff
			{Config: accApplicationFullConfig(path, "rotated-secret-1"), Check: resource.TestCheckResourceAttr(res, "external_services.oauth.auth_settings.client_secret", "rotated-secret-1")},
		},
	})
}
