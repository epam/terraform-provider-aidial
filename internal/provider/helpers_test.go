package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestStringPointer(t *testing.T) {
	if got := stringPointer(types.StringNull()); got != nil {
		t.Fatalf("expected nil for null, got %v", *got)
	}
	if got := stringPointer(types.StringUnknown()); got != nil {
		t.Fatalf("expected nil for unknown, got %v", *got)
	}
	got := stringPointer(types.StringValue("hello"))
	if got == nil || *got != "hello" {
		t.Fatalf("expected pointer to %q, got %v", "hello", got)
	}
}

func TestBoolPointer(t *testing.T) {
	if got := boolPointer(types.BoolNull()); got != nil {
		t.Fatalf("expected nil for null, got %v", *got)
	}
	got := boolPointer(types.BoolValue(true))
	if got == nil || *got != true {
		t.Fatalf("expected pointer to true, got %v", got)
	}
}

func TestInt64Pointer(t *testing.T) {
	if got := int64Pointer(types.Int64Null()); got != nil {
		t.Fatalf("expected nil for null, got %v", *got)
	}
	got := int64Pointer(types.Int64Value(42))
	if got == nil || *got != 42 {
		t.Fatalf("expected pointer to 42, got %v", got)
	}
}

func TestStringValue(t *testing.T) {
	if got := stringValue(nil); !got.IsNull() {
		t.Fatalf("expected null for nil pointer, got %v", got)
	}
	v := "hello"
	if got := stringValue(&v); got.ValueString() != "hello" {
		t.Fatalf("expected %q, got %q", "hello", got.ValueString())
	}
}

func TestBoolValue(t *testing.T) {
	if got := boolValue(nil); !got.IsNull() {
		t.Fatalf("expected null for nil pointer, got %v", got)
	}
	v := true
	if got := boolValue(&v); !got.ValueBool() {
		t.Fatal("expected true")
	}
}

func TestInt64Value(t *testing.T) {
	if got := int64Value(nil); !got.IsNull() {
		t.Fatalf("expected null for nil pointer, got %v", got)
	}
	v := int64(42)
	if got := int64Value(&v); got.ValueInt64() != 42 {
		t.Fatalf("expected 42, got %d", got.ValueInt64())
	}
}

func TestLimitsFromStateRoundTrip(t *testing.T) {
	m, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"models/platform/gpt": `{"day": 100, "minute": 1}`,
	})
	if diags.HasError() {
		t.Fatalf("failed to build map: %v", diags)
	}

	limits, diags := limitsFromState(m)
	if diags.HasError() {
		t.Fatalf("limitsFromState: %v", diags)
	}

	var decoded map[string]any
	if err := json.Unmarshal(limits["models/platform/gpt"], &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded["day"] != float64(100) {
		t.Fatalf("expected day=100, got %v", decoded["day"])
	}
}

func TestLimitsFromStateRejectsNonObjectJSON(t *testing.T) {
	m, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"bad": `[1, 2, 3]`,
	})
	if diags.HasError() {
		t.Fatalf("failed to build map: %v", diags)
	}

	_, diags = limitsFromState(m)
	if !diags.HasError() {
		t.Fatal("expected error for non-object JSON value")
	}
}

func TestLimitsFromStateRejectsInvalidJSON(t *testing.T) {
	m, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"bad": `not json`,
	})
	if diags.HasError() {
		t.Fatalf("failed to build map: %v", diags)
	}

	_, diags = limitsFromState(m)
	if !diags.HasError() {
		t.Fatal("expected error for invalid JSON value")
	}
}

func TestLimitsFromStateNullOrUnknownIsEmpty(t *testing.T) {
	limits, diags := limitsFromState(types.MapNull(types.StringType))
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if len(limits) != 0 {
		t.Fatalf("expected empty map, got %v", limits)
	}

	limits, diags = limitsFromState(types.MapUnknown(types.StringType))
	if diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
	if len(limits) != 0 {
		t.Fatalf("expected empty map, got %v", limits)
	}
}

func TestLimitsToStateNilIsNullMap(t *testing.T) {
	got := limitsToState(nil)
	if !got.IsNull() {
		t.Fatalf("expected null map, got %v", got)
	}
}

func TestLimitsToStateCanonicalizesJSON(t *testing.T) {
	got := limitsToState(map[string]json.RawMessage{
		"a": json.RawMessage(`{  "b" : 1 }`),
	})
	elements := got.Elements()
	v, ok := elements["a"].(types.String)
	if !ok {
		t.Fatalf("expected string element, got %T", elements["a"])
	}
	if v.ValueString() != `{"b":1}` {
		t.Fatalf("expected canonical JSON, got %q", v.ValueString())
	}
}

func TestShareFromStateRoundTrip(t *testing.T) {
	m, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"sample": `{"invitation_ttl": 3600, "max_accepted_users": 10}`,
	})
	if diags.HasError() {
		t.Fatalf("failed to build map: %v", diags)
	}

	share, diags := shareFromState(m)
	if diags.HasError() {
		t.Fatalf("shareFromState: %v", diags)
	}

	rule, ok := share["sample"]
	if !ok {
		t.Fatal("expected 'sample' key in result")
	}
	if rule.InvitationTTL == nil || *rule.InvitationTTL != 3600 {
		t.Fatalf("expected invitation_ttl=3600, got %v", rule.InvitationTTL)
	}
	if rule.MaxAcceptedUsers == nil || *rule.MaxAcceptedUsers != 10 {
		t.Fatalf("expected max_accepted_users=10, got %v", rule.MaxAcceptedUsers)
	}
}

func TestShareFromStateRejectsInvalidJSON(t *testing.T) {
	m, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"bad": `not json`,
	})
	if diags.HasError() {
		t.Fatalf("failed to build map: %v", diags)
	}

	_, diags = shareFromState(m)
	if !diags.HasError() {
		t.Fatal("expected error for invalid JSON value")
	}
}

func TestShareToStateNilIsNullMap(t *testing.T) {
	got := shareToState(nil)
	if !got.IsNull() {
		t.Fatalf("expected null map, got %v", got)
	}
}

func TestPathDefaultsToName(t *testing.T) {
	cfgSchema := schema.Schema{Attributes: map[string]schema.Attribute{
		"name": schema.StringAttribute{Optional: true},
		"path": schema.StringAttribute{Optional: true, Computed: true},
	}}
	objType := cfgSchema.Type().TerraformType(context.Background())

	run := func(t *testing.T, name, pathCfg tftypes.Value) types.String {
		t.Helper()
		cfg := tfsdk.Config{
			Schema: cfgSchema,
			Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
				"name": name,
				"path": pathCfg,
			}),
		}
		req := planmodifier.StringRequest{Config: cfg, ConfigValue: types.StringNull(), PlanValue: types.StringUnknown()}
		if !pathCfg.IsNull() {
			req.ConfigValue = types.StringValue("explicit")
			req.PlanValue = types.StringValue("explicit")
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		pathDefaultsToName{}.PlanModifyString(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("diags: %v", resp.Diagnostics)
		}
		return resp.PlanValue
	}

	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
	null := tftypes.NewValue(tftypes.String, nil)
	unknown := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)

	if got := run(t, str("my-name"), null); got.ValueString() != "my-name" {
		t.Errorf("unset path = %v, want my-name", got)
	}
	if got := run(t, str("my-name"), str("explicit")); got.ValueString() != "explicit" {
		t.Errorf("explicit path = %v, want explicit", got)
	}
	if got := run(t, unknown, null); !got.IsUnknown() {
		t.Errorf("unknown name should leave path unknown, got %v", got)
	}
}
