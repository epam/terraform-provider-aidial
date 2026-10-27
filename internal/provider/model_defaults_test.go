package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const sonnetDefaultsJSON = `{"max_tokens":64000,"custom_fields":{"configuration":{"thinking":{"type":"disabled"}}}}`

func dynamicFromJSON(t *testing.T, raw string) types.Dynamic {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	return types.DynamicValue(jsonToAttrValue(value))
}

func assertJSONEqual(t *testing.T, got []byte, want string) {
	t.Helper()
	var a, b any
	if err := json.Unmarshal(got, &a); err != nil {
		t.Fatalf("invalid JSON %s: %v", got, err)
	}
	if err := json.Unmarshal([]byte(want), &b); err != nil {
		t.Fatal(err)
	}
	if !jsonEqual(a, b) {
		t.Fatalf("JSON mismatch:\n got: %s\nwant: %s", got, want)
	}
}

func TestModelFromPlanSendsFreeFormDefaults(t *testing.T) {
	plan := newModelPlan(t)
	plan.Defaults = dynamicFromJSON(t, sonnetDefaultsJSON)

	model, err := modelFromPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}
	assertJSONEqual(t, model.Defaults, sonnetDefaultsJSON)

	body, _ := json.Marshal(model)
	if !bytes.Contains(body, []byte(`"defaults":{`)) {
		t.Fatalf("defaults missing from request body: %s", body)
	}
}

func TestModelFromPlanRejectsNonObjectDefaults(t *testing.T) {
	plan := newModelPlan(t)
	plan.Defaults = types.DynamicValue(types.StringValue("nope"))
	if _, err := modelFromPlan(context.Background(), plan); err == nil {
		t.Fatal("expected an error for a non-object defaults value")
	}
}

func TestModelFromPlanOmitsUnsetDefaults(t *testing.T) {
	plan := newModelPlan(t)
	plan.Defaults = types.DynamicNull()
	model, err := modelFromPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}
	body, _ := json.Marshal(model)
	if bytes.Contains(body, []byte(`"defaults"`)) {
		t.Fatalf("unset defaults must not be sent: %s", body)
	}
}

func TestSetModelStateKeepsConfiguredDefaults(t *testing.T) {
	// The API returns the same JSON with different key order and spacing.
	prior := dynamicFromJSON(t, sonnetDefaultsJSON)
	state := modelResourceModel{Name: types.StringValue("m"), Defaults: prior}
	api := json.RawMessage(`{"custom_fields": {"configuration": {"thinking": {"type": "disabled"}}}, "max_tokens": 64000}`)

	if err := setModelState(context.Background(), &state, &client.Model{Name: "m", Type: "CHAT", Defaults: api}); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if !state.Defaults.Equal(prior) {
		t.Fatalf("configured defaults must be kept, got %v", state.Defaults)
	}
}

func TestSetModelStateTakesChangedDefaultsFromAPI(t *testing.T) {
	state := modelResourceModel{Name: types.StringValue("m"), Defaults: dynamicFromJSON(t, `{"max_tokens":1000}`)}
	api := json.RawMessage(sonnetDefaultsJSON)

	if err := setModelState(context.Background(), &state, &client.Model{Name: "m", Type: "CHAT", Defaults: api}); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	got, _ := defaultsFromPlan(state.Defaults)
	assertJSONEqual(t, got, sonnetDefaultsJSON)
}

func TestSetModelStateEmptyDefaults(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		prior    types.Dynamic
		api      json.RawMessage
		wantNull bool
	}{
		"unset, API omits":     {types.DynamicNull(), nil, true},
		"unset, API {}":        {types.DynamicNull(), json.RawMessage(`{}`), true},
		"configured {}, omits": {dynamicFromJSON(t, `{}`), nil, false},
		"configured {}, {}":    {dynamicFromJSON(t, `{}`), json.RawMessage(`{}`), false},
	} {
		state := modelResourceModel{Name: types.StringValue("m"), Defaults: tc.prior}
		if err := setModelState(ctx, &state, &client.Model{Name: "m", Type: "CHAT", Defaults: tc.api}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if state.Defaults.IsNull() != tc.wantNull {
			t.Fatalf("%s: null=%v, want %v", name, state.Defaults.IsNull(), tc.wantNull)
		}
	}
}

func TestModelUpgradeStateV0Defaults(t *testing.T) {
	ctx := context.Background()
	r := &modelResource{}
	upgrader := r.UpgradeState(ctx)[0]
	priorType := upgrader.PriorSchema.Type().TerraformType(ctx).(tftypes.Object)

	var current resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &current)
	currentType := current.Schema.Type().TerraformType(ctx)

	toolType := priorType.AttributeTypes["defaults"].(tftypes.Object).AttributeTypes["tools"].(tftypes.List).ElementType.(tftypes.Object)
	staticType := toolType.AttributeTypes["static_function"]
	tools := tftypes.NewValue(tftypes.List{ElementType: toolType}, []tftypes.Value{
		tftypes.NewValue(toolType, map[string]tftypes.Value{
			"type":            tftypes.NewValue(tftypes.String, "static_function"),
			"static_function": tftypes.NewValue(staticType, map[string]tftypes.Value{"name": tftypes.NewValue(tftypes.String, "web_search")}),
		}),
	})

	run := func(defaults tftypes.Value) types.Dynamic {
		t.Helper()
		attributes := map[string]tftypes.Value{}
		for name, attributeType := range priorType.AttributeTypes {
			attributes[name] = tftypes.NewValue(attributeType, nil)
		}
		attributes["name"] = tftypes.NewValue(tftypes.String, "m")
		attributes["defaults"] = defaults
		req := resource.UpgradeStateRequest{State: &tfsdk.State{Schema: *upgrader.PriorSchema, Raw: tftypes.NewValue(priorType, attributes)}}
		resp := &resource.UpgradeStateResponse{State: tfsdk.State{Schema: current.Schema, Raw: tftypes.NewValue(currentType, nil)}}
		upgrader.StateUpgrader(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("upgrade: %v", resp.Diagnostics)
		}
		var upgraded modelResourceModel
		if diags := resp.State.Get(ctx, &upgraded); diags.HasError() {
			t.Fatalf("read upgraded state: %v", diags)
		}
		if upgraded.Name.ValueString() != "m" {
			t.Fatalf("other attributes must survive the upgrade, name=%v", upgraded.Name)
		}
		return upgraded.Defaults
	}

	defaultsType := priorType.AttributeTypes["defaults"]
	withTools := run(tftypes.NewValue(defaultsType, map[string]tftypes.Value{"tools": tools}))
	got, err := defaultsFromPlan(withTools)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, got, `{"tools":[{"type":"static_function","static_function":{"name":"web_search"}}]}`)

	if got := run(tftypes.NewValue(defaultsType, nil)); !got.IsNull() {
		t.Fatalf("null defaults must stay null, got %v", got)
	}
	if got := run(tftypes.NewValue(defaultsType, map[string]tftypes.Value{"tools": tftypes.NewValue(tftypes.List{ElementType: toolType}, nil)})); !got.IsNull() {
		t.Fatalf("defaults without tools must become null, got %v", got)
	}
}
