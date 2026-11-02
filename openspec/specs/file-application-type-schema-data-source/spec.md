# file-application-type-schema-data-source Specification

## Purpose

Let Terraform configurations read a custom application type schema defined in DIAL Core's static configuration, which has no bucket or path and cannot be managed through the API.

## Requirements

### Requirement: File application type schema client method

`internal/client/client.go` SHALL provide `GetFileApplicationTypeSchema(ctx, name)` against `GET /v1/admin/config/file/schemas/{name}`. It SHALL return the schema document without the `name` and `status` keys, together with the `status` value when Core returns one, and `nil` when the schema does not exist (404). There is no save or delete method, because the endpoint is read-only.

#### Scenario: Schema found
- **WHEN** GET returns the schema body plus `name` and `status`
- **THEN** the document contains neither key and `status` is returned separately

#### Scenario: Schema not found
- **WHEN** the API returns 404
- **THEN** the method returns nil without an error

#### Scenario: Request conventions
- **WHEN** the method calls the API
- **THEN** it sends the `Api-Key` header and percent-encodes `name` as one path segment

### Requirement: aidial_file_application_type_schema data source

The provider SHALL expose data source `aidial_file_application_type_schema` with `name` (required, non-empty) and computed `schema` (JSON object string), `schema_id` (the `$id`) and `status`. It has no `bucket`, `path` or `id`. A missing schema SHALL produce an error diagnostic naming the schema.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_file_application_type_schema` is included alongside `aidial_application_type_schema`

#### Scenario: Existing schema
- **WHEN** the static-config schema exists
- **THEN** `schema`, `schema_id` and, when Core returns it, `status` are populated

#### Scenario: Missing schema
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: Name required
- **WHEN** `name` is omitted or empty
- **THEN** validation fails at plan time
