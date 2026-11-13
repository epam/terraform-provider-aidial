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

func TestGetTranslatorSuccess(t *testing.T) {
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
		w.Write([]byte(`{"in":"openaiChatCompletions","out":"anthropicMessages","baseUrl":"http://x:8000","name":"t","status":"valid"}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetTranslator(context.Background(), "platform", "team/t")
	if err != nil {
		t.Fatalf("GetTranslator: %v", err)
	}
	want := &Translator{BaseURL: "http://x:8000", In: "openaiChatCompletions", Out: "anthropicMessages"}
	if got == nil || *got != *want {
		t.Fatalf("unexpected result: %+v", got)
	}
	if wantPath := "/v1/translators/platform/team%2Ft"; gotPath != wantPath {
		t.Fatalf("expected request path %q, got %q", wantPath, gotPath)
	}
	if gotKey != "test-api-key" {
		t.Fatalf("expected Api-Key header, got %q", gotKey)
	}
}

func TestGetTranslatorNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	got, err := c.GetTranslator(context.Background(), "platform", "missing")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil on 404, got %+v, %v", got, err)
	}
}

func TestGetTranslatorServerErrorIncludesStatusAndBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("nope"))
	}))
	c := testClient(t, srv)

	_, err := c.GetTranslator(context.Background(), "platform", "x")
	if err == nil || !strings.Contains(err.Error(), "403") || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected error with status and body, got %v", err)
	}
}

func TestSaveTranslatorSendsCamelCaseBody(t *testing.T) {
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
		w.Write([]byte(`{"name":"t"}`))
	}))
	c := testClient(t, srv)

	err := c.SaveTranslator(context.Background(), "platform", "t", &Translator{
		BaseURL: "http://x:8000", In: "openaiChatCompletions", Out: "anthropicMessages",
	})
	if err != nil {
		t.Fatalf("SaveTranslator: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/v1/translators/platform/t" {
		t.Fatalf("unexpected request %s %s", gotMethod, gotPath)
	}
	if len(gotBody) != 3 || gotBody["baseUrl"] != "http://x:8000" ||
		gotBody["in"] != "openaiChatCompletions" || gotBody["out"] != "anthropicMessages" {
		t.Fatalf("expected baseUrl, in and out, got %v", gotBody)
	}
}

func TestSaveTranslatorServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad translator"))
	}))
	c := testClient(t, srv)

	err := c.SaveTranslator(context.Background(), "platform", "x", &Translator{})
	if err == nil || !strings.Contains(err.Error(), "400") || !strings.Contains(err.Error(), "bad translator") {
		t.Fatalf("expected error with status and body, got %v", err)
	}
}

func TestDeleteTranslatorTreats404AsSuccess(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	if err := c.DeleteTranslator(context.Background(), "platform", "gone"); err != nil {
		t.Fatalf("expected success on 404, got %v", err)
	}
	if gotMethod != http.MethodDelete {
		t.Fatalf("expected DELETE, got %s", gotMethod)
	}
}

func TestTranslatorURLEscapesSegments(t *testing.T) {
	c := &Client{baseURL: "http://core"}
	for _, tc := range []struct{ bucket, path, want string }{
		{"platform", "t", "http://core/v1/translators/platform/t"},
		{"platform", "team/openai", "http://core/v1/translators/platform/team%2Fopenai"},
		{"platform", "a?b", "http://core/v1/translators/platform/a%3Fb"},
	} {
		if got := c.translatorURL(tc.bucket, tc.path); got != tc.want {
			t.Errorf("translatorURL(%q, %q) = %q, want %q", tc.bucket, tc.path, got, tc.want)
		}
	}
}

func TestGetFileTranslator(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		if gotPath == "/v1/admin/config/file/translators/missing" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"a/b","baseUrl":"http://x","in":"openaiEmbeddings","out":"openaiResponses"}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetFileTranslator(context.Background(), "a/b")
	if err != nil || got == nil || got.BaseURL != "http://x" || got.In != "openaiEmbeddings" || got.Out != "openaiResponses" {
		t.Fatalf("unexpected result %+v, %v", got, err)
	}
	if want := "/v1/admin/config/file/translators/a%2Fb"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}

	got, err = c.GetFileTranslator(context.Background(), "missing")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil on 404, got %+v, %v", got, err)
	}
}
