# file-toolset-data-source Specification

## Purpose

Let Terraform configurations read a toolset defined in DIAL Core's static configuration, which has no bucket or path and cannot be managed through the API.

## Requirements

### Requirement: File toolset client method

`internal/client/client.go` SHALL provide `GetFileToolset(ctx, name)` against `GET /v1/admin/config/file/toolsets/{name}`. It SHALL return the toolset, or `nil` when it does not exist (404). There is no save or delete method, because the endpoint is read-only.

#### Scenario: Toolset found
- **WHEN** GET returns a `ToolSet` body
- **THEN** the method returns it decoded

#### Scenario: Toolset not found
- **WHEN** the API returns 404
- **THEN** the method returns nil without an error

#### Scenario: Request conventions
- **WHEN** the method calls the API
- **THEN** it sends the `Api-Key` header and percent-encodes `name` as one path segment

### Requirement: aidial_file_toolset data source

The provider SHALL expose data source `aidial_file_toolset` with `name` (required, non-empty) and the computed toolset attributes of the `aidial_toolset` data source, except `bucket`, `path`, `id` and `api_name`, which it does not have. It SHALL NOT expose `auth_settings.client_secret`. A missing toolset SHALL produce an error diagnostic naming it.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_file_toolset` is included alongside `aidial_toolset`

#### Scenario: Existing toolset
- **WHEN** the static-config toolset exists
- **THEN** its attributes are populated with the same null-vs-empty rules as the resource

#### Scenario: Missing toolset
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: Name required
- **WHEN** `name` is omitted or empty
- **THEN** validation fails at plan time
