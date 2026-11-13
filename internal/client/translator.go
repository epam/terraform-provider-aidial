package client

import "context"

// Translator mirrors DIAL Core's Translator: a service that converts requests
// from one interface type to another. All three fields are required by Core.
// Core also returns name and status on GET, which are ignored here.
//
// Keys are camelCase, as in the OpenAPI document and as DIAL Core 0.50.1 returns
// them.
type Translator struct {
	BaseURL string `json:"baseUrl"`
	In      string `json:"in"`
	Out     string `json:"out"`
}

func (c *Client) translatorURL(bucket, path string) string {
	return c.baseURL + "/v1/translators/" + escapeSegment(bucket) + "/" + escapeSegment(path)
}

// GetTranslator reads an API-managed translator. A 404 returns (nil, nil).
func (c *Client) GetTranslator(ctx context.Context, bucket, path string) (*Translator, error) {
	return getResource[Translator](ctx, c, c.translatorURL(bucket, path))
}

// SaveTranslator creates or replaces a translator.
func (c *Client) SaveTranslator(ctx context.Context, bucket, path string, translator *Translator) error {
	return saveResource(ctx, c, c.translatorURL(bucket, path), translator)
}

// DeleteTranslator removes a translator. A 404 is success.
func (c *Client) DeleteTranslator(ctx context.Context, bucket, path string) error {
	return deleteResource(ctx, c, c.translatorURL(bucket, path))
}

func (c *Client) fileTranslatorURL(name string) string {
	return c.baseURL + "/v1/admin/config/file/translators/" + escapeSegment(name)
}

// GetFileTranslator reads a file-sourced translator: one defined in DIAL Core's
// static configuration. The endpoint is read-only, so there is no Save or
// Delete. A 404 returns (nil, nil).
func (c *Client) GetFileTranslator(ctx context.Context, name string) (*Translator, error) {
	return getResource[Translator](ctx, c, c.fileTranslatorURL(name))
}
