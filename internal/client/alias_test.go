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

func TestGetAliasSuccess(t *testing.T) {
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
			"name":"fast",
			"presentAs":"model",
			"target":{"path":"applications/chat"},
			"userRoles":["a"],
			"displayName":"Fast",
			"description":"d",
			"displayVersion":"1",
			"iconUrl":"http://i",
			"intro":"hello",
			"descriptionKeywords":["k"],
			"status":"valid"
		}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetAlias(context.Background(), "public", "team/fast")
	if err != nil {
		t.Fatalf("GetAlias: %v", err)
	}
	if got == nil || got.PresentAs != "model" || got.Target == nil || got.Target.Path != "applications/chat" {
		t.Fatalf("unexpected result: %+v", got)
	}
	if got.DisplayName == nil || *got.DisplayName != "Fast" || got.Intro == nil || *got.Intro != "hello" ||
		got.IconURL == nil || *got.IconURL != "http://i" || got.DisplayVersion == nil || got.Description == nil {
		t.Fatalf("camelCase scalars not decoded: %+v", got)
	}
	if got.UserRoles == nil || (*got.UserRoles)[0] != "a" || got.DescriptionWords == nil || (*got.DescriptionWords)[0] != "k" {
		t.Fatalf("collections not decoded: %+v", got)
	}
	if want := "/v1/aliases/public/team%2Ffast"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}
	if gotKey != "test-api-key" {
		t.Fatalf("expected Api-Key header, got %q", gotKey)
	}
}

func TestGetAliasNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	got, err := c.GetAlias(context.Background(), "public", "missing")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil on 404, got %+v, %v", got, err)
	}
}

func TestGetAliasServerErrorIncludesStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("nope"))
	}))
	c := testClient(t, srv)

	_, err := c.GetAlias(context.Background(), "public", "x")
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected error with status and body, got %v", err)
	}
}

func TestSaveAliasSendsOnlySetFields(t *testing.T) {
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
		w.Write([]byte(`{"name":"fast"}`))
	}))
	c := testClient(t, srv)

	err := c.SaveAlias(context.Background(), "public", "fast", &Alias{
		PresentAs: "model", Target: &AliasTarget{Path: "applications/chat"},
	})
	if err != nil {
		t.Fatalf("SaveAlias: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v1/aliases/public/fast" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	if len(gotBody) != 2 || gotBody["presentAs"] != "model" {
		t.Fatalf("expected only presentAs and target, got %v", gotBody)
	}
	target, _ := gotBody["target"].(map[string]any)
	if len(target) != 1 || target["path"] != "applications/chat" {
		t.Fatalf("expected target with only path, got %v", gotBody["target"])
	}
}

func TestSaveAliasUsesCamelCaseKeys(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &gotBody)
		w.Write([]byte(`{"name":"t"}`))
	}))
	c := testClient(t, srv)

	s, l := "x", []string{"a"}
	err := c.SaveAlias(context.Background(), "public", "t", &Alias{
		Name: "t", PresentAs: "toolset", Target: &AliasTarget{Path: "toolsets/x"},
		UserRoles: &l, DisplayName: &s, Description: &s, DisplayVersion: &s, IconURL: &s, Intro: &s, DescriptionWords: &l,
	})
	if err != nil {
		t.Fatalf("SaveAlias: %v", err)
	}
	for _, key := range []string{
		"name", "presentAs", "target", "userRoles", "displayName", "description", "displayVersion", "iconUrl", "intro", "descriptionKeywords",
	} {
		if _, ok := gotBody[key]; !ok {
			t.Errorf("body lacks camelCase key %q: %v", key, gotBody)
		}
	}
}

func TestSaveAliasServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad alias"))
	}))
	c := testClient(t, srv)

	err := c.SaveAlias(context.Background(), "public", "x", &Alias{})
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "bad alias") {
		t.Fatalf("expected error with status and body, got %v", err)
	}
}

func TestDeleteAliasTreats404AsSuccess(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	if err := c.DeleteAlias(context.Background(), "public", "gone"); err != nil {
		t.Fatalf("expected success on 404, got %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("expected DELETE, got %s", gotMethod)
	}
}

func TestAliasURLEscapesSegments(t *testing.T) {
	c := &Client{baseURL: "http://core"}
	for _, tc := range []struct{ bucket, path, want string }{
		{"public", "fast", "http://core/v1/aliases/public/fast"},
		{"platform", "team/fast", "http://core/v1/aliases/platform/team%2Ffast"},
		{"public", "a?b", "http://core/v1/aliases/public/a%3Fb"},
	} {
		if got := c.aliasURL(tc.bucket, tc.path); got != tc.want {
			t.Errorf("aliasURL(%q, %q) = %q, want %q", tc.bucket, tc.path, got, tc.want)
		}
	}
}

func TestGetFileAlias(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		if gotPath == "/v1/admin/config/file/aliases/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"a/b","presentAs":"application","target":{"path":"models/gpt"}}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetFileAlias(context.Background(), "a/b")
	if err != nil || got == nil || got.PresentAs != "application" || got.Target == nil || got.Target.Path != "models/gpt" {
		t.Fatalf("unexpected result %+v, %v", got, err)
	}
	if want := "/v1/admin/config/file/aliases/a%2Fb"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}

	got, err = c.GetFileAlias(context.Background(), "missing")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil on 404, got %+v, %v", got, err)
	}
}
