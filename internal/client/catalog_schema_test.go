package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetCatalogSchemaStripsMetadata(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		w.Write([]byte(`{"$id":"https://x/y","type":"object","name":"s","status":"APPLIED"}`))
	}))
	c := testClient(t, srv)

	got, err := c.GetCatalogSchema(context.Background(), "platform", "a/b")
	if err != nil {
		t.Fatalf("GetCatalogSchema: %v", err)
	}
	if gotPath != "/v1/catalog_schemas/platform/a%2Fb" {
		t.Fatalf("unexpected path %q", gotPath)
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

func TestGetCatalogSchemaNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	got, err := c.GetCatalogSchema(context.Background(), "b", "p")
	if err != nil || got != nil {
		t.Fatalf("expected nil, nil; got %s, %v", got, err)
	}
}

func TestGetCatalogSchemaServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	c := testClient(t, srv)

	if _, err := c.GetCatalogSchema(context.Background(), "b", "p"); err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestSaveCatalogSchemaSendsDocument(t *testing.T) {
	var method, body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		w.Write([]byte(`{"name":"s"}`))
	}))
	c := testClient(t, srv)

	if err := c.SaveCatalogSchema(context.Background(), "b", "p", CatalogSchema(`{"$id":"u"}`)); err != nil {
		t.Fatalf("SaveCatalogSchema: %v", err)
	}
	if method != http.MethodPut || body != `{"$id":"u"}` {
		t.Fatalf("unexpected request %s %s", method, body)
	}
}

func TestDeleteCatalogSchemaTreats404AsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	if err := c.DeleteCatalogSchema(context.Background(), "b", "p"); err != nil {
		t.Fatalf("expected success on 404, got %v", err)
	}
}
