# Spec Delta

## MODIFIED Requirements

### Requirement: aidial_catalog_schema resource

The provider SHALL expose resource `aidial_catalog_schema` with:

- `id` (computed, `<bucket>/<path>`)
- `bucket` (required, must be `platform`, replace on change; see `resource-identity`)
- `path` (required, non-empty, replace on change; see `resource-identity`)
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

#### Scenario: Identity attributes required
- **WHEN** `bucket` or `path` is omitted
- **THEN** validation fails at plan time

### Requirement: aidial_catalog_schema data source

The provider SHALL expose data source `aidial_catalog_schema` with `bucket` (required, must be `platform`), `path` (required, non-empty), and computed `id`, `schema` (JSON string) and `schema_id`. A missing schema SHALL produce an error diagnostic.

#### Scenario: Existing schema
- **WHEN** the schema exists
- **THEN** `schema` and `schema_id` are populated

#### Scenario: Missing schema
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: Identity attributes required
- **WHEN** `bucket` or `path` is omitted
- **THEN** validation fails at plan time
