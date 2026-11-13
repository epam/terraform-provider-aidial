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

func TestGetInterceptorSuccess(t *testing.T) {
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
			"name":"interceptors/platform/team%2Ffilter",
			"endpoint":"http://hook/chat",
			"displayName":"Hook",
			"forwardAuthToken":true,
			"maxRetryAttempts":3,
			"userRoles":["a"],
			"defaultHeaders":{"X":"y"},
			"defaults":{"k":"v"},
			"author":"someone","createdAt":1,"updatedAt":2,"status":"valid"
		}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetInterceptor(context.Background(), "platform", "team/filter")
	if err != nil {
		t.Fatalf("GetInterceptor: %v", err)
	}
	if got == nil || got.Endpoint == nil || *got.Endpoint != "http://hook/chat" {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.DisplayName == nil || *got.DisplayName != "Hook" || got.ForwardAuthToken == nil || !*got.ForwardAuthToken ||
		got.MaxRetryAttempts == nil || *got.MaxRetryAttempts != 3 {
		t.Fatalf("camelCase scalars not decoded: %+v", got)
	}
	if got.UserRoles == nil || (*got.UserRoles)[0] != "a" || got.DefaultHeaders["X"] != "y" {
		t.Fatalf("collections not decoded: %+v", got)
	}
	if want := "/v1/interceptors/platform/team%2Ffilter"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}
	if gotKey != "test-api-key" {
		t.Fatalf("expected Api-Key header, got %q", gotKey)
	}
}

func TestGetInterceptorNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	got, err := c.GetInterceptor(context.Background(), "platform", "missing")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil on 404, got %+v, %v", got, err)
	}
}

func TestGetInterceptorServerErrorIncludesStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("nope"))
	}))
	c := testClient(t, srv)

	_, err := c.GetInterceptor(context.Background(), "platform", "x")
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected error with status and body, got %v", err)
	}
}

func TestSaveInterceptorSendsOnlySetFields(t *testing.T) {
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
	if err := c.SaveInterceptor(context.Background(), "platform", "filter", &Interceptor{Endpoint: &endpoint}); err != nil {
		t.Fatalf("SaveInterceptor: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v1/interceptors/platform/filter" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	if len(gotBody) != 1 || gotBody["endpoint"] != endpoint {
		t.Fatalf("expected only endpoint, got %v", gotBody)
	}
}

func TestSaveInterceptorUsesCamelCaseKeys(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Write([]byte(`{"name":"t"}`))
	}))
	c := testClient(t, srv)

	s, b, n, l := "x", true, int64(2), []string{"a"}
	err := c.SaveInterceptor(context.Background(), "platform", "t", &Interceptor{
		Description: &s, DisplayName: &s, DisplayVersion: &s, IconURL: &s, CatalogSchemaID: &s,
		ForwardAuthToken: &b, MaxRetryAttempts: &n, UserRoles: &l, DefaultHeaders: map[string]string{"h": "v"},
		Defaults: json.RawMessage(`{"a":1}`), CatalogProperties: json.RawMessage(`{"a":1}`),
	})
	if err != nil {
		t.Fatalf("SaveInterceptor: %v", err)
	}
	for _, key := range []string{
		"description", "displayName", "displayVersion", "iconUrl", "catalogSchemaId", "forwardAuthToken",
		"maxRetryAttempts", "userRoles", "defaultHeaders", "defaults", "catalogProperties",
	} {
		if _, ok := gotBody[key]; !ok {
			t.Errorf("body lacks camelCase key %q: %v", key, gotBody)
		}
	}
}

func TestSaveInterceptorServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad interceptor"))
	}))
	c := testClient(t, srv)

	err := c.SaveInterceptor(context.Background(), "platform", "x", &Interceptor{})
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "bad interceptor") {
		t.Fatalf("expected error with status and body, got %v", err)
	}
}

func TestDeleteInterceptorTreats404AsSuccess(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	if err := c.DeleteInterceptor(context.Background(), "platform", "gone"); err != nil {
		t.Fatalf("expected success on 404, got %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("expected DELETE, got %s", gotMethod)
	}
}

func TestInterceptorURLEscapesSegments(t *testing.T) {
	c := &Client{baseURL: "http://core"}
	for _, tc := range []struct{ bucket, path, want string }{
		{"platform", "filter", "http://core/v1/interceptors/platform/filter"},
		{"platform", "team/filter", "http://core/v1/interceptors/platform/team%2Ffilter"},
		{"platform", "a?b", "http://core/v1/interceptors/platform/a%3Fb"},
	} {
		if got := c.interceptorURL(tc.bucket, tc.path); got != tc.want {
			t.Errorf("interceptorURL(%q, %q) = %q, want %q", tc.bucket, tc.path, got, tc.want)
		}
	}
}

func TestGetFileInterceptor(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		if gotPath == "/v1/admin/config/file/interceptors/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"a/b","endpoint":"http://hook/chat"}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetFileInterceptor(context.Background(), "a/b")
	if err != nil || got == nil || got.Name != "a/b" {
		t.Fatalf("unexpected result %+v, %v", got, err)
	}
	if want := "/v1/admin/config/file/interceptors/a%2Fb"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}

	got, err = c.GetFileInterceptor(context.Background(), "missing")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil on 404, got %+v, %v", got, err)
	}
}
