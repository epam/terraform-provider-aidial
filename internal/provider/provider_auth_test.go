package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const validJWT = "aaa.bbb.ccc"

func TestResolveCredentialMatrix(t *testing.T) {
	cases := []struct {
		name         string
		in           credentialInputs
		wantAPIKey   string
		wantIDToken  string
		wantErr      bool
		wantWarnings []string // substrings, one warning expected when non-empty
	}{
		{name: "config api key only", in: credentialInputs{configAPIKey: "k"}, wantAPIKey: "k"},
		{name: "config id token only", in: credentialInputs{configIDToken: validJWT}, wantIDToken: validJWT},
		{name: "env id token only", in: credentialInputs{envIDToken: validJWT}, wantIDToken: validJWT},
		{name: "env api key only", in: credentialInputs{envAPIKey: "k"}, wantAPIKey: "k"},
		{name: "both in config", in: credentialInputs{configAPIKey: "k", configIDToken: validJWT}, wantErr: true},
		{name: "both in config ignore env", in: credentialInputs{configAPIKey: "k", configIDToken: validJWT, envAPIKey: "e"}, wantErr: true},
		{name: "nothing", in: credentialInputs{}, wantErr: true},
		{
			name: "config key beats both env", in: credentialInputs{configAPIKey: "k", envAPIKey: "e", envIDToken: validJWT},
			wantAPIKey: "k", wantWarnings: []string{envAPIKey, envIDToken},
		},
		{
			name: "config token beats env key", in: credentialInputs{configIDToken: validJWT, envAPIKey: "e"},
			wantIDToken: validJWT, wantWarnings: []string{envAPIKey},
		},
		{
			name: "config token beats both env", in: credentialInputs{configIDToken: validJWT, envAPIKey: "e", envIDToken: "x.y.z"},
			wantIDToken: validJWT, wantWarnings: []string{envAPIKey, envIDToken},
		},
		{
			name: "config key beats env token", in: credentialInputs{configAPIKey: "k", envIDToken: validJWT},
			wantAPIKey: "k", wantWarnings: []string{envIDToken},
		},
		{
			name: "env tie token wins", in: credentialInputs{envAPIKey: "e", envIDToken: validJWT},
			wantIDToken: validJWT, wantWarnings: []string{envAPIKey},
		},
		{name: "empty config falls through to env", in: credentialInputs{configIDToken: "  ", envAPIKey: "e"}, wantAPIKey: "e"},
		{name: "whitespace token trimmed", in: credentialInputs{configIDToken: " " + validJWT + "\n"}, wantIDToken: validJWT},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, warnings, err := resolveCredential(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.apiKey != tc.wantAPIKey || got.idToken != tc.wantIDToken {
				t.Fatalf("got %+v, want apiKey=%q idToken=%q", got, tc.wantAPIKey, tc.wantIDToken)
			}
			if len(tc.wantWarnings) == 0 {
				if len(warnings) != 0 {
					t.Fatalf("expected no warnings, got %v", warnings)
				}
				return
			}
			if len(warnings) != 1 {
				t.Fatalf("expected exactly one warning, got %v", warnings)
			}
			for _, name := range tc.wantWarnings {
				if !strings.Contains(warnings[0], name) {
					t.Errorf("warning %q does not name %s", warnings[0], name)
				}
			}
		})
	}
}

//nolint:gosec // G101: dummy values used to prove they never reach messages
func TestResolveCredentialMessagesNeverContainValues(t *testing.T) {
	secrets := credentialInputs{configAPIKey: "SECRETKEY", configIDToken: "SECRET.TOK.EN", envAPIKey: "ENVSECRET", envIDToken: "ENV.SEC.RET"}
	_, _, err := resolveCredential(secrets)
	if err == nil || strings.Contains(err.Error(), "SECRET") {
		t.Fatalf("error must exist and not contain values: %v", err)
	}
	_, warnings, err := resolveCredential(credentialInputs{configIDToken: "SECRET.TOK.EN", envAPIKey: "ENVSECRET", envIDToken: "ENV.SEC.RET"})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range warnings {
		if strings.Contains(w, "SECRET") || strings.Contains(w, "ENV.SEC") {
			t.Fatalf("warning leaks a value: %s", w)
		}
	}
}

func TestCheckJWTShape(t *testing.T) {
	for _, ok := range []string{validJWT, "eyJ.abc-_.def"} {
		if err := checkJWTShape(ok); err != nil {
			t.Errorf("%q should be accepted: %v", ok, err)
		}
	}
	for _, bad := range []string{"", "not-a-jwt", "a.b", "a.b.c.d", "a..c", ".b.c", "a.b."} {
		err := checkJWTShape(bad)
		if err == nil {
			t.Errorf("%q should be rejected", bad)
			continue
		}
		if bad != "" && strings.Contains(err.Error(), bad) {
			t.Errorf("error echoes the token %q: %v", bad, err)
		}
	}
}

func providerSchema(t *testing.T) provider.SchemaResponse {
	t.Helper()
	resp := provider.SchemaResponse{}
	New().Schema(context.Background(), provider.SchemaRequest{}, &resp)
	return resp
}

func TestProviderCredentialSchema(t *testing.T) {
	attrs := providerSchema(t).Schema.Attributes
	for _, name := range []string{"api_key", "id_token"} {
		a, ok := attrs[name]
		if !ok {
			t.Fatalf("%s missing", name)
		}
		if !a.IsSensitive() || !a.IsOptional() || a.IsRequired() {
			t.Errorf("%s must be optional and sensitive", name)
		}
	}
	if !attrs["base_url"].IsRequired() {
		t.Error("base_url must stay required")
	}
}

func configureWith(t *testing.T, baseURL, apiKey, idToken *string) provider.ConfigureResponse {
	t.Helper()
	sch := providerSchema(t).Schema
	str := func(s *string) tftypes.Value {
		if s == nil {
			return tftypes.NewValue(tftypes.String, nil)
		}
		return tftypes.NewValue(tftypes.String, *s)
	}
	raw := tftypes.NewValue(sch.Type().TerraformType(context.Background()), map[string]tftypes.Value{
		"base_url": str(baseURL),
		"api_key":  str(apiKey),
		"id_token": str(idToken),
	})
	resp := provider.ConfigureResponse{}
	New().Configure(context.Background(), provider.ConfigureRequest{Config: tfsdk.Config{Schema: sch, Raw: raw}}, &resp)
	return resp
}

func strp(s string) *string { return &s }

func TestConfigureCredentials(t *testing.T) {
	const url = "http://localhost:8080"

	t.Run("id token from config", func(t *testing.T) {
		t.Setenv(envAPIKey, "")
		t.Setenv(envIDToken, "")
		resp := configureWith(t, strp(url), nil, strp(validJWT))
		if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 0 || resp.ResourceData == nil {
			t.Fatalf("unexpected result: %v", resp.Diagnostics)
		}
	})

	t.Run("token from env", func(t *testing.T) {
		t.Setenv(envAPIKey, "")
		t.Setenv(envIDToken, validJWT)
		resp := configureWith(t, strp(url), nil, nil)
		if resp.Diagnostics.HasError() || resp.ResourceData == nil {
			t.Fatalf("unexpected result: %v", resp.Diagnostics)
		}
	})

	t.Run("config overrides env with warning", func(t *testing.T) {
		t.Setenv(envAPIKey, "env-key")
		t.Setenv(envIDToken, "")
		resp := configureWith(t, strp(url), nil, strp(validJWT))
		if resp.Diagnostics.HasError() || resp.Diagnostics.WarningsCount() != 1 {
			t.Fatalf("expected one warning and no error: %v", resp.Diagnostics)
		}
		if strings.Contains(resp.Diagnostics[0].Detail(), "env-key") {
			t.Fatal("warning leaks the key value")
		}
	})

	t.Run("both in config", func(t *testing.T) {
		resp := configureWith(t, strp(url), strp("k"), strp(validJWT))
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error")
		}
	})

	t.Run("no credential", func(t *testing.T) {
		t.Setenv(envAPIKey, "")
		t.Setenv(envIDToken, "")
		resp := configureWith(t, strp(url), nil, nil)
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error")
		}
		d := resp.Diagnostics[0].Detail()
		for _, want := range []string{"api_key", "id_token", envAPIKey, envIDToken} {
			if !strings.Contains(d, want) {
				t.Errorf("error should mention %s: %s", want, d)
			}
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		t.Setenv(envAPIKey, "")
		t.Setenv(envIDToken, "")
		resp := configureWith(t, strp(url), nil, strp("not-a-jwt"))
		if !resp.Diagnostics.HasError() {
			t.Fatal("expected an error")
		}
		if strings.Contains(resp.Diagnostics[0].Detail(), "not-a-jwt") {
			t.Fatal("error echoes the token")
		}
	})
}
