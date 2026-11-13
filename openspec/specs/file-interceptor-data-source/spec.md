# file-interceptor-data-source Specification

## Purpose

Let Terraform configurations read an interceptor defined in DIAL Core's static configuration, which has no bucket or path and cannot be managed through the API.

## Requirements

### Requirement: File interceptor client method

`internal/client/interceptor.go` SHALL provide `GetFileInterceptor(ctx, name)` against `GET /v1/admin/config/file/interceptors/{name}`. It SHALL return the interceptor, or `nil` when it does not exist (404). There is no save or delete method, because the endpoint is read-only.

#### Scenario: Interceptor found
- **WHEN** GET returns an `Interceptor` body
- **THEN** the method returns it decoded

#### Scenario: Interceptor not found
- **WHEN** the API returns 404
- **THEN** the method returns nil without an error

#### Scenario: Request conventions
- **WHEN** the method calls the API
- **THEN** it sends the `Api-Key` header and percent-encodes `name` as one path segment

### Requirement: aidial_file_interceptor data source

The provider SHALL expose data source `aidial_file_interceptor` with `name` (required, non-empty) and the computed interceptor attributes of the `aidial_interceptor` data source, except `bucket`, `path`, `id` and `api_name`, which it does not have. A missing interceptor SHALL produce an error diagnostic naming it.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_file_interceptor` is included alongside `aidial_interceptor`

#### Scenario: Existing interceptor
- **WHEN** the static-config interceptor exists
- **THEN** its attributes are populated with the same null-vs-empty rules as the resource

#### Scenario: Missing interceptor
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: Name required
- **WHEN** `name` is omitted or empty
- **THEN** validation fails at plan time
