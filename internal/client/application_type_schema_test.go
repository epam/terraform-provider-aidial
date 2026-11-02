package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetApplicationTypeSchemaStripsMetadata(t *testing.T) {
	var gotPath, apiKey, ifNoneMatch string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		apiKey = r.Header.Get("Api-Key")
		ifNoneMatch = r.Header.Get("If-None-Match")
		w.Write([]byte(`{"$id":"https://x/y","type":"object","name":"s","status":"APPLIED"}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetApplicationTypeSchema(context.Background(), "platform", "apps/quick")
	if err != nil {
		t.Fatalf("GetApplicationTypeSchema: %v", err)
	}
	if gotPath != "/v1/schemas/platform/apps%2Fquick" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if apiKey == "" || ifNoneMatch != "" {
		t.Fatalf("unexpected headers: Api-Key=%q If-None-Match=%q", apiKey, ifNoneMatch)
	}
	var doc map[string]any
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["name"]; ok {
		t.Fatalf("name not stripped: %v", doc)
	}
	if _, ok := doc["status"]; ok {
		t.Fatalf("status not stripped: %v", doc)
	}
	if doc["$id"] != "https://x/y" || doc["type"] != "object" {
		t.Fatalf("schema keys lost: %v", doc)
	}
}

func TestGetApplicationTypeSchemaNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	got, err := c.GetApplicationTypeSchema(context.Background(), "b", "p")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil; got %s, %v", got, err)
	}
}

func TestGetApplicationTypeSchemaServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	c := testClient(t, srv)

	if _, err := c.GetApplicationTypeSchema(context.Background(), "b", "p"); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestSaveApplicationTypeSchemaSendsDocument(t *testing.T) {
	var method, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Write([]byte(`{"name":"s"}`))
	}))
	c := testClient(t, srv)

	if err := c.SaveApplicationTypeSchema(context.Background(), "b", "p", ApplicationTypeSchema(`{"$id":"u"}`)); err != nil {
		t.Fatalf("SaveApplicationTypeSchema: %v", err)
	}
	if method != http.MethodPut || body != `{"$id":"u"}` {
		t.Fatalf("unexpected request %s %s", method, body)
	}
}

func TestDeleteApplicationTypeSchemaTreats404AsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	if err := c.DeleteApplicationTypeSchema(context.Background(), "b", "p"); err != nil {
		t.Fatalf("expected success on 404, got %v", err)
	}
}

func TestGetFileApplicationTypeSchemaReturnsStatus(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Write([]byte(`{"$id":"https://x/y","name":"a/b","status":"APPLIED"}`))
	}))
	c := testClient(t, srv)

	got, status, err := c.GetFileApplicationTypeSchema(context.Background(), "a/b")
	if err != nil {
		t.Fatalf("GetFileApplicationTypeSchema: %v", err)
	}
	if gotPath != "/v1/admin/config/file/schemas/a%2Fb" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if status != "APPLIED" {
		t.Fatalf("unexpected status %q", status)
	}
	if string(got) != `{"$id":"https://x/y"}` {
		t.Fatalf("unexpected document %s", got)
	}
}

func TestGetFileApplicationTypeSchemaWithoutStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"$id":"u"}`))
	}))
	c := testClient(t, srv)

	_, status, err := c.GetFileApplicationTypeSchema(context.Background(), "n")
	if err != nil || status != "" {
		t.Fatalf("expected empty status, got %q, %v", status, err)
	}
}

func TestGetFileApplicationTypeSchemaNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	got, status, err := c.GetFileApplicationTypeSchema(context.Background(), "n")
	if err != nil || got != nil || status != "" {
		t.Fatalf("expected nil, empty, nil; got %s, %q, %v", got, status, err)
	}
}
