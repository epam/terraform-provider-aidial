# file-alias-data-source Specification

## Purpose

Let Terraform configurations read an alias defined in DIAL Core's static configuration, which has no bucket or path and cannot be managed through the API.

## Requirements

### Requirement: File alias client method

`internal/client/alias.go` SHALL provide `GetFileAlias(ctx, name)` against `GET /v1/admin/config/file/aliases/{name}`. It SHALL return the alias, or `nil` when it does not exist (404). There is no save or delete method, because the endpoint is read-only.

#### Scenario: Alias found
- **WHEN** GET returns an `Alias` body
- **THEN** the method returns it decoded

#### Scenario: Alias not found
- **WHEN** the API returns 404
- **THEN** the method returns nil without an error

#### Scenario: Request conventions
- **WHEN** the method calls the API
- **THEN** it sends the `Api-Key` header and percent-encodes `name` as one path segment

### Requirement: aidial_file_alias data source

The provider SHALL expose data source `aidial_file_alias` with `name` (required, non-empty) and the computed alias attributes of the `aidial_alias` data source, except `bucket`, `path`, `id` and `api_name`, which it does not have. A missing alias SHALL produce an error diagnostic naming it.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_file_alias` is included alongside `aidial_alias`

#### Scenario: Existing alias
- **WHEN** the static-config alias exists
- **THEN** its attributes are populated with the same null-vs-empty rules as the resource

#### Scenario: Missing alias
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: Name required
- **WHEN** `name` is omitted or empty
- **THEN** validation fails at plan time
