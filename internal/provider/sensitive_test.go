package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

// TestDataSourceUpstreamSecretsSensitive pins the sensitive-attributes spec:
// upstream key and secret_extra_data are redacted in every data source that
// exposes them, and their non-secret siblings are not.
func TestDataSourceUpstreamSecretsSensitive(t *testing.T) {
	ctx := context.Background()
	sources := map[string]datasource.DataSource{
		"aidial_model":      &modelDataSource{},
		"aidial_file_model": &fileModelDataSource{},
	}
	secret := map[string]bool{"key": true, "secret_extra_data": true}
	visible := []string{"endpoint", "responses_endpoint", "id", "extra_data", "tier", "weight"}

	for name, ds := range sources {
		t.Run(name, func(t *testing.T) {
			resp := &datasource.SchemaResponse{}
			ds.Schema(ctx, datasource.SchemaRequest{}, resp)

			upstreams, ok := resp.Schema.Attributes["upstreams"].(dsschema.ListNestedAttribute)
			if !ok {
				t.Fatalf("upstreams is not a ListNestedAttribute")
			}
			attrs := upstreams.NestedObject.Attributes

			for attr := range secret {
				a, ok := attrs[attr]
				if !ok {
					t.Fatalf("upstreams.%s missing", attr)
				}
				if !a.IsSensitive() {
					t.Errorf("upstreams.%s must be sensitive", attr)
				}
			}
			for _, attr := range visible {
				a, ok := attrs[attr]
				if !ok {
					t.Fatalf("upstreams.%s missing", attr)
				}
				if a.IsSensitive() {
					t.Errorf("upstreams.%s must not be sensitive", attr)
				}
			}
		})
	}
}
