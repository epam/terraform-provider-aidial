package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModelUpstreamAuthTypeFlexible(t *testing.T) {
	cases := map[string]*string{
		`{"authType":"BEARER"}`:           ptr("BEARER"),
		`{"authType":{}}`:                 nil,
		`{"authType":null}`:               nil,
		`{}`:                              nil,
		`{"authType":{"name":"API_KEY"}}`: ptr("API_KEY"),
	}
	for in, want := range cases {
		var u ModelUpstream
		if err := json.Unmarshal([]byte(in), &u); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if (u.AuthType == nil) != (want == nil) || (want != nil && *u.AuthType != *want) {
			t.Errorf("%s: got %v want %v", in, u.AuthType, want)
		}
	}
	var u ModelUpstream
	if err := json.Unmarshal([]byte(`{"extraData":{"a":"b"},"tier":2,"authType":{}}`), &u); err != nil || u.ExtraData["a"] != "b" || *u.Tier != 2 {
		t.Fatalf("other fields lost: %+v %v", u, err)
	}
}

// Regression: DIAL Core can return upstreams[].authType as an object ({}),
// which used to fail GetModel with "cannot unmarshal object into Go struct
// field ... authType of type string" ("Failed to read created model").
func TestGetModelObjectShapedUpstreamAuthType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"m","type":"CHAT","upstreams":[` +
			`{"extraData":"{\"compatible_model_id\":\"a\"}","weight":1,"tier":0,"authType":{}},` +
			`{"endpoint":"http://u","authType":"BEARER"}]}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetModel(context.Background(), "platform", "m")
	if err != nil {
		t.Fatalf("GetModel: %v", err)
	}
	if got == nil || len(got.Upstreams) != 2 {
		t.Fatalf("unexpected model: %+v", got)
	}
	first, second := got.Upstreams[0], got.Upstreams[1]
	if first.AuthType != nil || first.ExtraData["compatible_model_id"] != "a" || *first.Weight != 1 {
		t.Errorf("first upstream decoded wrongly: %+v", first)
	}
	if second.AuthType == nil || *second.AuthType != "BEARER" {
		t.Errorf("second upstream authType = %v, want BEARER", second.AuthType)
	}
}
