# application-type-schema-resource Specification

## Purpose

Manage DIAL Core custom application type schemas (JSON Schema documents under `/v1/schemas`) through the `aidial_application_type_schema` resource and data source, so they can be created, imported and looked up from Terraform.

## Requirements

### Requirement: Application type schema client methods

`internal/client/client.go` SHALL provide `GetApplicationTypeSchema(ctx, bucket, path)`, `SaveApplicationTypeSchema(ctx, bucket, path, schema)` and `DeleteApplicationTypeSchema(ctx, bucket, path)` against `/v1/schemas/{bucket}/{path}`, following the existing client conventions. The schema is carried as an opaque JSON object; the metadata keys `name` and `status` that Core adds on GET SHALL be removed.

#### Scenario: Request conventions
- **WHEN** any application type schema method calls the API
- **THEN** it sends the `Api-Key` header and does not use ETag, `If-Match` or `If-None-Match` headers

#### Scenario: API error
- **WHEN** a request returns a non-2xx status other than the GET/DELETE 404 cases
- **THEN** the error includes the status and response body

#### Scenario: PUT response
- **WHEN** `SaveApplicationTypeSchema` receives a successful response
- **THEN** the body is treated as `{"name": ...}` only

#### Scenario: Schema not found
- **WHEN** `GetApplicationTypeSchema` receives `404`
- **THEN** it returns `nil, nil`

#### Scenario: Metadata stripped
- **WHEN** GET returns the schema body plus `name` and `status`
- **THEN** the returned document contains neither key

#### Scenario: Idempotent delete
- **WHEN** `DeleteApplicationTypeSchema` receives `404`
- **THEN** it returns success

### Requirement: aidial_application_type_schema resource

The provider SHALL expose resource `aidial_application_type_schema` with:

- `id` (computed, `<bucket>/<path>`)
- `bucket` (required, must be `platform`, replace on change; see `resource-identity`)
- `path` (required, non-empty, replace on change; see `resource-identity`)
- `schema` (required, string containing a JSON object, intended for `jsonencode`)
- `schema_id` (computed, the `$id` of the schema)

At plan time `schema` MUST be a JSON object with a non-blank string `$id`.

#### Scenario: Resource registration
- **WHEN** provider resources are registered
- **THEN** `aidial_application_type_schema` is included

#### Scenario: Invalid schema
- **WHEN** `schema` is not valid JSON, is not an object, or lacks a non-blank `$id`
- **THEN** validation fails with a diagnostic before any API call

#### Scenario: Immutable identity
- **WHEN** `bucket` or `path` changes
- **THEN** Terraform plans replacement

#### Scenario: Identity attributes required
- **WHEN** `bucket` or `path` is omitted
- **THEN** validation fails at plan time

### Requirement: Application type schema lifecycle

Create and update SHALL PUT the schema, re-read via GET and write state from the response; a missing schema after a successful save is an error. Read SHALL remove the resource from state on 404. Delete SHALL treat 404 as success. Import SHALL accept `<bucket>/<path>` and reject other formats with a usage diagnostic. State SHALL keep the configured `schema` text when the API returns semantically equal JSON.

#### Scenario: Create then re-read
- **WHEN** a valid schema is applied
- **THEN** the schema is saved, re-read, and `schema_id` equals its `$id`

#### Scenario: No-op plan
- **WHEN** the API returns the same schema with different key order
- **THEN** the next plan shows no changes

#### Scenario: Out-of-band deletion
- **WHEN** the schema is deleted outside Terraform
- **THEN** refresh removes it from state

#### Scenario: Invalid import ID
- **WHEN** the import ID is not `<bucket>/<path>`
- **THEN** import fails with a usage diagnostic

### Requirement: aidial_application_type_schema data source

The provider SHALL expose data source `aidial_application_type_schema` with `bucket` (required, must be `platform`), `path` (required, non-empty), and computed `id`, `schema` (JSON string) and `schema_id`. A missing schema SHALL produce an error diagnostic. The data source MUST NOT read file-sourced schemas; those remain the job of `aidial_file_application_type_schema`.

#### Scenario: Existing schema
- **WHEN** the schema exists
- **THEN** `schema` and `schema_id` are populated and `id` is `<bucket>/<path>`

#### Scenario: Missing schema
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: Identity attributes required
- **WHEN** `bucket` or `path` is omitted
- **THEN** validation fails at plan time
