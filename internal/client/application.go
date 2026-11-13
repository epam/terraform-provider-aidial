package client

import (
	"context"
	"encoding/json"
	"strings"
)

// Application mirrors the part of DIAL Core's Application the provider
// manages. Optional fields are pointers so that "not set" (nil, omitted from
// the request) stays distinct from an explicit empty value. Core replaces the
// whole document on PUT, so fields that are not modelled here (function, mcp,
// routes, external_services, author, created_at, updated_at, dependencies, ...)
// are dropped on save and ignored on read.
//
// Keys are snake_case, as DIAL Core 0.50.1 returns them (the OpenAPI document
// mixes camelCase and snake_case; Core accepts both on write). Nested features
// and interfaces reuse the model types.
type Application struct {
	Name                    string                    `json:"name,omitempty"`
	Endpoint                *string                   `json:"endpoint,omitempty"`
	Description             *string                   `json:"description,omitempty"`
	DisplayName             *string                   `json:"display_name,omitempty"`
	DisplayVersion          *string                   `json:"display_version,omitempty"`
	IconURL                 *string                   `json:"icon_url,omitempty"`
	EditorURL               *string                   `json:"editor_url,omitempty"`
	ViewerURL               *string                   `json:"viewer_url,omitempty"`
	ApplicationTypeSchemaID *string                   `json:"application_type_schema_id,omitempty"`
	CatalogSchemaID         *string                   `json:"catalog_schema_id,omitempty"`
	ForwardAuthToken        *bool                     `json:"forward_auth_token,omitempty"`
	MaxRetryAttempts        *int64                    `json:"max_retry_attempts,omitempty"`
	UserRoles               *[]string                 `json:"user_roles,omitempty"`
	Interceptors            *[]string                 `json:"interceptors,omitempty"`
	DefaultHeaders          map[string]string         `json:"default_headers,omitempty"`
	Features                *ModelFeatures            `json:"features,omitempty"`
	Interfaces              map[string]ModelInterface `json:"interfaces,omitempty"`
	// Defaults, ApplicationProperties and CatalogProperties are free-form
	// objects (MapStringObject in the API), passed through as-is.
	Defaults              json.RawMessage `json:"defaults,omitempty"`
	ApplicationProperties json.RawMessage `json:"application_properties,omitempty"`
	CatalogProperties     json.RawMessage `json:"catalog_properties,omitempty"`
}

// applicationURL builds /v1/applications/{bucket}/{path}. Like toolsets, Core
// reads an application path as folders separated by "/" and rejects an encoded
// slash (%2F), so each path segment is escaped on its own and the slashes
// between them are kept.
func (c *Client) applicationURL(bucket, path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = escapeSegment(segment)
	}
	return c.baseURL + "/v1/applications/" + escapeSegment(bucket) + "/" + strings.Join(segments, "/")
}

// GetApplication reads an API-managed application. A 404 returns (nil, nil).
func (c *Client) GetApplication(ctx context.Context, bucket, path string) (*Application, error) {
	return getResource[Application](ctx, c, c.applicationURL(bucket, path))
}

// SaveApplication creates or replaces an application.
func (c *Client) SaveApplication(ctx context.Context, bucket, path string, application *Application) error {
	return saveResource(ctx, c, c.applicationURL(bucket, path), application)
}

// DeleteApplication removes an application. A 404 is success.
func (c *Client) DeleteApplication(ctx context.Context, bucket, path string) error {
	return deleteResource(ctx, c, c.applicationURL(bucket, path))
}

func (c *Client) fileApplicationURL(name string) string {
	return c.baseURL + "/v1/admin/config/file/applications/" + escapeSegment(name)
}

// GetFileApplication reads a file-sourced application: one defined in DIAL
// Core's static configuration. The endpoint is read-only, so there is no Save
// or Delete. A 404 returns (nil, nil).
func (c *Client) GetFileApplication(ctx context.Context, name string) (*Application, error) {
	return getResource[Application](ctx, c, c.fileApplicationURL(name))
}
