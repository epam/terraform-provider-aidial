package client

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestModelUpstreamExtraDataDecodesObjectOrString(t *testing.T) {
	want := FlexibleStringMap{"region": "us-east-1", "compatible_model_id": "anthropic.claude-sonnet-4-6"}
	for name, payload := range map[string]string{
		"object (Core 0.48.1)":     `{"id":"1","extraData":{"region":"us-east-1","compatible_model_id":"anthropic.claude-sonnet-4-6"}}`,
		"JSON string (newer Core)": `{"id":"1","extraData":"{\"region\":\"us-east-1\",\"compatible_model_id\":\"anthropic.claude-sonnet-4-6\"}"}`,
		"JSON string with spaces":  `{"id":"1","extraData":" { \"compatible_model_id\": \"anthropic.claude-sonnet-4-6\", \"region\": \"us-east-1\" } "}`,
	} {
		var upstream ModelUpstream
		if err := json.Unmarshal([]byte(payload), &upstream); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !reflect.DeepEqual(upstream.ExtraData, want) {
			t.Fatalf("%s: got %v, want %v", name, upstream.ExtraData, want)
		}
	}
}

func TestFlexibleStringMapEmptyAndNonStringValues(t *testing.T) {
	for _, payload := range []string{`null`, `""`, `"  "`} {
		var m FlexibleStringMap
		if err := json.Unmarshal([]byte(payload), &m); err != nil || m != nil {
			t.Fatalf("%s: expected nil map, got %v (%v)", payload, m, err)
		}
	}

	var m FlexibleStringMap
	if err := json.Unmarshal([]byte(`{"retries":3,"enabled":true,"nested":{"a":1}}`), &m); err != nil {
		t.Fatal(err)
	}
	if m["retries"] != "3" || m["enabled"] != "true" || m["nested"] != `{"a":1}` {
		t.Fatalf("non-string values must keep their JSON text, got %v", m)
	}

	if err := json.Unmarshal([]byte(`"region=us-east-1"`), &m); err == nil || !strings.Contains(err.Error(), "region=us-east-1") {
		t.Fatalf("a non-JSON string must fail with the value in the message, got %v", err)
	}
}

func TestModelUpstreamExtraDataIsSentAsObject(t *testing.T) {
	body, err := json.Marshal(ModelUpstream{Endpoint: ptr("e"), ExtraData: FlexibleStringMap{"region": "us-east-1"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"extraData":{"region":"us-east-1"}`) {
		t.Fatalf("extraData must be sent as an object, got %s", body)
	}
	body, _ = json.Marshal(ModelUpstream{Endpoint: ptr("e")})
	if strings.Contains(string(body), "extraData") {
		t.Fatalf("empty extraData must be omitted, got %s", body)
	}
}
