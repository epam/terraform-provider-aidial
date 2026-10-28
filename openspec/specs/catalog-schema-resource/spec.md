# catalog-schema-resource Specification

## Purpose

TBD - created by archiving change add-model-catalog-properties-and-schema. Update Purpose after archive.

## Requirements

### Requirement: Catalog schema client methods

`internal/client/client.go` SHALL provide `GetCatalogSchema(ctx, bucket, path)`, `SaveCatalogSchema(ctx, bucket, path, schema)` and `DeleteCatalogSchema(ctx, bucket, path)` against `/v1/catalog_schemas/{bucket}/{path}`, following the existing client conventions: `Api-Key` header, GET 404 returns `(nil, nil)`, DELETE 404 is success, other non-2xx are API errors with status and body, PUT response treated as `{"name": ...}` only, ETag headers not used. The schema is carried as an opaque JSON object; the metadata keys `name` and `status` that Core adds on GET SHALL be removed.

#### Scenario: Schema not found
- **WHEN** `GetCatalogSchema` receives `404`
- **THEN** it returns `nil, nil`

#### Scenario: Metadata stripped
- **WHEN** GET returns the schema body plus `name` and `status`
- **THEN** the returned document contains neither key

#### Scenario: Idempotent delete
- **WHEN** `DeleteCatalogSchema` receives `404`
- **THEN** it returns success

### Requirement: aidial_catalog_schema resource

The provider SHALL expose resource `aidial_catalog_schema` with:

- `id` (computed, `<bucket>/<path>`)
- `bucket` (optional, computed, default `platform`, replace on change)
- `path` (required, replace on change; there is no name to default from, as with `aidial_key`)
- `schema` (required, string containing a JSON object, intended for `jsonencode`)
- `schema_id` (computed, the `$id` of the schema)

At plan time `schema` MUST be a JSON object with a non-blank string `$id`.

#### Scenario: Resource registration
- **WHEN** provider resources are registered
- **THEN** `aidial_catalog_schema` is included

#### Scenario: Invalid schema
- **WHEN** `schema` is not valid JSON, is not an object, or lacks a non-blank `$id`
- **THEN** validation fails with a diagnostic before any API call

#### Scenario: Immutable identity
- **WHEN** `bucket` or `path` changes
- **THEN** Terraform plans replacement

### Requirement: Catalog schema lifecycle

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

### Requirement: aidial_catalog_schema data source

The provider SHALL expose data source `aidial_catalog_schema` with `bucket` (optional, default `platform`), `path` (required), and computed `id`, `schema` (JSON string) and `schema_id`. A missing schema SHALL produce an error diagnostic.

#### Scenario: Existing schema
- **WHEN** the schema exists
- **THEN** `schema` and `schema_id` are populated

#### Scenario: Missing schema
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic
