package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

// TestToolsetClientSecretSensitive pins the sensitive-attributes spec for
// toolsets: the OAuth client secret is sensitive on the resource, and neither
// toolset data source exposes it.
func TestToolsetClientSecretSensitive(t *testing.T) {
	ctx := context.Background()

	var rs resource.SchemaResponse
	(&toolsetResource{}).Schema(ctx, resource.SchemaRequest{}, &rs)
	auth, ok := rs.Schema.Attributes["auth_settings"].(rschema.SingleNestedAttribute)
	if !ok {
		t.Fatal("auth_settings is not a SingleNestedAttribute")
	}
	secret, ok := auth.Attributes["client_secret"]
	if !ok || !secret.IsSensitive() {
		t.Fatal("auth_settings.client_secret must exist and be sensitive on aidial_toolset")
	}
	for name, a := range auth.Attributes {
		if name != "client_secret" && a.IsSensitive() {
			t.Errorf("auth_settings.%s must not be sensitive", name)
		}
	}

	for name, ds := range map[string]datasource.DataSource{
		"aidial_toolset":      &toolsetDataSource{},
		"aidial_file_toolset": &fileToolsetDataSource{},
	} {
		var resp datasource.SchemaResponse
		ds.Schema(ctx, datasource.SchemaRequest{}, &resp)
		dsAuth, ok := resp.Schema.Attributes["auth_settings"].(dsschema.SingleNestedAttribute)
		if !ok {
			t.Fatalf("%s: auth_settings is not a SingleNestedAttribute", name)
		}
		if _, has := dsAuth.Attributes["client_secret"]; has {
			t.Errorf("%s must not expose auth_settings.client_secret", name)
		}
	}
}
