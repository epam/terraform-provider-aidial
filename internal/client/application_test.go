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

func TestApplicationNestedWireCasing(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("invalid body: %v", err)
		}
		w.Write([]byte(`{"name":"x"}`))
	}))
	c := testClient(t, srv)

	endpoint, transport, delivery := "http://h/mcp", "HTTP", "META"
	forward, rewrite := true, false
	secret, authType, scopes := "secret-12345", "OAUTH", []string{"s"}
	maxAttachments := int64(3)
	types, tools := []string{"*/*"}, []string{"t1"}
	domain := "x.com"
	app := &Application{
		Endpoint:             &endpoint,
		InputAttachmentTypes: &types,
		MaxInputAttachments:  &maxAttachments,
		Mcp: &ApplicationMcp{
			Endpoint: &endpoint, Transport: &transport, AllowedTools: &tools,
			ConfigDelivery: &delivery, ForwardPerRequestKey: &forward,
			McpApps: &ApplicationMcpApps{DomainOverride: &domain},
		},
		Routes: map[string]Route{"r1": {Paths: []string{"/a"}, RewritePath: &rewrite}},
		ExternalServices: map[string]ApplicationExternalService{"svc": {
			AuthSettings: &ApplicationExternalServiceAuth{AuthenticationType: &authType, ClientSecret: &secret, ScopesSupported: &scopes},
		}},
	}
	if err := c.SaveApplication(context.Background(), "platform", "a", app); err != nil {
		t.Fatalf("SaveApplication: %v", err)
	}

	if _, ok := body["input_attachment_types"]; !ok {
		t.Errorf("expected snake_case input_attachment_types, got %v", body)
	}
	if _, ok := body["max_input_attachments"]; !ok {
		t.Errorf("expected snake_case max_input_attachments, got %v", body)
	}
	mcp := body["mcp"].(map[string]any)
	for _, key := range []string{"allowedTools", "configDelivery", "forwardPerRequestKey", "mcpApps"} {
		if _, ok := mcp[key]; !ok {
			t.Errorf("mcp should use camelCase %q, got %v", key, mcp)
		}
	}
	route := body["routes"].(map[string]any)["r1"].(map[string]any)
	if _, ok := route["rewritePath"]; !ok {
		t.Errorf("route should use camelCase rewritePath, got %v", route)
	}
	auth := body["external_services"].(map[string]any)["svc"].(map[string]any)["auth_settings"].(map[string]any)
	for _, key := range []string{"authentication_type", "client_secret", "scopes_supported"} {
		if _, ok := auth[key]; !ok {
			t.Errorf("auth_settings should use snake_case %q, got %v", key, auth)
		}
	}
}

func TestGetApplicationDecodesNestedFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{
			"name":"x","endpoint":"http://x","reference":"r","intro":"i","override_name":"o","base_url":"http://b",
			"responses_endpoint":"http://r","responses_defaults":{"a":1},
			"input_attachment_types":["*/*"],"max_input_attachments":3,
			"description_keywords":[],"dependencies":["m1"],"allow_user_external_services":true,
			"mcp":{"endpoint":"http://h/mcp","transport":"HTTP","allowedTools":[],"configDelivery":"META","forwardPerRequestKey":true,"mcpApps":null},
			"routes":{"r1":{"paths":["/a"],"rewritePath":false,"order":2147483647,"attachmentPaths":{"requestBody":[],"responseBody":[]}}},
			"external_services":{"svc":{"display_name":"S","auth_settings":{"authentication_type":"OAUTH","client_id":"c","client_secret_hint":"2345","user_level_auth_status":"SIGNED_OUT"}}}
		}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetApplication(context.Background(), "platform", "a")
	if err != nil || got == nil {
		t.Fatalf("GetApplication: %v %v", got, err)
	}
	if got.Mcp == nil || got.Mcp.ConfigDelivery == nil || *got.Mcp.ConfigDelivery != "META" || got.Mcp.McpApps != nil ||
		got.Mcp.AllowedTools == nil || len(*got.Mcp.AllowedTools) != 0 {
		t.Fatalf("mcp not decoded: %+v", got.Mcp)
	}
	if r := got.Routes["r1"]; r.RewritePath == nil || *r.RewritePath || len(r.Paths) != 1 || r.Order == nil {
		t.Fatalf("route not decoded: %+v", r)
	}
	svc := got.ExternalServices["svc"]
	if svc.AuthSettings == nil || svc.AuthSettings.ClientSecret != nil || *svc.AuthSettings.ClientID != "c" {
		t.Fatalf("external service not decoded: %+v", svc)
	}
	if got.Reference == nil || *got.Reference != "r" || got.OverrideName == nil || got.BaseURL == nil || got.ResponsesEndpoint == nil ||
		string(got.ResponsesDefaults) != `{"a":1}` || got.AllowUserExternalServices == nil || !*got.AllowUserExternalServices ||
		got.Dependencies == nil || got.DescriptionKeywords == nil || got.InputAttachmentTypes == nil || got.MaxInputAttachments == nil {
		t.Fatalf("simple fields not decoded: %+v", got)
	}
}
