# alias-resource Specification

## Purpose

Manage DIAL Core aliases (`/v1/aliases/{bucket}/{path}`) through the `aidial_alias` resource and data source, so they can be created, imported and looked up from Terraform.

## ADDED Requirements

### Requirement: Alias client methods

`internal/client/alias.go` SHALL provide `GetAlias(ctx, bucket, path)`, `SaveAlias(ctx, bucket, path, alias)` and `DeleteAlias(ctx, bucket, path)` against `/v1/aliases/{bucket}/{path}`, following the existing client conventions. The body SHALL use the JSON keys of the OpenAPI `Alias` schema (`presentAs`, `target` with only `path`, `userRoles`, `displayName`, `displayVersion`, `iconUrl`, `intro`, `descriptionKeywords`) and omit unset fields.

#### Scenario: Request conventions
- **WHEN** any alias method calls the API
- **THEN** it sends the `Api-Key` header and does not use ETag, `If-Match` or `If-None-Match` headers

#### Scenario: API error
- **WHEN** a request returns a non-2xx status other than the GET/DELETE 404 cases
- **THEN** the error includes the status and response body

#### Scenario: PUT response
- **WHEN** `SaveAlias` receives a successful response
- **THEN** the body is treated as `{"name": ...}` only

#### Scenario: Body keys
- **WHEN** an alias with `present_as`, `user_roles` and `display_name` is saved
- **THEN** the request body uses the keys `presentAs`, `userRoles` and `displayName`

#### Scenario: Alias not found
- **WHEN** `GetAlias` receives `404`
- **THEN** it returns `nil, nil`

#### Scenario: Idempotent delete
- **WHEN** `DeleteAlias` receives `404`
- **THEN** it returns success

### Requirement: aidial_alias identity

The provider SHALL expose resource `aidial_alias` with `id` (computed, `<bucket>/<path>`), `bucket` (required, `public` or `platform`, replace on change) and `path` (required, non-empty, replace on change). See `resource-identity`.

#### Scenario: Resource registration
- **WHEN** provider resources are registered
- **THEN** `aidial_alias` is included

#### Scenario: Unsupported bucket
- **WHEN** `bucket` is neither `public` nor `platform`
- **THEN** validation fails at plan time

#### Scenario: Empty or omitted identity
- **WHEN** `bucket` or `path` is omitted, or `path` is an empty string
- **THEN** validation fails at plan time

#### Scenario: Immutable identity
- **WHEN** `bucket` or `path` changes
- **THEN** Terraform plans replacement

### Requirement: Alias name handling

`name` SHALL be optional and computed: sent to Core only when configured, otherwise derived by Core and read back. `api_name` SHALL be computed and hold the canonical name Core returns. `author`, `created_at` and `updated_at` SHALL NOT be exposed, and neither SHALL the read-only `status`.

#### Scenario: Name omitted
- **WHEN** `name` is not configured
- **THEN** no name is sent, and `name` in state equals `api_name`

### Requirement: aidial_alias attributes

The resource SHALL expose `present_as` (required, one of `model`, `application`, `toolset`), `target_path` (required, `models/<name>`, `applications/<name>` or `toolsets/<name>`) and the optional attributes `user_roles`, `display_name`, `description`, `display_version`, `icon_url`, `intro` and `description_keywords`. `target_path` is sent as `target.path`; Core accepts no other target field, so none is exposed.

#### Scenario: Minimal alias
- **WHEN** a config sets only `bucket`, `path`, `present_as` and `target_path`
- **THEN** the alias is created with just that body, and every other attribute reads back as null

#### Scenario: present_as required
- **WHEN** `present_as` is omitted
- **THEN** validation fails at plan time

#### Scenario: Unknown present_as
- **WHEN** `present_as` is not `model`, `application` or `toolset`
- **THEN** validation fails at plan time

#### Scenario: target_path required
- **WHEN** `target_path` is omitted
- **THEN** validation fails at plan time

#### Scenario: Malformed target_path
- **WHEN** `target_path` does not start with `models/`, `applications/` or `toolsets/` followed by a non-empty name
- **THEN** validation fails at plan time

#### Scenario: present_as and target kind
- **WHEN** `present_as` is `toolset` and `target_path` is not a `toolsets/` path, or `present_as` is `model` or `application` and `target_path` is a `toolsets/` path
- **THEN** validation fails at plan time, and any other combination is left to Core, whose 400 and `validationWarnings` are shown

#### Scenario: Description keywords
- **WHEN** `description_keywords` is configured
- **THEN** it is sent as `descriptionKeywords` and read back as a set of strings

### Requirement: Alias create, read, update, delete

Create and update SHALL PUT the alias and then re-read it with GET to obtain state. A GET `404` during refresh SHALL remove the resource from state. Delete SHALL treat `404` as success. Update SHALL be in place for any attribute other than `bucket` and `path`.

#### Scenario: Create reads back
- **WHEN** an alias is created
- **THEN** state reflects the GET result, with `api_name` populated

#### Scenario: Drift removal
- **WHEN** the alias no longer exists in Core during refresh
- **THEN** it is removed from state

#### Scenario: In-place update
- **WHEN** `target_path` or `display_name` changes
- **THEN** the alias is updated in place with PUT

#### Scenario: Delete already gone
- **WHEN** delete receives `404`
- **THEN** it succeeds

### Requirement: Alias state consistency

Where DIAL Core leaves an optional attribute out, it SHALL be an explicit null, so a config that leaves it unset plans no change; a configured empty collection SHALL stay empty. After apply, state SHALL equal the planned values for every configured attribute. Core fills in nothing but `name`, so no attribute is computed.

#### Scenario: Omitted optionals
- **WHEN** Core omits `user_roles`, `intro` and `description_keywords`
- **THEN** they are null in state and a re-plan shows no change

#### Scenario: Configured empty collection
- **WHEN** `user_roles = []` or `description_keywords = []` is configured
- **THEN** state holds an empty set and a re-plan shows no change

### Requirement: Alias import

The resource SHALL support import by `<bucket>/<path>`, taking both attributes from the import id, and then read the alias.

#### Scenario: Import
- **WHEN** `terraform import aidial_alias.x public/my-alias` runs
- **THEN** `bucket` is `public`, `path` is `my-alias` and the attributes are read from Core

#### Scenario: Malformed import id
- **WHEN** the import id has no `/` or an empty bucket or path
- **THEN** import fails with a diagnostic naming the expected format

### Requirement: aidial_alias data source

The provider SHALL expose data source `aidial_alias` with `bucket` (required, `public` or `platform`) and `path` (required, non-empty), computed `id`, `api_name` and every alias attribute the resource exposes, with the same null-vs-empty rules. An alias that Core does not have SHALL produce an error diagnostic naming the bucket and path. File-sourced aliases are the job of `aidial_file_alias`.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_alias` is included

#### Scenario: Existing alias
- **WHEN** the alias exists
- **THEN** `id`, `api_name` and the attributes are populated, equal to what the resource would hold

#### Scenario: Missing alias
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: API error
- **WHEN** the API returns a non-2xx status other than 404
- **THEN** the data source fails with a diagnostic carrying the status and body
