package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

func TestAuthHeaderNameValidators(t *testing.T) {
	ctx := context.Background()
	sch := rschema.Schema{Attributes: map[string]rschema.Attribute{
		"auth_type":        rschema.StringAttribute{Optional: true},
		"auth_header_name": rschema.StringAttribute{Optional: true},
	}}
	objType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"auth_type": tftypes.String, "auth_header_name": tftypes.String}}
	cfg := func(authType, header *string) tfsdk.Config {
		val := func(s *string) tftypes.Value {
			if s == nil {
				return tftypes.NewValue(tftypes.String, nil)
			}
			return tftypes.NewValue(tftypes.String, *s)
		}
		return tfsdk.Config{Schema: sch, Raw: tftypes.NewValue(objType, map[string]tftypes.Value{"auth_type": val(authType), "auth_header_name": val(header)})}
	}
	str := func(s string) *string { return &s }
	cases := []struct {
		name      string
		authType  *string
		header    *string
		wantErrOn string // "", "auth_type" or "auth_header_name"
	}{
		{"custom header with name", str("CUSTOM_HEADER"), str("X-Key"), ""},
		{"custom header without name", str("CUSTOM_HEADER"), nil, "auth_type"},
		{"bearer with name", str("BEARER"), str("X-Key"), "auth_header_name"},
		{"unset type with name", nil, str("X-Key"), "auth_header_name"},
		{"bearer alone", str("BEARER"), nil, ""},
	}
	for _, c := range cases {
		config := cfg(c.authType, c.header)
		var errs [2]bool
		run := func(i int, v validator.String, attr string, val *string) {
			req := validator.StringRequest{Path: path.Root(attr), Config: config, ConfigValue: types.StringPointerValue(val)}
			resp := &validator.StringResponse{}
			v.ValidateString(ctx, req, resp)
			errs[i] = resp.Diagnostics.HasError()
		}
		run(0, authHeaderNameRequiredForCustomHeader{}, "auth_type", c.authType)
		run(1, authHeaderNameOnlyForCustomHeader{}, "auth_header_name", c.header)
		wantType, wantHeader := c.wantErrOn == "auth_type", c.wantErrOn == "auth_header_name"
		if errs[0] != wantType || errs[1] != wantHeader {
			t.Errorf("%s: errors (type, header) = %v, want (%v, %v)", c.name, errs, wantType, wantHeader)
		}
	}
}
