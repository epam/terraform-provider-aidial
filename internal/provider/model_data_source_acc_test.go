package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccModelDataSource_matchesResource(t *testing.T) {
	path := accPath(t)
	cfg := accProviderConfig() + fmt.Sprintf(`
resource "aidial_model" "t" {
  bucket = "platform"
  path         = %q
  name         = %q
  type         = "CHAT"
  display_name = "acc"
  description  = "acc data source"
  endpoint     = "https://example.invalid/v1/chat/completions"
}

data "aidial_model" "t" {
  name   = aidial_model.t.name
  bucket = aidial_model.t.bucket
  path   = aidial_model.t.path
}

`, path, path)

	const res, ds = "aidial_model.t", "data.aidial_model.t"
	attrs := []string{"id", "bucket", "path", "name", "api_name", "type", "display_name", "description", "endpoint", "status"}
	checks := make([]resource.TestCheckFunc, 0, len(attrs))
	for _, attr := range attrs {
		checks = append(checks, resource.TestCheckResourceAttrPair(ds, attr, res, attr))
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, Check: resource.ComposeAggregateTestCheckFunc(checks...)},
		},
	})
}
