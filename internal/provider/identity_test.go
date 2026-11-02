package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// identityStringAttrs returns the bucket and path attributes of every resource and
// data source that is addressed by <bucket>/<path>, as validator-carrying views.
type identityAttr struct {
	required, optional, computed bool
	validators                   []validator.String
}

func resourceIdentity(t *testing.T, r resource.Resource) (bucket, path identityAttr, replaces bool) {
	t.Helper()
	var resp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
	conv := func(name string) identityAttr {
		a, ok := resp.Schema.Attributes[name].(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s is not a StringAttribute", name)
		}
		return identityAttr{a.Required, a.Optional, a.Computed, a.Validators}
	}
	b := resp.Schema.Attributes["bucket"].(schema.StringAttribute)
	p := resp.Schema.Attributes["path"].(schema.StringAttribute)
	return conv("bucket"), conv("path"), len(b.PlanModifiers) > 0 && len(p.PlanModifiers) > 0
}

func dataSourceIdentity(t *testing.T, d datasource.DataSource) (bucket, path identityAttr) {
	t.Helper()
	var resp datasource.SchemaResponse
	d.Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	conv := func(name string) identityAttr {
		a, ok := resp.Schema.Attributes[name].(dsschema.StringAttribute)
		if !ok {
			t.Fatalf("%s is not a StringAttribute", name)
		}
		return identityAttr{a.Required, a.Optional, a.Computed, a.Validators}
	}
	return conv("bucket"), conv("path")
}

func validate(t *testing.T, vs []validator.String, v string) bool {
	t.Helper()
	for _, val := range vs {
		resp := &validator.StringResponse{}
		val.ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(v)}, resp)
		if resp.Diagnostics.HasError() {
			return false
		}
	}
	return true
}

func checkIdentity(t *testing.T, bucket, path identityAttr) {
	t.Helper()
	for name, a := range map[string]identityAttr{"bucket": bucket, "path": path} {
		if !a.required || a.optional || a.computed {
			t.Errorf("%s must be required only, got required=%v optional=%v computed=%v", name, a.required, a.optional, a.computed)
		}
	}
	if !validate(t, bucket.validators, "platform") {
		t.Error(`bucket "platform" must be accepted`)
	}
	if validate(t, bucket.validators, "other") {
		t.Error(`bucket "other" must be rejected`)
	}
	if validate(t, bucket.validators, "") {
		t.Error("empty bucket must be rejected")
	}
	if validate(t, path.validators, "") {
		t.Error("empty path must be rejected")
	}
	if !validate(t, path.validators, "team/model-1") {
		t.Error("nested path must be accepted")
	}
}

func TestResourceIdentityRequired(t *testing.T) {
	cases := map[string]resource.Resource{
		"aidial_model":          &modelResource{},
		"aidial_role":           &roleResource{},
		"aidial_route":          &routeResource{},
		"aidial_key":            &keyResource{},
		"aidial_catalog_schema": &catalogSchemaResource{},
	}
	for name, r := range cases {
		t.Run(name, func(t *testing.T) {
			bucket, path, replaces := resourceIdentity(t, r)
			checkIdentity(t, bucket, path)
			if !replaces {
				t.Error("bucket and path must force replacement")
			}
		})
	}
}

func TestDataSourceIdentityRequired(t *testing.T) {
	cases := map[string]datasource.DataSource{
		"aidial_model":          &modelDataSource{},
		"aidial_role":           &roleDataSource{},
		"aidial_catalog_schema": &catalogSchemaDataSource{},
	}
	for name, d := range cases {
		t.Run(name, func(t *testing.T) {
			bucket, path := dataSourceIdentity(t, d)
			checkIdentity(t, bucket, path)
		})
	}
}
