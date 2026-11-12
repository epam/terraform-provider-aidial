package provider

import (
	"fmt"
	"strings"
)

// Names of environment variables, not credential values.
const (
	envAPIKey  = "AIDIAL_API_KEY"  //nolint:gosec // G101: variable name, not a credential
	envIDToken = "AIDIAL_ID_TOKEN" //nolint:gosec // G101: variable name, not a credential
)

// credentialInputs holds every place a credential can come from. Empty (or
// whitespace-only) strings count as not set.
type credentialInputs struct {
	configAPIKey  string
	configIDToken string
	envAPIKey     string
	envIDToken    string
}

// resolvedCredential is the single credential chosen by resolveCredential.
type resolvedCredential struct {
	apiKey  string
	idToken string
}

// resolveCredential applies the provider-authentication precedence rule:
// configuration beats the environment; with nothing in configuration,
// AIDIAL_ID_TOKEN beats AIDIAL_API_KEY. Both attributes in configuration, or
// no credential at all, is an error. Every environment variable that is set
// but not used is reported in warnings (by name, never by value).
func resolveCredential(in credentialInputs) (resolvedCredential, []string, error) {
	in.configAPIKey = strings.TrimSpace(in.configAPIKey)
	in.configIDToken = strings.TrimSpace(in.configIDToken)
	in.envAPIKey = strings.TrimSpace(in.envAPIKey)
	in.envIDToken = strings.TrimSpace(in.envIDToken)

	if in.configAPIKey != "" && in.configIDToken != "" {
		return resolvedCredential{}, nil, fmt.Errorf("set either api_key or id_token in the provider configuration, not both")
	}

	var (
		res     resolvedCredential
		ignored []string
	)

	switch {
	case in.configAPIKey != "":
		res.apiKey = in.configAPIKey
		ignored = ignoredEnv(in.envAPIKey != "", in.envIDToken != "")
	case in.configIDToken != "":
		res.idToken = in.configIDToken
		ignored = ignoredEnv(in.envAPIKey != "", in.envIDToken != "")
	case in.envIDToken != "":
		res.idToken = in.envIDToken
		ignored = ignoredEnv(in.envAPIKey != "", false)
	case in.envAPIKey != "":
		res.apiKey = in.envAPIKey
	default:
		return resolvedCredential{}, nil, fmt.Errorf(
			"no credential: set api_key or id_token in the provider configuration, or the %s or %s environment variable",
			envAPIKey, envIDToken,
		)
	}

	var warnings []string
	if len(ignored) > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"Ignored environment variable(s) %s: a credential from the provider configuration, or a higher-precedence one, is used instead.",
			strings.Join(ignored, ", "),
		))
	}

	return res, warnings, nil
}

func ignoredEnv(apiKey, idToken bool) []string {
	var names []string
	if apiKey {
		names = append(names, envAPIKey)
	}

	if idToken {
		names = append(names, envIDToken)
	}

	return names
}

// checkJWTShape verifies only that token has three non-empty dot-separated
// segments. Signature, expiry and audience are checked by DIAL Core. The
// error never contains the token.
func checkJWTShape(token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("id_token is not a JWT: expected three dot-separated segments")
	}

	for _, p := range parts {
		if p == "" {
			return fmt.Errorf("id_token is not a JWT: a segment is empty")
		}
	}

	return nil
}
