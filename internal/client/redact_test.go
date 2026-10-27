package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRedactSensitiveJSON(t *testing.T) {
	key := &Key{
		Name:    "my-key",
		Key:     "super-secret-value",
		Project: "demo",
	}
	payload, err := json.Marshal(key)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	redacted := string(redactSensitiveJSON(payload))
	if strings.Contains(redacted, "super-secret-value") {
		t.Fatalf("expected key value to be redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, `"key":"***REDACTED***"`) {
		t.Fatalf("expected redaction placeholder for key, got: %s", redacted)
	}
	if !strings.Contains(redacted, `"name":"my-key"`) {
		t.Fatalf("expected unrelated fields to survive redaction, got: %s", redacted)
	}
}

func TestRedactSensitiveJSONNestedUpstream(t *testing.T) {
	secretKey := "upstream-secret-key"
	model := &Model{
		Name: "my-model",
		Type: "CHAT",
		Upstreams: []ModelUpstream{
			{
				Endpoint:        ptr("https://example.invalid"),
				Key:             &secretKey,
				SecretExtraData: map[string]string{"token": "another-secret"},
			},
		},
	}
	payload, err := json.Marshal(model)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	redacted := string(redactSensitiveJSON(payload))
	if strings.Contains(redacted, "upstream-secret-key") || strings.Contains(redacted, "another-secret") {
		t.Fatalf("expected nested upstream secrets to be redacted, got: %s", redacted)
	}
	if !strings.Contains(redacted, `"endpoint":"https://example.invalid"`) {
		t.Fatalf("expected unrelated nested fields to survive redaction, got: %s", redacted)
	}
}

func TestRedactSensitiveJSONNonJSONPassesThrough(t *testing.T) {
	raw := []byte("not json")
	if got := string(redactSensitiveJSON(raw)); got != "not json" {
		t.Fatalf("expected non-JSON payload to pass through unchanged, got: %s", got)
	}
}
