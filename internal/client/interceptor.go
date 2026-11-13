package client

import (
	"context"
	"encoding/json"
)

// Interceptor mirrors the part of DIAL Core's Interceptor the provider manages.
// Optional fields are pointers so that "not set" (nil, omitted from the
// request) stays distinct from an explicit empty value. Core replaces the whole
// document on PUT, so fields that are not modelled here (author, createdAt,
// updatedAt, reference, dependencies, descriptionKeywords, ...) are dropped on
// save and ignored on read.
//
// Keys are camelCase, as in the OpenAPI document and as DIAL Core 0.50.1 returns
// them. Nested features and interfaces reuse the model types.
type Interceptor struct {
	Name             string                    `json:"name,omitempty"`
	Endpoint         *string                   `json:"endpoint,omitempty"`
	Description      *string                   `json:"description,omitempty"`
	DisplayName      *string                   `json:"displayName,omitempty"`
	DisplayVersion   *string                   `json:"displayVersion,omitempty"`
	IconURL          *string                   `json:"iconUrl,omitempty"`
	ForwardAuthToken *bool                     `json:"forwardAuthToken,omitempty"`
	MaxRetryAttempts *int64                    `json:"maxRetryAttempts,omitempty"`
	UserRoles        *[]string                 `json:"userRoles,omitempty"`
	DefaultHeaders   map[string]string         `json:"defaultHeaders,omitempty"`
	CatalogSchemaID  *string                   `json:"catalogSchemaId,omitempty"`
	Features         *ModelFeatures            `json:"features,omitempty"`
	Interfaces       map[string]ModelInterface `json:"interfaces,omitempty"`
	// Defaults and CatalogProperties are free-form objects (MapStringObject in
	// the API), passed through as-is.
	Defaults          json.RawMessage `json:"defaults,omitempty"`
	CatalogProperties json.RawMessage `json:"catalogProperties,omitempty"`
}

func (c *Client) interceptorURL(bucket, path string) string {
	return c.baseURL + "/v1/interceptors/" + escapeSegment(bucket) + "/" + escapeSegment(path)
}

// GetInterceptor reads an API-managed interceptor. A 404 returns (nil, nil).
func (c *Client) GetInterceptor(ctx context.Context, bucket, path string) (*Interceptor, error) {
	return getResource[Interceptor](ctx, c, c.interceptorURL(bucket, path))
}

// SaveInterceptor creates or replaces an interceptor.
func (c *Client) SaveInterceptor(ctx context.Context, bucket, path string, interceptor *Interceptor) error {
	return saveResource(ctx, c, c.interceptorURL(bucket, path), interceptor)
}

// DeleteInterceptor removes an interceptor. A 404 is success.
func (c *Client) DeleteInterceptor(ctx context.Context, bucket, path string) error {
	return deleteResource(ctx, c, c.interceptorURL(bucket, path))
}

func (c *Client) fileInterceptorURL(name string) string {
	return c.baseURL + "/v1/admin/config/file/interceptors/" + escapeSegment(name)
}

// GetFileInterceptor reads a file-sourced interceptor: one defined in DIAL
// Core's static configuration. The endpoint is read-only, so there is no Save
// or Delete. A 404 returns (nil, nil).
func (c *Client) GetFileInterceptor(ctx context.Context, name string) (*Interceptor, error) {
	return getResource[Interceptor](ctx, c, c.fileInterceptorURL(name))
}
