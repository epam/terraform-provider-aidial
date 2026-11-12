package client

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClientRejectsBadCredentials(t *testing.T) {
	cases := map[string]Credentials{
		"none": {},
		"both": {APIKey: "k", IDToken: "a.b.c"},
	}
	for name, creds := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewClient("http://localhost", creds); err == nil {
				t.Fatalf("expected an error for %s credentials", name)
			}
		})
	}
}

func TestCredentialHeaders(t *testing.T) {
	cases := []struct {
		name          string
		creds         Credentials
		wantAPIKey    string
		wantAuthorize string
	}{
		{"api key", APIKeyCredentials("the-key"), "the-key", ""},
		{"id token", IDTokenCredentials("h.p.s"), "", "Bearer h.p.s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotAPIKey, gotAuth string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotAPIKey = r.Header.Get("Api-Key")
				gotAuth = r.Header.Get("Authorization")
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"name":"r"}`))
			}))
			defer srv.Close()

			c, err := NewClient(srv.URL, tc.creds)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			if _, err := c.GetRole(context.Background(), "platform", "r"); err != nil {
				t.Fatalf("GetRole: %v", err)
			}
			if gotAPIKey != tc.wantAPIKey {
				t.Errorf("Api-Key = %q, want %q", gotAPIKey, tc.wantAPIKey)
			}
			if gotAuth != tc.wantAuthorize {
				t.Errorf("Authorization = %q, want %q", gotAuth, tc.wantAuthorize)
			}
		})
	}
}

func TestRequestLogDoesNotContainCredentials(t *testing.T) {
	//nolint:gosec // G101: dummy values used to prove they are never logged
	const secretKey, secretToken = "super-secret-api-key", "eyJhbGciOi.secretpayload.secretsig"

	cases := []struct {
		name     string
		creds    Credentials
		secret   string
		wantKind string
	}{
		{"api key", APIKeyCredentials(secretKey), secretKey, "credential=api_key present=true"},
		{"id token", IDTokenCredentials(secretToken), secretToken, "credential=id_token present=true"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))
			defer srv.Close()

			var buf bytes.Buffer
			orig := log.Writer()
			log.SetOutput(&buf)
			defer log.SetOutput(orig)

			c, err := NewClient(srv.URL, tc.creds)
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			if err := c.SaveRole(context.Background(), "platform", "r", &Role{Name: "r"}); err != nil {
				t.Fatalf("SaveRole: %v", err)
			}

			out := buf.String()
			if strings.Contains(out, tc.secret) {
				t.Fatalf("log contains the credential: %s", out)
			}
			if !strings.Contains(out, tc.wantKind) {
				t.Fatalf("log %q lacks %q", out, tc.wantKind)
			}
		})
	}
}
