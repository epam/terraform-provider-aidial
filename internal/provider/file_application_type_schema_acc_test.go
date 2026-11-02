package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The ephemeral Core has no static application type schemas, so this checks
// that the data source reaches the file-config endpoint and reports a missing
// schema as an error.
func TestAccFileApplicationTypeSchemaDataSource_notFound(t *testing.T) {
	cfg := accProviderConfig() + `
data "aidial_file_application_type_schema" "t" {
  name = "tf-acc-no-such-schema"
}
`
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{Config: cfg, ExpectError: regexp.MustCompile(`(?i)not found`)},
		},
	})
}
