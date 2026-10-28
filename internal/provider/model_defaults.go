package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"reflect"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// defaultsFromPlan converts the free-form `defaults` value into the JSON object
// sent to DIAL Core (Model.defaults is MapStringObject in the API).
func defaultsFromPlan(value types.Dynamic) (json.RawMessage, error) {
	return freeFormObjectFromPlan("defaults", value)
}

// freeFormObjectFromPlan converts a free-form object attribute (a MapStringObject
// in the API) into the JSON sent to DIAL Core; name is used in diagnostics.
func freeFormObjectFromPlan(name string, value types.Dynamic) (json.RawMessage, error) {
	if value.IsNull() || value.IsUnknown() || value.IsUnderlyingValueNull() || value.IsUnderlyingValueUnknown() {
		return nil, nil
	}
	decoded, err := attrValueToJSON(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s: %w", name, err)
	}
	if _, ok := decoded.(map[string]any); !ok {
		return nil, fmt.Errorf("%s must be an object", name)
	}
	return json.Marshal(decoded)
}

// defaultsToState keeps the prior (planned or stored) value when the API
// returned the same JSON, so Terraform's types and formatting from the config
// survive the read-back. Otherwise the API value becomes the new state.
func defaultsToState(prior types.Dynamic, api json.RawMessage) (types.Dynamic, error) {
	return freeFormObjectToState(prior, api)
}

// freeFormObjectToState is defaultsToState for any free-form object attribute.
func freeFormObjectToState(prior types.Dynamic, api json.RawMessage) (types.Dynamic, error) {
	var apiValue any
	if len(bytes.TrimSpace(api)) > 0 {
		decoder := json.NewDecoder(bytes.NewReader(api))
		decoder.UseNumber()
		if err := decoder.Decode(&apiValue); err != nil {
			return types.DynamicNull(), err
		}
	}

	if !prior.IsNull() && !prior.IsUnknown() && !prior.IsUnderlyingValueNull() && !prior.IsUnderlyingValueUnknown() {
		if priorValue, err := attrValueToJSON(prior); err == nil {
			compareTo := apiValue
			if compareTo == nil {
				compareTo = map[string]any{}
			}
			if jsonEqual(priorValue, compareTo) {
				return prior, nil
			}
		}
	}

	if object, ok := apiValue.(map[string]any); apiValue == nil || (ok && len(object) == 0) {
		return types.DynamicNull(), nil
	}
	return types.DynamicValue(jsonToAttrValue(apiValue)), nil
}

func jsonEqual(a, b any) bool {
	encodedA, errA := json.Marshal(a)
	encodedB, errB := json.Marshal(b)
	if errA != nil || errB != nil {
		return false
	}
	var decodedA, decodedB any
	if json.Unmarshal(encodedA, &decodedA) != nil || json.Unmarshal(encodedB, &decodedB) != nil {
		return false
	}
	return reflect.DeepEqual(decodedA, decodedB)
}

// attrValueToJSON converts a Terraform value into plain Go values that
// encoding/json marshals as the equivalent JSON.
func attrValueToJSON(value attr.Value) (any, error) {
	if value.IsUnknown() {
		return nil, fmt.Errorf("value is not known yet")
	}
	if value.IsNull() {
		return nil, nil
	}
	switch v := value.(type) {
	case basetypes.DynamicValue:
		if v.IsUnderlyingValueNull() {
			return nil, nil
		}
		return attrValueToJSON(v.UnderlyingValue())
	case basetypes.ObjectValue:
		return attrMapToJSON(v.Attributes())
	case basetypes.MapValue:
		return attrMapToJSON(v.Elements())
	case basetypes.TupleValue:
		return attrListToJSON(v.Elements())
	case basetypes.ListValue:
		return attrListToJSON(v.Elements())
	case basetypes.SetValue:
		return attrListToJSON(v.Elements())
	case basetypes.StringValue:
		return v.ValueString(), nil
	case basetypes.BoolValue:
		return v.ValueBool(), nil
	case basetypes.NumberValue:
		return json.Number(v.ValueBigFloat().Text('g', -1)), nil
	case basetypes.Int64Value:
		return v.ValueInt64(), nil
	case basetypes.Float64Value:
		return v.ValueFloat64(), nil
	}
	return nil, fmt.Errorf("unsupported value type %T", value)
}

func attrMapToJSON(elements map[string]attr.Value) (map[string]any, error) {
	result := make(map[string]any, len(elements))
	for key, element := range elements {
		converted, err := attrValueToJSON(element)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		result[key] = converted
	}
	return result, nil
}

func attrListToJSON(elements []attr.Value) ([]any, error) {
	result := make([]any, 0, len(elements))
	for i, element := range elements {
		converted, err := attrValueToJSON(element)
		if err != nil {
			return nil, fmt.Errorf("[%d]: %w", i, err)
		}
		result = append(result, converted)
	}
	return result, nil
}

// jsonToAttrValue builds a Terraform value from decoded JSON (decoded with
// UseNumber): objects become objects, arrays become tuples.
func jsonToAttrValue(value any) attr.Value {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		attrTypes := make(map[string]attr.Type, len(v))
		attrValues := make(map[string]attr.Value, len(v))
		for _, key := range keys {
			converted := jsonToAttrValue(v[key])
			attrTypes[key] = converted.Type(context.Background())
			attrValues[key] = converted
		}
		return types.ObjectValueMust(attrTypes, attrValues)
	case []any:
		elementTypes := make([]attr.Type, 0, len(v))
		elements := make([]attr.Value, 0, len(v))
		for _, item := range v {
			converted := jsonToAttrValue(item)
			elementTypes = append(elementTypes, converted.Type(context.Background()))
			elements = append(elements, converted)
		}
		return types.TupleValueMust(elementTypes, elements)
	case string:
		return types.StringValue(v)
	case bool:
		return types.BoolValue(v)
	case json.Number:
		if number, ok := new(big.Float).SetString(string(v)); ok {
			return types.NumberValue(number)
		}
		return types.StringValue(string(v))
	case float64:
		return types.NumberValue(big.NewFloat(v))
	}
	return types.StringNull()
}

// modelDefaultsSchemaV0 is the `defaults` attribute before schema version 1:
// a fixed object that only modeled `tools`.
func modelDefaultsSchemaV0() schema.Attribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"tools": schema.ListNestedAttribute{
				Optional: true,
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{Required: true},
						"static_function": schema.SingleNestedAttribute{
							Optional: true,
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{Required: true},
							},
						},
					},
				},
			},
		},
	}
}

var _ resource.ResourceWithUpgradeState = &modelResource{}

func (r *modelResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	var current resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &current)

	prior := current.Schema
	prior.Version = 0
	prior.Attributes = make(map[string]schema.Attribute, len(current.Schema.Attributes))
	for name, attribute := range current.Schema.Attributes {
		prior.Attributes[name] = attribute
	}
	prior.Attributes["defaults"] = modelDefaultsSchemaV0()

	currentType := current.Schema.Type().TerraformType(ctx)

	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema: &prior,
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var attributes map[string]tftypes.Value
				if err := req.State.Raw.As(&attributes); err != nil {
					resp.Diagnostics.AddError("Unable to upgrade aidial_model state", err.Error())
					return
				}
				defaults, err := defaultsFromV0(ctx, attributes["defaults"])
				if err != nil {
					resp.Diagnostics.AddError("Unable to upgrade aidial_model defaults", err.Error())
					return
				}
				attributes["defaults"] = defaults
				if err := tftypes.ValidateValue(currentType, attributes); err != nil {
					resp.Diagnostics.AddError("Unable to upgrade aidial_model state", err.Error())
					return
				}
				resp.State.Raw = tftypes.NewValue(currentType, attributes)
			},
		},
	}
}

// defaultsFromV0 converts the old `defaults = { tools = [...] }` state value
// into the free-form value, keeping the same JSON that was sent to DIAL Core.
func defaultsFromV0(ctx context.Context, value tftypes.Value) (tftypes.Value, error) {
	null := tftypes.NewValue(tftypes.DynamicPseudoType, nil)
	if value.IsNull() || !value.IsKnown() {
		return null, nil
	}
	var defaults map[string]tftypes.Value
	if err := value.As(&defaults); err != nil {
		return null, err
	}
	toolsValue, ok := defaults["tools"]
	if !ok || toolsValue.IsNull() || !toolsValue.IsKnown() {
		return null, nil
	}
	var toolValues []tftypes.Value
	if err := toolsValue.As(&toolValues); err != nil {
		return null, err
	}

	tools := make([]any, 0, len(toolValues))
	for _, toolValue := range toolValues {
		var fields map[string]tftypes.Value
		if err := toolValue.As(&fields); err != nil {
			return null, err
		}
		tool := map[string]any{}
		var toolType string
		if err := fields["type"].As(&toolType); err == nil && !fields["type"].IsNull() {
			tool["type"] = toolType
		}
		if staticFunction := fields["static_function"]; !staticFunction.IsNull() && staticFunction.IsKnown() {
			var staticFields map[string]tftypes.Value
			if err := staticFunction.As(&staticFields); err != nil {
				return null, err
			}
			var name string
			if err := staticFields["name"].As(&name); err != nil {
				return null, err
			}
			tool["static_function"] = map[string]any{"name": name}
		}
		tools = append(tools, tool)
	}

	return jsonToAttrValue(map[string]any{"tools": tools}).ToTerraformValue(ctx)
}
