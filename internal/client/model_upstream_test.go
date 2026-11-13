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

func TestModelUpstreamAuthHeaderNameDecode(t *testing.T) {
	var u ModelUpstream
	in := `{"authType":{"type":"CUSTOM_HEADER","headerName":"X-Key"}}`
	if err := json.Unmarshal([]byte(in), &u); err != nil {
		t.Fatal(err)
	}
	if u.AuthType == nil || *u.AuthType != "CUSTOM_HEADER" || u.AuthHeaderName == nil || *u.AuthHeaderName != "X-Key" {
		t.Fatalf("got %v %v", u.AuthType, u.AuthHeaderName)
	}
	// Core 0.50 returns "headerName": null for the other types.
	if err := json.Unmarshal([]byte(`{"authType":{"type":"BEARER","headerName":null}}`), &u); err != nil {
		t.Fatal(err)
	}
	if *u.AuthType != "BEARER" || u.AuthHeaderName != nil {
		t.Fatalf("stale header name kept: %v %v", u.AuthType, u.AuthHeaderName)
	}
}

func TestModelUpstreamAuthTypeEncode(t *testing.T) {
	cases := []struct {
		u    ModelUpstream
		want string
	}{
		{ModelUpstream{}, `{}`},
		{ModelUpstream{AuthType: ptr("BEARER")}, `{"authType":"BEARER"}`},
		{ModelUpstream{AuthType: ptr("CUSTOM_HEADER"), AuthHeaderName: ptr("X-Key")}, `{"authType":{"headerName":"X-Key","type":"CUSTOM_HEADER"}}`},
		{ModelUpstream{AuthType: ptr("CUSTOM_HEADER")}, `{"authType":{"type":"CUSTOM_HEADER"}}`},
		{ModelUpstream{Endpoint: ptr("http://u"), AuthType: ptr("API_KEY")}, `{"endpoint":"http://u","authType":"API_KEY"}`},
	}
	for _, c := range cases {
		got, err := json.Marshal(c.u)
		if err != nil || string(got) != c.want {
			t.Errorf("got %s (%v), want %s", got, err, c.want)
		}
	}
}
