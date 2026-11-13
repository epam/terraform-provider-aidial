package client

import (
	"context"
	"encoding/json"
	"strings"
)

// ToolSetAuthSettings mirrors the configuration part of DIAL Core's
// ResourceAuthSettings. The runtime fields (app_level_auth_status,
// global_auth_status, user_level_auth_status, code_verifier), the read-only
// client_secret_hint and code_challenge (Core derives it itself and
// overwrites any value sent) are deliberately not modelled.
type ToolSetAuthSettings struct {
	AuthenticationType      *string   `json:"authentication_type,omitempty"`
	APIKeyHeader            *string   `json:"api_key_header,omitempty"`
	AuthorizationEndpoint   *string   `json:"authorization_endpoint,omitempty"`
	TokenEndpoint           *string   `json:"token_endpoint,omitempty"`
	TokenEndpointAuthMethod *string   `json:"token_endpoint_auth_method,omitempty"`
	ClientID                *string   `json:"client_id,omitempty"`
	ClientSecret            *string   `json:"client_secret,omitempty"`
	RedirectURI             *string   `json:"redirect_uri,omitempty"`
	ScopesSupported         *[]string `json:"scopes_supported,omitempty"`
	CodeChallengeMethod     *string   `json:"code_challenge_method,omitempty"`
	Resource                *string   `json:"resource,omitempty"`
	DynamicallyRegistered   *bool     `json:"dynamically_registered,omitempty"`
}

// ToolSet mirrors DIAL Core's ToolSet. Optional fields are pointers so that
// "not set" (nil, omitted from the request) stays distinct from an explicit
// empty value ("" or []). created_at, updated_at and status are not modelled.
//
// Unlike models, DIAL Core 0.50.1 reads and writes toolsets with snake_case
// keys (it rejects the camelCase names the OpenAPI document lists), so every
// tag here is snake_case. Nested interfaces keep the keys of ModelInterface,
// which Core serializes the same way for toolsets and models.
type ToolSet struct {
	Name                 string                    `json:"name,omitempty"`
	DisplayName          *string                   `json:"display_name,omitempty"`
	Description          *string                   `json:"description,omitempty"`
	Author               *string                   `json:"author,omitempty"`
	Intro                *string                   `json:"intro,omitempty"`
	DisplayVersion       *string                   `json:"display_version,omitempty"`
	IconURL              *string                   `json:"icon_url,omitempty"`
	Reference            *string                   `json:"reference,omitempty"`
	Provider             *string                   `json:"provider,omitempty"`
	VendorWebsite        *string                   `json:"vendor_website,omitempty"`
	Endpoint             *string                   `json:"endpoint,omitempty"`
	ResponsesEndpoint    *string                   `json:"responses_endpoint,omitempty"`
	BaseURL              *string                   `json:"base_url,omitempty"`
	OverrideName         *string                   `json:"override_name,omitempty"`
	CatalogSchemaID      *string                   `json:"catalog_schema_id,omitempty"`
	Transport            *string                   `json:"transport,omitempty"`
	ForwardAuthToken     *bool                     `json:"forward_auth_token,omitempty"`
	ForwardPerRequestKey *bool                     `json:"forward_per_request_key,omitempty"`
	MaxInputAttachments  *int64                    `json:"max_input_attachments,omitempty"`
	MaxRetryAttempts     *int64                    `json:"max_retry_attempts,omitempty"`
	AllowedTools         *[]string                 `json:"allowed_tools,omitempty"`
	Dependencies         *[]string                 `json:"dependencies,omitempty"`
	DescriptionKeywords  *[]string                 `json:"description_keywords,omitempty"`
	InputAttachmentTypes *[]string                 `json:"input_attachment_types,omitempty"`
	Interceptors         *[]string                 `json:"interceptors,omitempty"`
	UserRoles            *[]string                 `json:"user_roles,omitempty"`
	DefaultHeaders       map[string]string         `json:"default_headers,omitempty"`
	Features             *ModelFeatures            `json:"features,omitempty"`
	Interfaces           map[string]ModelInterface `json:"interfaces,omitempty"`
	AuthSettings         *ToolSetAuthSettings      `json:"auth_settings,omitempty"`
	// Defaults, ResponsesDefaults and CatalogProperties are free-form objects
	// (MapStringObject in the API), passed through as-is.
	Defaults          json.RawMessage `json:"defaults,omitempty"`
	ResponsesDefaults json.RawMessage `json:"responses_defaults,omitempty"`
	CatalogProperties json.RawMessage `json:"catalog_properties,omitempty"`
}

// toolsetURL builds /v1/toolsets/{bucket}/{path}. Unlike the other resource
// types, Core reads a toolset path as folders separated by "/" and rejects an
// encoded slash (%2F), so each path segment is escaped on its own and the
// slashes between them are kept.
func (c *Client) toolsetURL(bucket, path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = escapeSegment(segment)
	}
	return c.baseURL + "/v1/toolsets/" + escapeSegment(bucket) + "/" + strings.Join(segments, "/")
}

// GetToolset reads an API-managed toolset. A 404 returns (nil, nil).
func (c *Client) GetToolset(ctx context.Context, bucket, path string) (*ToolSet, error) {
	return getResource[ToolSet](ctx, c, c.toolsetURL(bucket, path))
}

// SaveToolset creates or replaces a toolset.
func (c *Client) SaveToolset(ctx context.Context, bucket, path string, toolset *ToolSet) error {
	return saveResource(ctx, c, c.toolsetURL(bucket, path), toolset)
}

// DeleteToolset removes a toolset. A 404 is success.
func (c *Client) DeleteToolset(ctx context.Context, bucket, path string) error {
	return deleteResource(ctx, c, c.toolsetURL(bucket, path))
}

func (c *Client) fileToolsetURL(name string) string {
	return c.baseURL + "/v1/admin/config/file/toolsets/" + escapeSegment(name)
}

// GetFileToolset reads a file-sourced toolset: one defined in DIAL Core's
// static configuration. The endpoint is read-only, so there is no Save or
// Delete. A 404 returns (nil, nil).
func (c *Client) GetFileToolset(ctx context.Context, name string) (*ToolSet, error) {
	return getResource[ToolSet](ctx, c, c.fileToolsetURL(name))
}
