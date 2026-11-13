# interceptor-resource Specification

## Purpose

Manage DIAL Core interceptors (`/v1/interceptors/{bucket}/{path}`) through the `aidial_interceptor` resource and data source, so they can be created, imported and looked up from Terraform.

## ADDED Requirements

### Requirement: Interceptor client methods

`internal/client/interceptor.go` SHALL provide `GetInterceptor(ctx, bucket, path)`, `SaveInterceptor(ctx, bucket, path, interceptor)` and `DeleteInterceptor(ctx, bucket, path)` against `/v1/interceptors/{bucket}/{path}`, following the existing client conventions (`getResource`, `saveResource`, `deleteResource`). The body SHALL use the camelCase JSON keys of the OpenAPI `Interceptor` schema (`displayName`, `forwardAuthToken`, `maxRetryAttempts`, `userRoles`, `catalogSchemaId`, `defaultHeaders`) and omit unset fields.

#### Scenario: Request conventions
- **WHEN** any interceptor method calls the API
- **THEN** it sends the `Api-Key` header and does not use ETag, `If-Match` or `If-None-Match` headers

#### Scenario: API error
- **WHEN** a request returns a non-2xx status other than the GET/DELETE 404 cases
- **THEN** the error includes the status and response body

#### Scenario: PUT response
- **WHEN** `SaveInterceptor` receives a successful response
- **THEN** the body is treated as `{"name": ...}` only

#### Scenario: CamelCase body
- **WHEN** an interceptor with `display_name`, `forward_auth_token` and `max_retry_attempts` is saved
- **THEN** the request body uses the keys `displayName`, `forwardAuthToken` and `maxRetryAttempts`

#### Scenario: Interceptor not found
- **WHEN** `GetInterceptor` receives `404`
- **THEN** it returns `nil, nil`

#### Scenario: Idempotent delete
- **WHEN** `DeleteInterceptor` receives `404`
- **THEN** it returns success

### Requirement: aidial_interceptor identity

The provider SHALL expose resource `aidial_interceptor` with `id` (computed, `<bucket>/<path>`), `bucket` (required, only `platform`, replace on change) and `path` (required, non-empty, replace on change). See `resource-identity`.

#### Scenario: Resource registration
- **WHEN** provider resources are registered
- **THEN** `aidial_interceptor` is included

#### Scenario: Unsupported bucket
- **WHEN** `bucket` is not `platform`
- **THEN** validation fails at plan time

#### Scenario: Empty or omitted identity
- **WHEN** `bucket` or `path` is omitted, or `path` is an empty string
- **THEN** validation fails at plan time

#### Scenario: Immutable identity
- **WHEN** `bucket` or `path` changes
- **THEN** Terraform plans replacement

### Requirement: Interceptor name handling

`name` SHALL be optional and computed: sent to Core only when configured, otherwise derived by Core and read back. `api_name` SHALL be computed and hold the canonical name Core returns. `author`, `created_at` and `updated_at` SHALL NOT be exposed.

#### Scenario: Name omitted
- **WHEN** `name` is not configured
- **THEN** no name is sent, and `name` in state equals `api_name`

### Requirement: aidial_interceptor attributes

The resource SHALL expose `endpoint` (required string) and the optional attributes `description`, `display_name`, `display_version`, `icon_url` and `catalog_schema_id` (strings), `forward_auth_token` (bool), `max_retry_attempts` (int), `user_roles` (set of strings), `default_headers` (map of strings), `features` and `interfaces` (same shape and validation as on `aidial_model`), and the free-form objects `defaults` and `catalog_properties` as Terraform dynamic attributes. `display_name` is a plain string; the locale-map form is unsupported.

#### Scenario: Minimal interceptor
- **WHEN** a config sets only `bucket`, `path` and `endpoint`
- **THEN** the interceptor is created with just that body, and every other attribute reads back as computed or null

#### Scenario: Endpoint required
- **WHEN** `endpoint` is omitted
- **THEN** validation fails at plan time

#### Scenario: Features and interfaces reuse
- **WHEN** `features` or `interfaces` is configured on an interceptor and on a model with the same values
- **THEN** both are validated, sent and read back identically

#### Scenario: Free-form values keep their form
- **WHEN** Core returns JSON equal to the configured `defaults` or `catalog_properties`
- **THEN** the configured value and its types are kept in state

### Requirement: Attributes DIAL Core fills in

`forward_auth_token` (false) and `max_retry_attempts` (1) are filled in by DIAL Core when a request leaves them out, so they SHALL be optional and computed, keep their value across updates, and not be sent until known.

#### Scenario: Minimal config after apply
- **WHEN** a config sets only `bucket`, `path` and `endpoint`
- **THEN** after apply `forward_auth_token` is false and `max_retry_attempts` is 1, as Core returned, and a re-plan shows no change

### Requirement: Interceptor create, read, update, delete

Create and update SHALL PUT the interceptor and then re-read it with GET to obtain state. A GET `404` during refresh SHALL remove the resource from state. Delete SHALL treat `404` as success. Update SHALL be in place for any attribute other than `bucket` and `path`.

#### Scenario: Create reads back
- **WHEN** an interceptor is created
- **THEN** state reflects the GET result, with `api_name` populated

#### Scenario: Drift removal
- **WHEN** the interceptor no longer exists in Core during refresh
- **THEN** it is removed from state

#### Scenario: In-place update
- **WHEN** `endpoint` or `description` changes
- **THEN** the interceptor is updated in place with PUT

#### Scenario: Delete already gone
- **WHEN** delete receives `404`
- **THEN** it succeeds

### Requirement: Interceptor state consistency

Where DIAL Core leaves an optional attribute out, it SHALL be an explicit null, so a config that leaves it unset plans no change; a configured empty collection SHALL stay empty. `interfaces` and `default_headers` are empty maps when Core has none, as on `aidial_model`. After apply, state SHALL equal the planned values for every configured attribute.

#### Scenario: Omitted optional collections
- **WHEN** Core omits `user_roles`, `features` and `catalog_properties`
- **THEN** they are null in state, `interfaces` and `default_headers` are empty maps, and a re-plan shows no change

#### Scenario: Configured empty collection
- **WHEN** `user_roles = []` is configured
- **THEN** state holds an empty set and a re-plan shows no change

### Requirement: Interceptor import

The resource SHALL support import by `<bucket>/<path>`, taking both attributes from the import id, and then read the interceptor.

#### Scenario: Import
- **WHEN** `terraform import aidial_interceptor.x platform/team/filter` runs
- **THEN** `bucket` is `platform`, `path` is `team/filter` and the attributes are read from Core

#### Scenario: Malformed import id
- **WHEN** the import id has no `/` or an empty bucket or path
- **THEN** import fails with a diagnostic naming the expected format

### Requirement: aidial_interceptor data source

The provider SHALL expose data source `aidial_interceptor` with `bucket` (required, only `platform`) and `path` (required, non-empty), computed `id`, `api_name` and every interceptor attribute the resource exposes, with the same null-vs-empty rules. An interceptor that Core does not have SHALL produce an error diagnostic naming the bucket and path. File-sourced interceptors are the job of `aidial_file_interceptor`.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_interceptor` is included

#### Scenario: Existing interceptor
- **WHEN** the interceptor exists
- **THEN** `id`, `api_name` and the attributes are populated, equal to what the resource would hold

#### Scenario: Missing interceptor
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: API error
- **WHEN** the API returns a non-2xx status other than 404
- **THEN** the data source fails with a diagnostic carrying the status and body
