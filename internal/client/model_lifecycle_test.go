package client

import (
	"encoding/json"
	"testing"
)

// DIAL Core 0.50.2 adds deprecationDate, endOfService and provider to Model.
func TestModelLifecycleFieldsJSON(t *testing.T) {
	var got Model
	in := `{"name":"m","type":"CHAT","deprecationDate":"2026-01-01","endOfService":"2026-06-01","provider":"Acme"}`
	if err := json.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if got.DeprecationDate == nil || *got.DeprecationDate != "2026-01-01" ||
		got.EndOfService == nil || *got.EndOfService != "2026-06-01" ||
		got.Provider == nil || *got.Provider != "Acme" {
		t.Fatalf("fields not decoded: %+v", got)
	}
	out, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	for k, v := range map[string]string{"deprecationDate": "2026-01-01", "endOfService": "2026-06-01", "provider": "Acme"} {
		if m[k] != v {
			t.Errorf("%s = %v, want %s", k, m[k], v)
		}
	}
	out, _ = json.Marshal(Model{Name: "m", Type: "CHAT"})
	m = map[string]any{}
	_ = json.Unmarshal(out, &m)
	for _, k := range []string{"deprecationDate", "endOfService", "provider"} {
		if _, ok := m[k]; ok {
			t.Errorf("%s must be omitted when unset", k)
		}
	}
}
