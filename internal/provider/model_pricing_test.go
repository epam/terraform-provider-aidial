package provider

import (
	"context"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestPricingRateStateRoundTrip(t *testing.T) {
	cond := &client.PricingRate{
		Test:    &client.PricingCondition{Field: "promptTokens", Operator: ">", Value: "100000"},
		IfTrue:  &client.PricingRate{Rate: ptr("0.0000005")},
		IfFalse: &client.PricingRate{Rate: ptr("0.0000001")},
	}

	obj, err := pricingRateToState("prompt", cond)
	if err != nil {
		t.Fatal(err)
	}
	back, err := pricingRateToClient(context.Background(), obj)
	if err != nil {
		t.Fatal(err)
	}
	if back.Rate != nil || back.Test == nil || back.Test.Value != "100000" || *back.IfTrue.Rate != "0.0000005" || *back.IfFalse.Rate != "0.0000001" {
		t.Fatalf("unexpected round trip: %+v", back)
	}

	flat, err := pricingRateToState("prompt", &client.PricingRate{Rate: ptr("1")})
	if err != nil {
		t.Fatal(err)
	}
	if !flat.Attributes()["test"].IsNull() || !flat.Attributes()["if_true"].Equal(types.StringNull()) {
		t.Fatalf("flat rate must leave the condition attributes null: %v", flat)
	}

	null, err := pricingRateToState("prompt", nil)
	if err != nil || !null.IsNull() {
		t.Fatalf("nil rate must be a null object, got %v, %v", null, err)
	}
}

func TestPricingRateStateRejectsNestedCondition(t *testing.T) {
	nested := &client.PricingRate{
		Test:   &client.PricingCondition{Field: "f", Operator: "==", Value: "1"},
		IfTrue: &client.PricingRate{Test: &client.PricingCondition{Field: "g", Operator: "==", Value: "2"}},
	}
	if _, err := pricingRateToState("prompt", nested); err == nil {
		t.Fatal("expected an error for a nested condition")
	}
}
