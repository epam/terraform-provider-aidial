package client

import (
	"encoding/json"
	"testing"
)

func strPtr(s string) *string { return &s }

func TestPricingRateMarshalFlatIsPlainString(t *testing.T) {
	out, err := json.Marshal(ModelPricing{Unit: "token", Prompt: &PricingRate{Rate: strPtr("0.000002")}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), `{"unit":"token","prompt":"0.000002"}`; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestPricingRateConditionalRoundTrip(t *testing.T) {
	in := ModelPricing{Unit: "token", Prompt: &PricingRate{
		Test:    &PricingCondition{Field: "promptTokens", Operator: ">", Value: "100000"},
		IfTrue:  &PricingRate{Rate: strPtr("0.0000005")},
		IfFalse: &PricingRate{Rate: strPtr("0.0000001")},
	}}
	out, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"unit":"token","prompt":{"test":{"field":"promptTokens","operator":"\u003e","value":100000},"ifTrue":"0.0000005","ifFalse":"0.0000001"}}`
	if string(out) != want {
		t.Fatalf("got %s, want %s", out, want)
	}

	var back ModelPricing
	if err := json.Unmarshal(out, &back); err != nil {
		t.Fatal(err)
	}
	p := back.Prompt
	if p == nil || p.Rate != nil || p.Test == nil || p.Test.Value != "100000" || p.Test.Operator != ">" ||
		p.IfTrue == nil || *p.IfTrue.Rate != "0.0000005" || *p.IfFalse.Rate != "0.0000001" {
		t.Fatalf("unexpected round trip: %+v", p)
	}
}

func TestPricingRateUnmarshalForms(t *testing.T) {
	var p ModelPricing
	body := `{"unit":"token","prompt":"1","completion":{"rate":"2"},"cacheRead":3,"cacheWrite":{"test":{"field":"f","operator":"==","value":"x"},"ifTrue":{"rate":"4"},"ifFalse":{"rate":"5"}}}`
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatal(err)
	}
	if *p.Prompt.Rate != "1" || *p.Completion.Rate != "2" || *p.CacheRead.Rate != "3" {
		t.Fatalf("unexpected rates: %+v %+v %+v", p.Prompt, p.Completion, p.CacheRead)
	}
	if p.CacheWrite.Test.Value != "x" || *p.CacheWrite.IfTrue.Rate != "4" || *p.CacheWrite.IfFalse.Rate != "5" {
		t.Fatalf("unexpected conditional: %+v", p.CacheWrite)
	}
}
