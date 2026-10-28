package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPathParametersEscapedAsSingleSegments(t *testing.T) {
	type getter func(c *Client, bucket, path string) error

	getters := map[string]struct {
		prefix string
		get    getter
	}{
		"role":  {"/v1/roles/", func(c *Client, b, p string) error { _, err := c.GetRole(context.Background(), b, p); return err }},
		"key":   {"/v1/keys/", func(c *Client, b, p string) error { _, err := c.GetKey(context.Background(), b, p); return err }},
		"model": {"/v1/models/", func(c *Client, b, p string) error { _, err := c.GetModel(context.Background(), b, p); return err }},
		"route": {"/v1/routes/", func(c *Client, b, p string) error { _, err := c.GetRoute(context.Background(), b, p); return err }},
		"catalog schema": {"/v1/catalog_schemas/", func(c *Client, b, p string) error {
			_, err := c.GetCatalogSchema(context.Background(), b, p)
			return err
		}},
	}

	cases := []struct {
		name, bucket, path string
		wantPath           string
	}{
		{"slash in path", "platform", "team/admin", "platform/team%2Fadmin"},
		{"slash in bucket", "a/b", "c", "a%2Fb/c"},
		{"query delimiter", "platform", "a?b", "platform/a%3Fb"},
		{"fragment and percent", "platform", "a#b%c", "platform/a%23b%25c"},
		{"space", "platform", "a b", "platform/a%20b"},
		{"plain names unchanged", "platform", "gpt-4_o.v1:0", "platform/gpt-4_o.v1:0"},
	}

	for kind, g := range getters {
		for _, tc := range cases {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				var gotPath, gotQuery string
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					gotPath, gotQuery = r.URL.EscapedPath(), r.URL.RawQuery
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(`{}`))
				}))
				c := testClient(t, srv)

				if err := g.get(c, tc.bucket, tc.path); err != nil {
					t.Fatalf("get: %v", err)
				}
				if want := g.prefix + tc.wantPath; gotPath != want {
					t.Fatalf("request path: want %q, got %q", want, gotPath)
				}
				if gotQuery != "" {
					t.Fatalf("unexpected query string %q", gotQuery)
				}
			})
		}
	}
}

func TestGetFileModelNameEscaped(t *testing.T) {
	cases := map[string]string{
		"a/b":                    "/v1/admin/config/file/models/a%2Fb",
		"a?b":                    "/v1/admin/config/file/models/a%3Fb",
		"gpt-5.6-terra-2026:0_x": "/v1/admin/config/file/models/gpt-5.6-terra-2026:0_x",
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			var gotPath, gotQuery string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath, gotQuery = r.URL.EscapedPath(), r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(`{}`))
			}))
			c := testClient(t, srv)

			if _, err := c.GetFileModel(context.Background(), name); err != nil {
				t.Fatalf("GetFileModel: %v", err)
			}
			if gotPath != want || gotQuery != "" {
				t.Fatalf("want path %q and no query, got %q and %q", want, gotPath, gotQuery)
			}
		})
	}
}
