package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func nullPricingRate() types.Object { return types.ObjectNull(pricingRateAttrTypes) }

func cond(op, value string) *client.PricingCondition {
	return &client.PricingCondition{Field: "promptTokens", Operator: op, Value: value}
}

func TestPricingRateStateRoundTrip(t *testing.T) {
	rate := &client.PricingRate{
		Test:    cond(">", "100000"),
		IfTrue:  &client.PricingRate{Rate: ptr("0.0000005")},
		IfFalse: &client.PricingRate{Rate: ptr("0.0000001")},
	}

	obj, err := pricingRateToState("prompt", rate, nullPricingRate())
	if err != nil {
		t.Fatal(err)
	}
	back, err := pricingRateToClient(context.Background(), obj)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, rate) {
		t.Fatalf("unexpected round trip: %+v", back)
	}

	flat, err := pricingRateToState("prompt", &client.PricingRate{Rate: ptr("1")}, nullPricingRate())
	if err != nil {
		t.Fatal(err)
	}
	if !flat.Attributes()["test"].IsNull() || !flat.Attributes()["if_true"].IsNull() {
		t.Fatalf("flat rate must leave the condition attributes null: %v", flat)
	}

	null, err := pricingRateToState("prompt", nil, nullPricingRate())
	if err != nil || !null.IsNull() {
		t.Fatalf("nil rate must be a null object, got %v, %v", null, err)
	}
}

// Conditions nest to any depth, as in DIAL Core's PricingRate.
func TestPricingRateNestedConditionRoundTrip(t *testing.T) {
	rate := &client.PricingRate{
		Test: cond(">", "100000"),
		IfTrue: &client.PricingRate{
			Test:    cond("<=", "500000"),
			IfTrue:  &client.PricingRate{Rate: ptr("3")},
			IfFalse: &client.PricingRate{Rate: ptr("2")},
		},
		IfFalse: &client.PricingRate{Rate: ptr("1")},
	}

	obj, err := pricingRateToState("prompt", rate, nullPricingRate())
	if err != nil {
		t.Fatal(err)
	}
	back, err := pricingRateToClient(context.Background(), obj)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, rate) {
		t.Fatalf("unexpected round trip: %+v", back)
	}
}

// A branch written as { rate = "1" } is kept as written, not rewritten to "1".
func TestPricingRateStateKeepsPriorForm(t *testing.T) {
	branch := types.ObjectValueMust(map[string]attr.Type{"rate": types.StringType}, map[string]attr.Value{"rate": types.StringValue("1")})
	prior := types.ObjectValueMust(pricingRateAttrTypes, map[string]attr.Value{
		"rate": types.StringNull(),
		"test": types.ObjectValueMust(pricingConditionAttrTypes, map[string]attr.Value{
			"field": types.StringValue("promptTokens"), "operator": types.StringValue(">"), "value": types.StringValue("100000"),
		}),
		"if_true":  types.DynamicValue(branch),
		"if_false": types.DynamicValue(types.StringValue("2")),
	})
	rate := &client.PricingRate{
		Test:    cond(">", "100000"),
		IfTrue:  &client.PricingRate{Rate: ptr("1")},
		IfFalse: &client.PricingRate{Rate: ptr("2")},
	}

	got, err := pricingRateToState("prompt", rate, prior)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(prior) {
		t.Fatalf("prior form must be kept\n got: %v\nwant: %v", got, prior)
	}

	changed := &client.PricingRate{Test: cond(">", "100000"), IfTrue: &client.PricingRate{Rate: ptr("9")}, IfFalse: &client.PricingRate{Rate: ptr("2")}}
	got, err = pricingRateToState("prompt", changed, prior)
	if err != nil {
		t.Fatal(err)
	}
	if got.Equal(prior) {
		t.Fatal("a different price from Core must replace the prior value")
	}
}

func TestPricingBranchToClientValidation(t *testing.T) {
	obj := func(attrs map[string]attr.Value) attr.Value {
		types_ := map[string]attr.Type{}
		for k, v := range attrs {
			types_[k] = v.Type(context.Background())
		}
		return types.ObjectValueMust(types_, attrs)
	}
	s := func(v string) attr.Value { return types.StringValue(v) }

	bad := map[string]attr.Value{
		"unknown attribute": obj(map[string]attr.Value{"bogus": s("1")}),
		"rate and test":     obj(map[string]attr.Value{"rate": s("1"), "if_true": s("1")}),
		"test without branches": obj(map[string]attr.Value{"test": obj(map[string]attr.Value{
			"field": s("f"), "operator": s("=="), "value": s("1")})}),
		"bad operator": obj(map[string]attr.Value{
			"test":    obj(map[string]attr.Value{"field": s("f"), "operator": s("=>"), "value": s("1")}),
			"if_true": s("1"), "if_false": s("2")}),
		"empty object": obj(map[string]attr.Value{}),
		"bool":         types.BoolValue(true),
	}
	for name, v := range bad {
		if _, err := pricingBranchToClient(v); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
