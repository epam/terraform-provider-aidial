package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
)

// PricingRate is DIAL Core's PricingRate: either a flat Rate, or a conditional
// price (Test, IfTrue, IfFalse). A flat rate is written as a plain JSON string,
// the legacy form Core accepts; Core may return either a string or an object.
type PricingRate struct {
	Rate    *string
	Test    *PricingCondition
	IfTrue  *PricingRate
	IfFalse *PricingRate
}

// PricingCondition is DIAL Core's Condition. Value is kept as text: it is
// written as a JSON number when it is one, otherwise as a JSON string.
type PricingCondition struct {
	Field    string
	Operator string
	Value    string
}

var jsonNumberPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][+-]?[0-9]+)?$`)

type pricingRateWire struct {
	Rate    json.RawMessage   `json:"rate,omitempty"`
	Test    *PricingCondition `json:"test,omitempty"`
	IfTrue  *PricingRate      `json:"ifTrue,omitempty"`
	IfFalse *PricingRate      `json:"ifFalse,omitempty"`
}

func (r PricingRate) MarshalJSON() ([]byte, error) {
	if r.Test == nil && r.IfTrue == nil && r.IfFalse == nil && r.Rate != nil {
		return json.Marshal(*r.Rate)
	}

	wire := pricingRateWire{Test: r.Test, IfTrue: r.IfTrue, IfFalse: r.IfFalse}
	if r.Rate != nil {
		raw, err := json.Marshal(*r.Rate)
		if err != nil {
			return nil, err
		}
		wire.Rate = raw
	}

	return json.Marshal(wire)
}

func (r *PricingRate) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*r = PricingRate{}
		return nil
	}

	if trimmed[0] != '{' {
		text, err := scalarText(trimmed)
		if err != nil {
			return fmt.Errorf("pricing rate: %w", err)
		}
		*r = PricingRate{Rate: &text}
		return nil
	}

	var wire pricingRateWire
	if err := json.Unmarshal(trimmed, &wire); err != nil {
		return err
	}

	result := PricingRate{Test: wire.Test, IfTrue: wire.IfTrue, IfFalse: wire.IfFalse}
	if len(wire.Rate) > 0 && !bytes.Equal(wire.Rate, []byte("null")) {
		text, err := scalarText(wire.Rate)
		if err != nil {
			return fmt.Errorf("pricing rate: %w", err)
		}
		result.Rate = &text
	}
	*r = result

	return nil
}

func (c PricingCondition) MarshalJSON() ([]byte, error) {
	var value any = c.Value
	if jsonNumberPattern.MatchString(c.Value) {
		value = json.Number(c.Value)
	}

	return json.Marshal(struct {
		Field    string `json:"field"`
		Operator string `json:"operator"`
		Value    any    `json:"value"`
	}{c.Field, c.Operator, value})
}

func (c *PricingCondition) UnmarshalJSON(data []byte) error {
	var wire struct {
		Field    string          `json:"field"`
		Operator string          `json:"operator"`
		Value    json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	result := PricingCondition{Field: wire.Field, Operator: wire.Operator}
	if len(wire.Value) > 0 && !bytes.Equal(wire.Value, []byte("null")) {
		text, err := scalarText(wire.Value)
		if err != nil {
			return fmt.Errorf("pricing condition value: %w", err)
		}
		result.Value = text
	}
	*c = result

	return nil
}

// scalarText returns a JSON string's content, or the literal text of a number
// or boolean. Objects and arrays are rejected.
func scalarText(raw []byte) (string, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) > 0 && raw[0] == '"' {
		var text string
		if err := json.Unmarshal(raw, &text); err != nil {
			return "", err
		}
		return text, nil
	}
	if len(raw) > 0 && (raw[0] == '{' || raw[0] == '[') {
		return "", fmt.Errorf("expected a string or number, got %s", raw)
	}

	return string(raw), nil
}
