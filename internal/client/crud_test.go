package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient(t *testing.T, srv *httptest.Server) *Client {
	t.Cleanup(srv.Close)

	c, err := NewClient(srv.URL, "test-api-key")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return c
}

func TestGetResourceSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"my-role"}`))
	}))
	c := testClient(t, srv)

	got, err := getResource[Role](context.Background(), c, c.baseURL+"/v1/roles/b/p")
	if err != nil {
		t.Fatalf("getResource: %v", err)
	}
	if got == nil || got.Name != "my-role" {
		t.Fatalf("unexpected result: %+v", got)
	}
}

func TestGetResourceNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	got, err := getResource[Role](context.Background(), c, c.baseURL+"/v1/roles/b/p")
	if err != nil {
		t.Fatalf("expected nil error on 404, got: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil result on 404, got: %+v", got)
	}
}

func TestGetResourceServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	c := testClient(t, srv)

	if _, err := getResource[Role](context.Background(), c, c.baseURL+"/v1/roles/b/p"); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}

func TestSaveResourceSuccess(t *testing.T) {
	var receivedKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		receivedKey = r.Header.Get("Api-Key")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"my-role"}`))
	}))
	c := testClient(t, srv)

	err := saveResource(context.Background(), c, c.baseURL+"/v1/roles/b/p", &Role{Name: "my-role"})
	if err != nil {
		t.Fatalf("saveResource: %v", err)
	}
	if receivedKey != "test-api-key" {
		t.Fatalf("expected Api-Key header to be sent, got %q", receivedKey)
	}
}

func TestSaveResourceServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid"))
	}))
	c := testClient(t, srv)

	err := saveResource(context.Background(), c, c.baseURL+"/v1/roles/b/p", &Role{Name: "my-role"})
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
}

func TestDeleteResourceTreats404AsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	c := testClient(t, srv)

	if err := deleteResource(context.Background(), c, c.baseURL+"/v1/roles/b/p"); err != nil {
		t.Fatalf("expected 404 to be treated as success, got: %v", err)
	}
}

func TestDeleteResourceServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	c := testClient(t, srv)

	if err := deleteResource(context.Background(), c, c.baseURL+"/v1/roles/b/p"); err == nil {
		t.Fatal("expected error on 500, got nil")
	}
}
