package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     string
}

// FlexibleFloat64 decodes a JSON number that the DIAL API may encode either
// as a bare number or, for values exceeding safe float precision (e.g. the
// int64-max "unlimited" sentinel), as a quoted numeric string.
type FlexibleFloat64 float64

func (f *FlexibleFloat64) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	switch v := raw.(type) {
	case float64:
		*f = FlexibleFloat64(v)
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("invalid numeric string %q: %w", v, err)
		}
		*f = FlexibleFloat64(parsed)
	default:
		return fmt.Errorf("cannot unmarshal %T into FlexibleFloat64", raw)
	}

	return nil
}

// FlexibleString decodes a JSON value that DIAL Core may encode either as a
// string or as a bare number (e.g. the epoch-millisecond createdAt/updatedAt
// timestamps) into its string form.
type FlexibleString string

func (f *FlexibleString) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	switch v := raw.(type) {
	case nil:
		*f = ""
	case string:
		*f = FlexibleString(v)
	case float64:
		*f = FlexibleString(strconv.FormatFloat(v, 'f', -1, 64))
	default:
		return fmt.Errorf("cannot unmarshal %T into FlexibleString", raw)
	}

	return nil
}

// RoleMetadata is the "_metadata" object DIAL Core attaches to a role on
// read. It carries the status and timestamps that older responses placed at
// the top level of the role.
type RoleMetadata struct {
	Status    string         `json:"status,omitempty"`
	CreatedAt FlexibleString `json:"createdAt,omitempty"`
	UpdatedAt FlexibleString `json:"updatedAt,omitempty"`
}

type CostLimit struct {
	Day    *FlexibleFloat64 `json:"day,omitempty"`
	Minute *FlexibleFloat64 `json:"minute,omitempty"`
	Month  *FlexibleFloat64 `json:"month,omitempty"`
	Week   *FlexibleFloat64 `json:"week,omitempty"`
}

type ShareRule struct {
	InvitationTTL    *int64 `json:"invitation_ttl,omitempty"`
	MaxAcceptedUsers *int64 `json:"max_accepted_users,omitempty"`
}

type Role struct {
	Name        string                     `json:"name"`
	Description *string                    `json:"description,omitempty"`
	DisplayName *string                    `json:"displayName,omitempty"`
	Limits      map[string]json.RawMessage `json:"limits,omitempty"`
	CostLimit   *CostLimit                 `json:"costLimit,omitempty"`
	Share       map[string]ShareRule       `json:"share,omitempty"`
	Topics      []string                   `json:"topics,omitempty"`
	GrantedKeys []string                   `json:"grantedKeys,omitempty"`
	CreatedAt   string                     `json:"createdAt,omitempty"`
	UpdatedAt   string                     `json:"updatedAt,omitempty"`
	Status      string                     `json:"status,omitempty"`
	Metadata    *RoleMetadata              `json:"_metadata,omitempty"`
}

type Bucket struct {
	Bucket  string `json:"bucket"`
	AppData string `json:"appdata"`
}

func NewClient(baseURL, apiKey string) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")

	if baseURL == "" {
		return nil, fmt.Errorf("base_url is required")
	}

	if apiKey == "" {
		return nil, fmt.Errorf("api_key is required")
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base_url: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("base_url must use http or https")
	}

	if u.Host == "" {
		return nil, fmt.Errorf("base_url must include a host")
	}

	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
		apiKey:     apiKey,
	}, nil
}

func (c *Client) roleURL(bucket, path string) string {
	escBucket := strings.ReplaceAll(bucket, "/", "%2F")
	escPath := strings.ReplaceAll(path, "/", "%2F")
	return c.baseURL + "/v1/roles/" + escBucket + "/" + escPath
}

func (c *Client) newRequest(
	ctx context.Context,
	method, endpoint string,
	body io.Reader,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return req, nil
}

func apiError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)

	return fmt.Errorf(
		"API error %d: %s",
		resp.StatusCode,
		strings.TrimSpace(string(body)),
	)
}

const redactedPlaceholder = "***REDACTED***"

// sensitiveJSONFields lists JSON object keys whose values carry raw
// credential material (a key's own secret value, an upstream's auth
// key/secretExtraData) and must never be written to logs.
var sensitiveJSONFields = map[string]bool{
	"key":             true,
	"secretExtraData": true,
}

// redactSensitiveJSON returns a copy of a JSON payload with any
// sensitiveJSONFields values masked, for safe use in log output. It is
// best-effort: if the payload does not parse as JSON, it is returned
// unchanged rather than dropped, so logging never fails because of it.
func redactSensitiveJSON(payload []byte) []byte {
	var data any
	if err := json.Unmarshal(payload, &data); err != nil {
		return payload
	}

	redactJSONValue(data)

	redacted, err := json.Marshal(data)
	if err != nil {
		return payload
	}

	return redacted
}

func redactJSONValue(v any) {
	switch val := v.(type) {
	case map[string]any:
		for key, child := range val {
			if sensitiveJSONFields[key] {
				if child != nil {
					val[key] = redactedPlaceholder
				}
				continue
			}
			redactJSONValue(child)
		}
	case []any:
		for _, child := range val {
			redactJSONValue(child)
		}
	}
}

// getResource performs a GET against url and decodes a 2xx response body
// into T. A 404 is not an error: it returns (nil, nil), matching DIAL
// Core's "absent" semantics for roles, keys, models, and routes.
func getResource[T any](ctx context.Context, c *Client, url string) (*T, error) {
	req, err := c.newRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, apiError(resp)
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

// saveResource PUTs the JSON encoding of body to url, logging the request
// and response (with sensitive fields redacted) along the way. DIAL Core's
// PUT response only contains {"name": ...}, so callers re-read the
// resource via getResource to obtain its authoritative state.
func saveResource(ctx context.Context, c *Client, url string, body any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := c.newRequest(ctx, http.MethodPut, url, bytes.NewReader(payload))
	if err != nil {
		return err
	}

	log.Printf(
		"DIAL request method=%s url=%s api_key_present=%t content_type=%q body=%s",
		req.Method,
		req.URL.String(),
		req.Header.Get("Api-Key") != "",
		req.Header.Get("Content-Type"),
		string(redactSensitiveJSON(payload)),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Printf(
		"DIAL response method=%s url=%s status=%d body=%s",
		req.Method,
		req.URL.String(),
		resp.StatusCode,
		string(redactSensitiveJSON(responseBody)),
	)

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"API error %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(responseBody)),
		)
	}

	return nil
}

// deleteResource issues a DELETE against url. A 404 is treated as success,
// matching DIAL Core's idempotent-delete semantics.
func deleteResource(ctx context.Context, c *Client, url string) error {
	req, err := c.newRequest(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return apiError(resp)
	}

	return nil
}

func (c *Client) GetBucket(ctx context.Context, bucket string) (*Bucket, error) {
	req, err := c.newRequest(ctx, http.MethodGet, c.baseURL+"/v1/bucket", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, apiError(resp)
	}

	var result Bucket
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GetRole(ctx context.Context, bucket, path string) (*Role, error) {
	return getResource[Role](ctx, c, c.roleURL(bucket, path))
}

func (c *Client) SaveRole(ctx context.Context, bucket, path string, role *Role) error {
	return saveResource(ctx, c, c.roleURL(bucket, path), role)
}

func (c *Client) DeleteRole(ctx context.Context, bucket, path string) error {
	return deleteResource(ctx, c, c.roleURL(bucket, path))
}

type AllowedIPAddressRanges struct {
	Ranges []string `json:"ranges,omitempty"`
}

type Key struct {
	AllowedIPAddressRanges *AllowedIPAddressRanges `json:"allowedIpAddressRanges,omitempty"`
	Key                    string                  `json:"key,omitempty"`
	Project                string                  `json:"project,omitempty"`
	Role                   string                  `json:"role,omitempty"`
	Roles                  []string                `json:"roles,omitempty"`
	Secured                *bool                   `json:"secured,omitempty"`
	Name                   string                  `json:"name,omitempty"`
	Status                 string                  `json:"status,omitempty"`
}

func (c *Client) keyURL(bucket, path string) string {
	escBucket := strings.ReplaceAll(bucket, "/", "%2F")
	escPath := strings.ReplaceAll(path, "/", "%2F")
	return c.baseURL + "/v1/keys/" + escBucket + "/" + escPath
}

func (c *Client) GetKey(ctx context.Context, bucket, path string) (*Key, error) {
	return getResource[Key](ctx, c, c.keyURL(bucket, path))
}

func (c *Client) SaveKey(ctx context.Context, bucket, path string, key *Key) error {
	return saveResource(ctx, c, c.keyURL(bucket, path), key)
}

func (c *Client) DeleteKey(ctx context.Context, bucket, path string) error {
	return deleteResource(ctx, c, c.keyURL(bucket, path))
}

// Model types

type ModelFeatures struct {
	AccessibleByPerRequestKey              *bool    `json:"accessible_by_per_request_key,omitempty"`
	AllowResume                            *bool    `json:"allow_resume,omitempty"`
	AssistantAttachmentsInRequestSupported *bool    `json:"assistant_attachments_in_request_supported,omitempty"`
	AutoCachingSupported                   *bool    `json:"auto_caching_supported,omitempty"`
	CacheSupported                         *bool    `json:"cache_supported,omitempty"`
	ConfigurationEndpoint                  *string  `json:"configuration_endpoint,omitempty"`
	ConsentRequired                        *bool    `json:"consent_required,omitempty"`
	ContentPartsSupported                  *bool    `json:"content_parts_supported,omitempty"`
	CustomTemperatureSupported             *bool    `json:"custom_temperature_supported,omitempty"`
	FolderAttachmentsSupported             *bool    `json:"folder_attachments_supported,omitempty"`
	MaxCompletionTokensSupported           *bool    `json:"max_completion_tokens_supported,omitempty"`
	MaxTokensSupported                     *bool    `json:"max_tokens_supported,omitempty"`
	ParallelToolCallsSupported             *bool    `json:"parallel_tool_calls_supported,omitempty"`
	RateEndpoint                           *string  `json:"rate_endpoint,omitempty"`
	ReasoningEfforts                       []string `json:"reasoning_efforts,omitempty"`
	SeedSupported                          *bool    `json:"seed_supported,omitempty"`
	SupportCommentInRateResponse           *bool    `json:"support_comment_in_rate_response,omitempty"`
	SystemPromptSupported                  *bool    `json:"system_prompt_supported,omitempty"`
	TemperatureSupported                   *bool    `json:"temperature_supported,omitempty"`
	TokenizeEndpoint                       *string  `json:"tokenize_endpoint,omitempty"`
	ToolsSupported                         *bool    `json:"tools_supported,omitempty"`
	TruncatePromptEndpoint                 *string  `json:"truncate_prompt_endpoint,omitempty"`
	URLAttachmentsSupported                *bool    `json:"url_attachments_supported,omitempty"`
	SkillsSupported                        *bool    `json:"skills_supported,omitempty"`
}

type ModelLimits struct {
	MaxPromptTokens     *int64 `json:"maxPromptTokens,omitempty"`
	MaxCompletionTokens *int64 `json:"maxCompletionTokens,omitempty"`
	MaxTotalTokens      *int64 `json:"maxTotalTokens,omitempty"`
}

type ModelPricing struct {
	Unit       string  `json:"unit"`
	Prompt     *string `json:"prompt,omitempty"`
	Completion *string `json:"completion,omitempty"`
	CacheRead  *string `json:"cacheRead,omitempty"`
	CacheWrite *string `json:"cacheWrite,omitempty"`
}

type ModelInterface struct {
	BaseURL *string `json:"base_url,omitempty"`
}

// FlexibleStringMap is sent as a JSON object but decodes from either an
// object or a string holding a JSON object: DIAL Core 0.48.1 returns
// upstream extraData as an object, newer versions as a string (the OpenAPI
// spec types it as string). Non-string values are kept as their JSON text.
type FlexibleStringMap map[string]string

func (m *FlexibleStringMap) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		*m = nil
		return nil
	}
	if len(trimmed) > 0 && trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			*m = nil
			return nil
		}
		if err := m.decodeObject([]byte(text)); err != nil {
			return fmt.Errorf("expected a JSON object inside the string, got %q: %w", text, err)
		}
		return nil
	}
	return m.decodeObject(trimmed)
}

func (m *FlexibleStringMap) decodeObject(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var raw map[string]any
	if err := decoder.Decode(&raw); err != nil {
		return err
	}
	result := make(FlexibleStringMap, len(raw))
	for key, value := range raw {
		if text, ok := value.(string); ok {
			result[key] = text
			continue
		}
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		result[key] = string(encoded)
	}
	*m = result
	return nil
}

type ModelUpstream struct {
	Endpoint          *string           `json:"endpoint,omitempty"`
	ResponsesEndpoint *string           `json:"responsesEndpoint,omitempty"`
	ID                *string           `json:"id,omitempty"`
	ExtraData         FlexibleStringMap `json:"extraData,omitempty"`
	Key               *string           `json:"key,omitempty"`
	SecretExtraData   FlexibleStringMap `json:"secretExtraData,omitempty"`
	Tier              *int              `json:"tier,omitempty"`
	Weight            *int              `json:"weight,omitempty"`
}

// Model mirrors DIAL Core's Model. Optional fields are pointers so that
// "not set" (nil, omitted from the request) stays distinct from an explicit
// empty value ("" or []), which Core treats differently.
type Model struct {
	Name                 string                    `json:"name"`
	Type                 string                    `json:"type"`
	DisplayName          *string                   `json:"displayName,omitempty"`
	Description          *string                   `json:"description,omitempty"`
	Endpoint             *string                   `json:"endpoint,omitempty"`
	UserRoles            *[]string                 `json:"userRoles,omitempty"`
	ResponsesEndpoint    *string                   `json:"responsesEndpoint,omitempty"`
	DisplayVersion       *string                   `json:"displayVersion,omitempty"`
	TokenizerModel       *string                   `json:"tokenizerModel,omitempty"`
	IconURL              *string                   `json:"iconUrl,omitempty"`
	Intro                *string                   `json:"intro,omitempty"`
	Reference            *string                   `json:"reference,omitempty"`
	ForwardAuthToken     *bool                     `json:"forwardAuthToken,omitempty"`
	MaxRetryAttempts     *int64                    `json:"maxRetryAttempts,omitempty"`
	MaxInputAttachments  *int64                    `json:"maxInputAttachments,omitempty"`
	Author               *string                   `json:"author,omitempty"`
	Status               string                    `json:"status,omitempty"`
	Features             *ModelFeatures            `json:"features,omitempty"`
	Limits               *ModelLimits              `json:"limits,omitempty"`
	Pricing              *ModelPricing             `json:"pricing,omitempty"`
	Upstreams            []ModelUpstream           `json:"upstreams,omitempty"`
	Dependencies         *[]string                 `json:"dependencies,omitempty"`
	DescriptionKeywords  *[]string                 `json:"descriptionKeywords,omitempty"`
	InputAttachmentTypes *[]string                 `json:"inputAttachmentTypes,omitempty"`
	Interceptors         *[]string                 `json:"interceptors,omitempty"`
	FieldsHashingOrder   *[]string                 `json:"fieldsHashingOrder,omitempty"`
	OverrideName         *string                   `json:"overrideName,omitempty"`
	Interfaces           map[string]ModelInterface `json:"interfaces,omitempty"`
	DefaultHeaders       map[string]string         `json:"defaultHeaders,omitempty"`
	// Defaults is a free-form object (MapStringObject in the API), passed through as-is.
	Defaults json.RawMessage `json:"defaults,omitempty"`
	// CatalogSchemaID references a catalog schema (see CatalogSchema) by its $id.
	CatalogSchemaID *string `json:"catalogSchemaId,omitempty"`
	// CatalogProperties is a free-form object (MapStringObject in the API), passed through as-is.
	CatalogProperties json.RawMessage `json:"catalogProperties,omitempty"`
}

func (c *Client) modelURL(bucket, path string) string {
	escBucket := strings.ReplaceAll(bucket, "/", "%2F")
	escPath := strings.ReplaceAll(path, "/", "%2F")
	return c.baseURL + "/v1/models/" + escBucket + "/" + escPath
}

func (c *Client) GetModel(ctx context.Context, bucket, path string) (*Model, error) {
	return getResource[Model](ctx, c, c.modelURL(bucket, path))
}

func (c *Client) SaveModel(ctx context.Context, bucket, path string, model *Model) error {
	return saveResource(ctx, c, c.modelURL(bucket, path), model)
}

func (c *Client) DeleteModel(ctx context.Context, bucket, path string) error {
	return deleteResource(ctx, c, c.modelURL(bucket, path))
}

func (c *Client) fileModelURL(name string) string {
	return c.baseURL + "/v1/admin/config/file/models/" + name
}

// GetFileModel reads a file-sourced model: one defined in DIAL Core's
// static configuration rather than created via PUT /v1/models/{bucket}/{path}.
// File-sourced models have no bucket/path and are read-only (GET only, no
// PUT/DELETE), so unlike GetModel there is no corresponding Save/Delete.
func (c *Client) GetFileModel(ctx context.Context, name string) (*Model, error) {
	return getResource[Model](ctx, c, c.fileModelURL(name))
}

// Route types

type RouteResponse struct {
	Body   *string `json:"body,omitempty"`
	Status *int64  `json:"status,omitempty"`
}

type RouteAttachmentPath struct {
	RequestBody  []string `json:"requestBody,omitempty"`
	ResponseBody []string `json:"responseBody,omitempty"`
}

type Route struct {
	Name             string               `json:"name"`
	Order            *int64               `json:"order,omitempty"`
	Methods          []string             `json:"methods,omitempty"`
	Paths            []string             `json:"paths,omitempty"`
	Permissions      []string             `json:"permissions,omitempty"`
	UserRoles        []string             `json:"userRoles,omitempty"`
	MaxRetryAttempts *int64               `json:"maxRetryAttempts,omitempty"`
	RewritePath      *bool                `json:"rewritePath,omitempty"`
	Response         *RouteResponse       `json:"response,omitempty"`
	AttachmentPaths  *RouteAttachmentPath `json:"attachmentPaths,omitempty"`
	Upstreams        []ModelUpstream      `json:"upstreams,omitempty"`
	Status           string               `json:"status,omitempty"`
}

func (c *Client) routeURL(bucket, path string) string {
	escBucket := strings.ReplaceAll(bucket, "/", "%2F")
	escPath := strings.ReplaceAll(path, "/", "%2F")
	return c.baseURL + "/v1/routes/" + escBucket + "/" + escPath
}

func (c *Client) GetRoute(ctx context.Context, bucket, path string) (*Route, error) {
	return getResource[Route](ctx, c, c.routeURL(bucket, path))
}

func (c *Client) SaveRoute(ctx context.Context, bucket, path string, route *Route) error {
	return saveResource(ctx, c, c.routeURL(bucket, path), route)
}

func (c *Client) DeleteRoute(ctx context.Context, bucket, path string) error {
	return deleteResource(ctx, c, c.routeURL(bucket, path))
}

// CatalogSchema is a catalog schema document: a JSON Schema with a required
// $id plus arbitrary extra keys, so it is kept as an opaque JSON object.
type CatalogSchema = json.RawMessage

func (c *Client) catalogSchemaURL(bucket, path string) string {
	escBucket := strings.ReplaceAll(bucket, "/", "%2F")
	escPath := strings.ReplaceAll(path, "/", "%2F")
	return c.baseURL + "/v1/catalog_schemas/" + escBucket + "/" + escPath
}

// GetCatalogSchema returns the schema document without the name and status
// keys DIAL Core merges into the GET response (EntityMetadata), or nil when
// the schema does not exist.
func (c *Client) GetCatalogSchema(ctx context.Context, bucket, path string) (CatalogSchema, error) {
	doc, err := getResource[map[string]json.RawMessage](ctx, c, c.catalogSchemaURL(bucket, path))
	if err != nil || doc == nil {
		return nil, err
	}
	delete(*doc, "name")
	delete(*doc, "status")
	return json.Marshal(doc)
}

func (c *Client) SaveCatalogSchema(ctx context.Context, bucket, path string, schema CatalogSchema) error {
	return saveResource(ctx, c, c.catalogSchemaURL(bucket, path), schema)
}

func (c *Client) DeleteCatalogSchema(ctx context.Context, bucket, path string) error {
	return deleteResource(ctx, c, c.catalogSchemaURL(bucket, path))
}
