package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetApplicationSuccess(t *testing.T) {
	var gotPath, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		gotKey = r.Header.Get("Api-Key")
		for _, h := range []string{"If-Match", "If-None-Match"} {
			if r.Header.Get(h) != "" {
				t.Errorf("unexpected %s header", h)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"name":"applications/public/team/filter",
			"endpoint":"http://hook/chat",
			"display_name":"Hook",
			"forward_auth_token":true,
			"max_retry_attempts":3,
			"user_roles":["a"],
			"default_headers":{"X":"y"},
			"interceptors":["i"],
			"application_type_schema_id":"https://s/x",
			"application_properties":{"p":1},
			"defaults":{"k":"v"},
			"author":"someone","created_at":1,"updated_at":2,"status":"valid","routes":{}
		}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetApplication(context.Background(), "public", "team/filter")
	if err != nil {
		t.Fatalf("GetApplication: %v", err)
	}
	if got == nil || got.Endpoint == nil || *got.Endpoint != "http://hook/chat" {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.DisplayName == nil || *got.DisplayName != "Hook" || got.ForwardAuthToken == nil || !*got.ForwardAuthToken ||
		got.MaxRetryAttempts == nil || *got.MaxRetryAttempts != 3 {
		t.Fatalf("snake_case scalars not decoded: %+v", got)
	}
	if got.UserRoles == nil || (*got.UserRoles)[0] != "a" || got.DefaultHeaders["X"] != "y" ||
		got.Interceptors == nil || (*got.Interceptors)[0] != "i" || got.ApplicationTypeSchemaID == nil || string(got.ApplicationProperties) != `{"p":1}` {
		t.Fatalf("collections not decoded: %+v", got)
	}
	if want := "/v1/applications/public/team/filter"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}
	if gotKey != "test-api-key" {
		t.Fatalf("expected Api-Key header, got %q", gotKey)
	}
}

func TestGetApplicationNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	got, err := c.GetApplication(context.Background(), "platform", "missing")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil on 404, got %+v, %v", got, err)
	}
}

func TestGetApplicationServerErrorIncludesStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("nope"))
	}))
	c := testClient(t, srv)

	_, err := c.GetApplication(context.Background(), "platform", "x")
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected error with status and body, got %v", err)
	}
}

func TestSaveApplicationSendsOnlySetFields(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.EscapedPath()
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &gotBody); err != nil {
			t.Errorf("body is not JSON: %v", err)
		}
		for _, h := range []string{"If-Match", "If-None-Match"} {
			if r.Header.Get(h) != "" {
				t.Errorf("unexpected %s header", h)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"filter"}`))
	}))
	c := testClient(t, srv)

	endpoint := "http://hook/chat"
	if err := c.SaveApplication(context.Background(), "platform", "filter", &Application{Endpoint: &endpoint}); err != nil {
		t.Fatalf("SaveApplication: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v1/applications/platform/filter" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	if len(gotBody) != 1 || gotBody["endpoint"] != endpoint {
		t.Fatalf("expected only endpoint, got %v", gotBody)
	}
}

func TestSaveApplicationUsesSnakeCaseKeys(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Write([]byte(`{"name":"t"}`))
	}))
	c := testClient(t, srv)

	s, b, n, l := "x", true, int64(2), []string{"a"}
	err := c.SaveApplication(context.Background(), "platform", "t", &Application{
		Description: &s, DisplayName: &s, DisplayVersion: &s, IconURL: &s, EditorURL: &s, ViewerURL: &s, ApplicationTypeSchemaID: &s, CatalogSchemaID: &s, Interceptors: &l,
		ForwardAuthToken: &b, MaxRetryAttempts: &n, UserRoles: &l, DefaultHeaders: map[string]string{"h": "v"},
		Defaults: json.RawMessage(`{"a":1}`), ApplicationProperties: json.RawMessage(`{"a":1}`), CatalogProperties: json.RawMessage(`{"a":1}`),
	})
	if err != nil {
		t.Fatalf("SaveApplication: %v", err)
	}
	for _, key := range []string{
		"description", "display_name", "display_version", "icon_url", "editor_url", "viewer_url", "application_type_schema_id",
		"catalog_schema_id", "interceptors", "forward_auth_token", "max_retry_attempts", "user_roles", "default_headers",
		"defaults", "application_properties", "catalog_properties",
	} {
		if _, ok := gotBody[key]; !ok {
			t.Errorf("body lacks snake_case key %q: %v", key, gotBody)
		}
	}
}

func TestSaveApplicationServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad application"))
	}))
	c := testClient(t, srv)

	err := c.SaveApplication(context.Background(), "platform", "x", &Application{})
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "bad application") {
		t.Fatalf("expected error with status and body, got %v", err)
	}
}

func TestDeleteApplicationTreats404AsSuccess(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	if err := c.DeleteApplication(context.Background(), "platform", "gone"); err != nil {
		t.Fatalf("expected success on 404, got %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("expected DELETE, got %s", gotMethod)
	}
}

func TestApplicationURLKeepsPathSlashesAndEscapesSegments(t *testing.T) {
	c := &Client{baseURL: "http://core"}
	for _, tc := range []struct{ bucket, path, want string }{
		{"platform", "filter", "http://core/v1/applications/platform/filter"},
		{"public", "team/sub/app", "http://core/v1/applications/public/team/sub/app"},
		{"b/x", "a b/c?d", "http://core/v1/applications/b%2Fx/a%20b/c%3Fd"},
	} {
		if got := c.applicationURL(tc.bucket, tc.path); got != tc.want {
			t.Errorf("applicationURL(%q, %q) = %q, want %q", tc.bucket, tc.path, got, tc.want)
		}
	}
}

func TestGetFileApplication(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		if gotPath == "/v1/admin/config/file/applications/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"a/b","endpoint":"http://hook/chat"}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetFileApplication(context.Background(), "a/b")
	if err != nil || got == nil || got.Name != "a/b" {
		t.Fatalf("unexpected result %+v, %v", got, err)
	}
	if want := "/v1/admin/config/file/applications/a%2Fb"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}

	got, err = c.GetFileApplication(context.Background(), "missing")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil on 404, got %+v, %v", got, err)
	}
}
