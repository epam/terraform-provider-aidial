package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func accApplicationTypeSchemaConfig(path, description string) string {
	return accProviderConfig() + fmt.Sprintf(`
resource "aidial_application_type_schema" "t" {
  bucket = "platform"
  path   = %[1]q

  schema = jsonencode({
    "$id" = "https://dial.epam.com/application_type_schemas/%[1]s"
    type  = "object"
    properties = {
      prompt = { type = "string", description = %[2]q }
    }
  })
}

data "aidial_application_type_schema" "t" {
  bucket = aidial_application_type_schema.t.bucket
  path   = aidial_application_type_schema.t.path
}
`, path, description)
}

func TestAccApplicationTypeSchema_lifecycle(t *testing.T) {
	path := accPath(t)
	const res, ds = "aidial_application_type_schema.t", "data.aidial_application_type_schema.t"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { accPreCheck(t) },
		ProtoV6ProviderFactories: accProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: accApplicationTypeSchemaConfig(path, "first"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(res, "id", "platform/"+path),
					resource.TestCheckResourceAttr(res, "schema_id", "https://dial.epam.com/application_type_schemas/"+path),
					resource.TestCheckResourceAttrPair(ds, "id", res, "id"),
					resource.TestCheckResourceAttrPair(ds, "schema_id", res, "schema_id"),
				),
			},
			{
				Config: accApplicationTypeSchemaConfig(path, "second"),
				Check:  resource.TestCheckResourceAttr(res, "schema_id", "https://dial.epam.com/application_type_schemas/"+path),
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
