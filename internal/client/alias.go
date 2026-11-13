package client

import "context"

// AliasTarget is the part of an alias target DIAL Core accepts: only a path.
// Core rejects target.id and target.version on a stored alias.
type AliasTarget struct {
	Path string `json:"path"`
}

// Alias mirrors the part of DIAL Core's Alias the provider manages. Optional
// fields are pointers so that "not set" (nil, omitted from the request) stays
// distinct from an explicit empty value. Core replaces the whole document on
// PUT, so fields that are not modelled here (author, createdAt, updatedAt,
// status, ...) are dropped on save and ignored on read.
//
// Keys are camelCase, as in the OpenAPI document and as DIAL Core 0.50.1
// returns them. Core rejects endpoint, features, catalogProperties and
// catalogSchemaId on a stored alias, so they are not modelled.
type Alias struct {
	Name             string       `json:"name,omitempty"`
	PresentAs        string       `json:"presentAs,omitempty"`
	Target           *AliasTarget `json:"target,omitempty"`
	UserRoles        *[]string    `json:"userRoles,omitempty"`
	DisplayName      *string      `json:"displayName,omitempty"`
	Description      *string      `json:"description,omitempty"`
	DisplayVersion   *string      `json:"displayVersion,omitempty"`
	IconURL          *string      `json:"iconUrl,omitempty"`
	Intro            *string      `json:"intro,omitempty"`
	DescriptionWords *[]string    `json:"descriptionKeywords,omitempty"`
}

func (c *Client) aliasURL(bucket, path string) string {
	return c.baseURL + "/v1/aliases/" + escapeSegment(bucket) + "/" + escapeSegment(path)
}

// GetAlias reads an API-managed alias. A 404 returns (nil, nil).
func (c *Client) GetAlias(ctx context.Context, bucket, path string) (*Alias, error) {
	return getResource[Alias](ctx, c, c.aliasURL(bucket, path))
}

// SaveAlias creates or replaces an alias.
func (c *Client) SaveAlias(ctx context.Context, bucket, path string, alias *Alias) error {
	return saveResource(ctx, c, c.aliasURL(bucket, path), alias)
}

// DeleteAlias removes an alias. A 404 is success.
func (c *Client) DeleteAlias(ctx context.Context, bucket, path string) error {
	return deleteResource(ctx, c, c.aliasURL(bucket, path))
}

func (c *Client) fileAliasURL(name string) string {
	return c.baseURL + "/v1/admin/config/file/aliases/" + escapeSegment(name)
}

// GetFileAlias reads a file-sourced alias: one defined in DIAL Core's static
// configuration. The endpoint is read-only, so there is no Save or Delete. A
// 404 returns (nil, nil).
func (c *Client) GetFileAlias(ctx context.Context, name string) (*Alias, error) {
	return getResource[Alias](ctx, c, c.fileAliasURL(name))
}
