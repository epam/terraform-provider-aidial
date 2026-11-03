package client

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestAllowedIPAddressRangesMarshalsCIDRList(t *testing.T) {
	key := Key{Role: "r", AllowedIPAddressRanges: &AllowedIPAddressRanges{CIDRs: []string{"10.0.0.0/8"}}}
	data, err := json.Marshal(key)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	_ = json.Unmarshal(data, &got)
	if !reflect.DeepEqual(got["allowedIpAddressRanges"], []any{"10.0.0.0/8"}) {
		t.Fatalf("unexpected body: %s", data)
	}

	data, _ = json.Marshal(Key{AllowedIPAddressRanges: &AllowedIPAddressRanges{CIDRs: []string{}}})
	if string(data) != `{"allowedIpAddressRanges":[]}` {
		t.Fatalf("an empty list must be sent, got %s", data)
	}
	data, _ = json.Marshal(Key{})
	if string(data) != `{}` {
		t.Fatalf("a nil list must be omitted, got %s", data)
	}
}

func TestAllowedIPAddressRangesUnmarshal(t *testing.T) {
	cases := map[string]struct {
		body        string
		want        []string
		undecodable bool
	}{
		"object with base64 bytes":   {`{"ranges":[{"mask":"/////w==","maskedBaseIp":"wAAAAA=="}]}`, []string{"192.0.0.0/32"}, false},
		"object with signed bytes":   {`{"ranges":[{"mask":[-1,-1,-1,0],"maskedBaseIp":[-58,51,100,0]}]}`, []string{"198.51.100.0/24"}, false},
		"object with base64 ipv4":    {`{"ranges":[{"mask":"////AA==","maskedBaseIp":"xjNkAA=="}]}`, []string{"198.51.100.0/24"}, false},
		"empty object":               {`{"ranges":[]}`, []string{}, false},
		"cidr strings":               {`["198.51.100.14/24"]`, []string{"198.51.100.0/24"}, false},
		"unsupported representation": {`{"ranges":[{"mask":{"x":1},"maskedBaseIp":1}]}`, nil, true},
	}
	for name, tc := range cases {
		var key Key
		if err := json.Unmarshal([]byte(`{"allowedIpAddressRanges":`+tc.body+`}`), &key); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got := key.AllowedIPAddressRanges
		if got == nil || got.Undecodable != tc.undecodable {
			t.Fatalf("%s: unexpected result %+v", name, got)
		}
		if !tc.undecodable && !reflect.DeepEqual(got.CIDRs, tc.want) {
			t.Fatalf("%s: got %v, want %v", name, got.CIDRs, tc.want)
		}
	}
}
