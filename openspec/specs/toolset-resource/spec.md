# toolset-resource Specification

## Purpose

Manage DIAL Core toolsets (`/v1/toolsets/{bucket}/{path}`) through the `aidial_toolset` resource and data source, so they can be created, imported and looked up from Terraform.

## Requirements

### Requirement: Toolset client methods

`internal/client/client.go` SHALL provide `GetToolset(ctx, bucket, path)`, `SaveToolset(ctx, bucket, path, toolset)` and `DeleteToolset(ctx, bucket, path)` against `/v1/toolsets/{bucket}/{path}`, following the existing client conventions. The body SHALL use the snake_case keys DIAL Core reads and writes for toolsets (Core rejects the camelCase names the OpenAPI document lists), and omit unset fields. The `{path}` keeps its `/` separators, because Core treats them as folders.

#### Scenario: Request conventions
- **WHEN** any toolset method calls the API
- **THEN** it sends the `Api-Key` header and does not use ETag, `If-Match` or `If-None-Match` headers

#### Scenario: API error
- **WHEN** a request returns a non-2xx status other than the GET/DELETE 404 cases
- **THEN** the error includes the status and response body

#### Scenario: PUT response
- **WHEN** `SaveToolset` receives a successful response
- **THEN** the body is treated as `{"name": ...}` only

#### Scenario: Snake_case body
- **WHEN** a toolset with `display_name`, `forward_auth_token` and `auth_settings` is saved
- **THEN** the request body uses the keys `display_name`, `forward_auth_token` and `auth_settings`

#### Scenario: Nested path
- **WHEN** a toolset is read with bucket `public` and path `team/sub/tool`
- **THEN** the request path is `/v1/toolsets/public/team/sub/tool`, with each segment percent-encoded on its own

#### Scenario: Toolset not found
- **WHEN** `GetToolset` receives `404`
- **THEN** it returns `nil, nil`

#### Scenario: Idempotent delete
- **WHEN** `DeleteToolset` receives `404`
- **THEN** it returns success

#### Scenario: Client secret not logged
- **WHEN** a request body containing `auth_settings.client_secret` is written to the debug log
- **THEN** the secret value is replaced with a redaction placeholder

### Requirement: aidial_toolset identity

The provider SHALL expose resource `aidial_toolset` with `id` (computed, `<bucket>/<path>`), `bucket` (required, non-empty string, replace on change) and `path` (required, non-empty, replace on change). `bucket` SHALL NOT be restricted to `platform`; DIAL Core decides which buckets it accepts. See `resource-identity`.

#### Scenario: Resource registration
- **WHEN** provider resources are registered
- **THEN** `aidial_toolset` is included

#### Scenario: Non-platform bucket
- **WHEN** `bucket` is `public`
- **THEN** validation passes and requests go to `/v1/toolsets/public/<path>`

#### Scenario: Empty bucket or path
- **WHEN** `bucket` or `path` is an empty string, or either is omitted
- **THEN** validation fails at plan time

#### Scenario: Immutable identity
- **WHEN** `bucket` or `path` changes
- **THEN** Terraform plans replacement

### Requirement: Toolset name handling

`name` SHALL be optional and computed: sent to Core only when configured, otherwise derived by Core and read back. `api_name` SHALL be computed and hold the canonical name Core returns. `created_at`, `updated_at` and Core's `status` SHALL NOT be exposed.

#### Scenario: Name omitted
- **WHEN** `name` is not configured
- **THEN** no name is sent, and `name` in state equals `api_name`

### Requirement: aidial_toolset scalar attributes

The resource SHALL expose optional string attributes `description`, `display_name`, `intro`, `display_version`, `icon_url`, `provider_name`, `vendor_website`, `endpoint`, `responses_endpoint`, `base_url`, `override_name` and `catalog_schema_id`, in snake_case after the DIAL Core field. `provider_name` is Core's `provider`, which Terraform reserves as an attribute name. `display_name` and `intro` are plain strings; the locale-map form is unsupported.

#### Scenario: Minimal toolset
- **WHEN** a config sets only `bucket`, `path` and `endpoint`
- **THEN** the toolset is created with just that body, and every other attribute reads back as computed or null

### Requirement: aidial_toolset transport and numeric attributes

The resource SHALL expose optional `transport` (`HTTP` or `SSE`), bools `forward_auth_token` and `forward_per_request_key`, and ints `max_input_attachments` and `max_retry_attempts`.

#### Scenario: Invalid transport
- **WHEN** `transport` is neither `HTTP` nor `SSE`
- **THEN** validation fails at plan time

### Requirement: Attributes DIAL Core fills in

DIAL Core fills in values the request leaves out, so `reference` (a generated identifier), `transport` (`HTTP`), `forward_auth_token` and `forward_per_request_key` (false), `max_retry_attempts` (1), `allowed_tools`, `dependencies`, `description_keywords` and `interceptors` (empty sets) and `auth_settings` (type `NONE`) SHALL be optional and computed, and stay stable across updates. Computed values SHALL NOT be sent until they are known.

#### Scenario: Minimal config after apply
- **WHEN** a config sets only `bucket`, `path` and `endpoint`
- **THEN** after apply the defaulted attributes hold the values Core returned, the request carried only `endpoint`, and a re-plan shows no change

#### Scenario: Generated reference is stable
- **WHEN** an attribute is updated in place
- **THEN** the update sends the `reference` kept in state, so Core does not generate a new one

### Requirement: Toolset author is server-set

`author` SHALL be computed only, because DIAL Core sets it from the caller and ignores a sent value.

#### Scenario: Author is server-set
- **WHEN** a toolset is created
- **THEN** `author` is unknown in the plan, never sent, and read back from Core

### Requirement: aidial_toolset collection attributes

The resource SHALL also expose: optional sets of strings `input_attachment_types` and `user_roles` (the other string sets are in the previous requirement); optional, computed map `default_headers`; `features` and `interfaces` with the same shape and validation as on `aidial_model`; free-form objects `defaults`, `responses_defaults` and `catalog_properties` as Terraform dynamic attributes; and nested `auth_settings`.

#### Scenario: Features and interfaces reuse
- **WHEN** `features` or `interfaces` is configured on a toolset and on a model with the same values
- **THEN** both are validated, sent and read back identically

#### Scenario: Free-form values keep their form
- **WHEN** Core returns JSON equal to the configured `defaults`, `responses_defaults` or `catalog_properties`
- **THEN** the configured value and its types are kept in state

### Requirement: Toolset authentication settings

`auth_settings` SHALL expose `authentication_type` (`OAUTH`, `API_KEY`, `NONE` or `DIAL_NATIVE`), `api_key_header`, `authorization_endpoint`, `token_endpoint`, `token_endpoint_auth_method`, `client_id`, `client_secret`, `redirect_uri`, `scopes_supported` (set of strings), `code_challenge_method`, `resource` and `dynamically_registered` (bool).

#### Scenario: Invalid authentication type
- **WHEN** `authentication_type` is not one of the allowed values
- **THEN** validation fails at plan time

#### Scenario: Runtime status ignored
- **WHEN** Core returns auth status fields, `code_verifier`, `code_challenge` or `client_secret_hint`
- **THEN** none of them appears in state or causes a plan difference, because the schema has no such attributes

### Requirement: Toolset client secret handling

`auth_settings.client_secret` SHALL be sensitive. Core never returns it, so state SHALL keep the configured value across reads, and it SHALL be null after import.

#### Scenario: Secret kept across refresh
- **WHEN** Core's GET omits `client_secret`
- **THEN** the value in state stays as configured and the next plan shows no change

#### Scenario: Secret after import
- **WHEN** a toolset is imported
- **THEN** `client_secret` is null

### Requirement: Toolset create, read, update, delete

Create and update SHALL PUT the toolset and then re-read it with GET to obtain state. A GET `404` during refresh SHALL remove the resource from state. Delete SHALL treat `404` as success. Update SHALL be in place for any attribute other than `bucket` and `path`.

#### Scenario: Create reads back
- **WHEN** a toolset is created
- **THEN** state reflects the GET result, with `api_name` populated

#### Scenario: Drift removal
- **WHEN** the toolset no longer exists in Core during refresh
- **THEN** it is removed from state

#### Scenario: In-place update
- **WHEN** `endpoint` or `description` changes
- **THEN** the toolset is updated in place with PUT

#### Scenario: Delete already gone
- **WHEN** delete receives `404`
- **THEN** it succeeds

### Requirement: Toolset state consistency

Where DIAL Core leaves an optional attribute out, it SHALL be an explicit null, so a config that leaves it unset plans no change; a configured empty collection SHALL stay empty. `interfaces` and `default_headers` are empty maps when Core has none, as on `aidial_model`. For the attributes Core fills in, an unset config reads back as Core's value. After apply, state SHALL equal the planned values for every configured attribute.

#### Scenario: Omitted optional collections
- **WHEN** Core omits `user_roles`, `input_attachment_types`, `features` and `catalog_properties`
- **THEN** they are null in state, `interfaces` and `default_headers` are empty maps, and a re-plan shows no change

#### Scenario: Configured empty collection
- **WHEN** `user_roles = []` is configured
- **THEN** state holds an empty set and a re-plan shows no change

#### Scenario: Core-defaulted collection
- **WHEN** `interceptors` is not configured
- **THEN** state holds the empty set Core returns and a re-plan shows no change

### Requirement: Toolset import

The resource SHALL support import by `<bucket>/<path>`, taking both attributes from the import id, and then read the toolset.

#### Scenario: Import
- **WHEN** `terraform import aidial_toolset.x public/team/tool` runs
- **THEN** `bucket` is `public`, `path` is `team/tool` and the attributes are read from Core

#### Scenario: Malformed import id
- **WHEN** the import id has no `/` or an empty bucket or path
- **THEN** import fails with a diagnostic naming the expected format

### Requirement: aidial_toolset data source

The provider SHALL expose data source `aidial_toolset` with `bucket` (required, non-empty) and `path` (required, non-empty), computed `id`, `api_name` and every toolset attribute the resource exposes, with the same null-vs-empty rules. `auth_settings` on the data source SHALL NOT include `client_secret`. A toolset that Core does not have SHALL produce an error diagnostic naming the bucket and path. File-sourced toolsets are the job of `aidial_file_toolset`.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_toolset` is included

#### Scenario: Existing toolset
- **WHEN** the toolset exists
- **THEN** `id`, `api_name` and the attributes are populated, equal to what the resource would hold

#### Scenario: Missing toolset
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: API error
- **WHEN** the API returns a non-2xx status other than 404
- **THEN** the data source fails with a diagnostic carrying the status and body
