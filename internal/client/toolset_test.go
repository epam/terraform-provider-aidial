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

func TestGetToolsetSuccess(t *testing.T) {
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
			"name":"toolsets/public/team/tool",
			"endpoint":"http://tool/mcp",
			"transport":"HTTP",
			"allowed_tools":["a","b"],
			"vendor_website":"https://example.invalid",
			"forward_auth_token":true,
			"max_retry_attempts":3,
			"description_keywords":["k"],
			"auth_settings":{"authentication_type":"OAUTH","client_id":"id","client_secret_hint":"abc","user_level_auth_status":"SIGNED_OUT"},
			"created_at":1,
			"updated_at":2
		}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetToolset(context.Background(), "public", "team/tool")
	if err != nil {
		t.Fatalf("GetToolset: %v", err)
	}
	if got == nil || got.Endpoint == nil || *got.Endpoint != "http://tool/mcp" {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.AllowedTools == nil || len(*got.AllowedTools) != 2 {
		t.Fatalf("allowed_tools not decoded: %+v", got.AllowedTools)
	}
	if got.VendorWebsite == nil || *got.VendorWebsite != "https://example.invalid" {
		t.Fatalf("vendor_website not decoded: %+v", got.VendorWebsite)
	}
	if got.ForwardAuthToken == nil || !*got.ForwardAuthToken || got.MaxRetryAttempts == nil || *got.MaxRetryAttempts != 3 {
		t.Fatalf("snake_case scalars not decoded: %+v", got)
	}
	if got.DescriptionKeywords == nil || (*got.DescriptionKeywords)[0] != "k" {
		t.Fatalf("description_keywords not decoded: %+v", got.DescriptionKeywords)
	}
	if got.AuthSettings == nil || got.AuthSettings.ClientID == nil || *got.AuthSettings.ClientID != "id" {
		t.Fatalf("authSettings not decoded: %+v", got.AuthSettings)
	}
	if got.AuthSettings.ClientSecret != nil {
		t.Fatalf("client_secret must stay nil when Core omits it, got %q", *got.AuthSettings.ClientSecret)
	}
	// Core reads a toolset path as folders: the slash must stay a slash.
	if want := "/v1/toolsets/public/team/tool"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}
	if gotKey != "test-api-key" {
		t.Fatalf("expected Api-Key header, got %q", gotKey)
	}
}

func TestGetToolsetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	got, err := c.GetToolset(context.Background(), "platform", "missing")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil on 404, got %+v, %v", got, err)
	}
}

func TestGetToolsetServerErrorIncludesStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("nope"))
	}))
	c := testClient(t, srv)

	_, err := c.GetToolset(context.Background(), "platform", "x")
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected error with status and body, got %v", err)
	}
}

func TestSaveToolsetSendsOnlySetFields(t *testing.T) {
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
		w.Write([]byte(`{"name":"toolsets/platform/tool"}`))
	}))
	c := testClient(t, srv)

	endpoint := "http://tool/mcp"
	secret := "s3cr3t"
	err := c.SaveToolset(context.Background(), "platform", "tool", &ToolSet{
		Endpoint:     &endpoint,
		AuthSettings: &ToolSetAuthSettings{ClientSecret: &secret},
	})
	if err != nil {
		t.Fatalf("SaveToolset: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v1/toolsets/platform/tool" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	if len(gotBody) != 2 || gotBody["endpoint"] != endpoint {
		t.Fatalf("expected only endpoint and auth_settings, got %v", gotBody)
	}
	if auth, _ := gotBody["auth_settings"].(map[string]any); auth["client_secret"] != secret {
		t.Fatalf("client_secret must be sent, got %v", gotBody["auth_settings"])
	}
}

func TestSaveToolsetUsesSnakeCaseKeys(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Write([]byte(`{"name":"t"}`))
	}))
	c := testClient(t, srv)

	s, b, n, l := "x", true, int64(2), []string{"a"}
	err := c.SaveToolset(context.Background(), "platform", "t", &ToolSet{
		DisplayName: &s, ForwardAuthToken: &b, ForwardPerRequestKey: &b, MaxRetryAttempts: &n, MaxInputAttachments: &n,
		DescriptionKeywords: &l, InputAttachmentTypes: &l, UserRoles: &l, AllowedTools: &l,
		IconURL: &s, DisplayVersion: &s, ResponsesEndpoint: &s, BaseURL: &s, OverrideName: &s, CatalogSchemaID: &s,
		Provider: &s, VendorWebsite: &s, DefaultHeaders: map[string]string{"h": "v"},
		Defaults: json.RawMessage(`{"a":1}`), ResponsesDefaults: json.RawMessage(`{"a":1}`), CatalogProperties: json.RawMessage(`{"a":1}`),
	})
	if err != nil {
		t.Fatalf("SaveToolset: %v", err)
	}
	for _, key := range []string{
		"display_name", "forward_auth_token", "forward_per_request_key", "max_retry_attempts", "max_input_attachments",
		"description_keywords", "input_attachment_types", "user_roles", "allowed_tools", "icon_url", "display_version",
		"responses_endpoint", "base_url", "override_name", "catalog_schema_id", "provider", "vendor_website",
		"default_headers", "defaults", "responses_defaults", "catalog_properties",
	} {
		if _, ok := gotBody[key]; !ok {
			t.Errorf("body lacks snake_case key %q: %v", key, gotBody)
		}
	}
	for key := range gotBody {
		if strings.ToLower(key) != key {
			t.Errorf("body key %q is not snake_case", key)
		}
	}
}

func TestSaveToolsetServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad toolset"))
	}))
	c := testClient(t, srv)

	err := c.SaveToolset(context.Background(), "platform", "x", &ToolSet{})
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "bad toolset") {
		t.Fatalf("expected error with status and body, got %v", err)
	}
}

func TestDeleteToolsetTreats404AsSuccess(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	if err := c.DeleteToolset(context.Background(), "platform", "gone"); err != nil {
		t.Fatalf("expected success on 404, got %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("expected DELETE, got %s", gotMethod)
	}
}

func TestToolsetURLKeepsPathSlashesAndEscapesSegments(t *testing.T) {
	c := &Client{baseURL: "http://core"}
	for _, tc := range []struct{ bucket, path, want string }{
		{"platform", "tool", "http://core/v1/toolsets/platform/tool"},
		{"public", "team/sub/tool", "http://core/v1/toolsets/public/team/sub/tool"},
		{"b/x", "a b/c?d", "http://core/v1/toolsets/b%2Fx/a%20b/c%3Fd"},
	} {
		if got := c.toolsetURL(tc.bucket, tc.path); got != tc.want {
			t.Errorf("toolsetURL(%q, %q) = %q, want %q", tc.bucket, tc.path, got, tc.want)
		}
	}
}

func TestGetFileToolset(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		if gotPath == "/v1/admin/config/file/toolsets/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"a/b","endpoint":"http://tool/mcp"}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetFileToolset(context.Background(), "a/b")
	if err != nil || got == nil || got.Name != "a/b" {
		t.Fatalf("unexpected result %+v, %v", got, err)
	}
	if want := "/v1/admin/config/file/toolsets/a%2Fb"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}

	got, err = c.GetFileToolset(context.Background(), "missing")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil on 404, got %+v, %v", got, err)
	}
}

func TestRedactSensitiveJSONToolsetClientSecret(t *testing.T) {
	secret := "oauth-client-secret" //nolint:gosec // G101: dummy value used to prove it is redacted
	payload, err := json.Marshal(&ToolSet{
		Name:         "t",
		AuthSettings: &ToolSetAuthSettings{ClientID: ptr("id"), ClientSecret: &secret},
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	redacted := string(redactSensitiveJSON(payload))
	if strings.Contains(redacted, secret) {
		t.Fatalf("client_secret leaked into log output: %s", redacted)
	}
	if !strings.Contains(redacted, `"client_id":"id"`) {
		t.Fatalf("expected client_id to survive redaction, got: %s", redacted)
	}
}
