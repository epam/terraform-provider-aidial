package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetFileModelSuccess(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"gpt-5.6-terra-2026-07-09","type":"CHAT"}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetFileModel(context.Background(), "gpt-5.6-terra-2026-07-09")
	if err != nil {
		t.Fatalf("GetFileModel: %v", err)
	}
	if got == nil || got.Name != "gpt-5.6-terra-2026-07-09" || got.Type != "CHAT" {
		t.Fatalf("unexpected result: %+v", got)
	}
	if want := "/v1/admin/config/file/models/gpt-5.6-terra-2026-07-09"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}
}

func TestGetFileModelNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	got, err := c.GetFileModel(context.Background(), "does-not-exist")
	if err != nil {
		t.Fatalf("expected nil error on 404, got: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil result on 404, got: %+v", got)
	}
}

func TestGetFileModelEscapesDotsInName(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"a/b"}`))
	}))
	c := testClient(t, srv)

	if _, err := c.GetFileModel(context.Background(), "a/b"); err != nil {
		t.Fatalf("GetFileModel: %v", err)
	}
	// url.PathEscape encodes "/" as %2F rather than treating it as a
	// path separator, so a name containing a slash stays a single
	// path segment.
	if want := "/v1/admin/config/file/models/a/b"; gotPath != want {
		t.Fatalf("expected request path %q, got %q", want, gotPath)
	}
}
