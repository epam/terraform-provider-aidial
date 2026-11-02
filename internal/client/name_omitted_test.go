package client

import (
	"encoding/json"
	"testing"
)

// DIAL Core ignores the name in the request body, so an unset name must not be sent.
func TestUnsetNameIsOmittedFromRequestBody(t *testing.T) {
	for name, v := range map[string]any{"model": Model{Type: "chat"}, "role": Role{}, "route": Route{}} {
		body, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(body, &m); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, ok := m["name"]; ok {
			t.Errorf("%s: body %s must not contain name", name, body)
		}
	}
}

func TestConfiguredNameIsSent(t *testing.T) {
	body, _ := json.Marshal(Role{Name: "r"})
	if !jsonHas(body, "name", `"r"`) {
		t.Fatalf("name missing from %s", body)
	}
}

func jsonHas(body []byte, key, want string) bool {
	var m map[string]json.RawMessage
	if json.Unmarshal(body, &m) != nil {
		return false
	}
	return string(m[key]) == want
}
