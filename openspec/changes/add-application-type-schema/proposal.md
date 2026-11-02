# Proposal

## Why

DIAL Core 0.49.4 documents custom application type schemas (`/v1/schemas/{bucket}/{path}`, `ApplicationTypeSchema`): JSON Schema documents, keyed by `$id`, that describe a custom application type. The provider manages catalog schemas but has no way to create, read or look up application type schemas, nor to read the ones defined in Core's static configuration. Users must manage them out of band.

## What Changes

- Add resource `aidial_application_type_schema` (create, read, update, delete, import) over `/v1/schemas/{bucket}/{path}`, modelled on `aidial_catalog_schema`: `bucket`, `path`, `schema` (JSON object string with a non-blank `$id`), computed `id` and `schema_id`.
- Add data source `aidial_application_type_schema` that looks up an API-managed schema by `bucket` and `path`.
- Add data source `aidial_file_application_type_schema` that reads a static-config schema by `name` only (`/v1/admin/config/file/schemas/{name}`), exposing `schema`, `schema_id` and `status`.
- Same identity rules as the other resources: `bucket` must be `platform`, `path` non-empty, both force replacement, `id` is `<bucket>/<path>`.
- Keep the configured `schema` text when the API returns semantically equal JSON; strip Core's `name`/`status` metadata on read.
- Commit the pending `openapicore/` refresh to tag 0.49.4 (only `info.version` changes, 0.49.2 to 0.49.3) together with this change.
- Add examples, regenerated `docs/`, README and `AGENTS.md` updates, and an integration manifest.

## Capabilities

### New Capabilities

- `application-type-schema-resource`: client methods, the `aidial_application_type_schema` resource and the API-managed data source of the same name.
- `file-application-type-schema-data-source`: the `aidial_file_application_type_schema` data source for static-config schemas.

### Modified Capabilities

- `provider-identity`: the registered resource and data source type list gains the three new types.
- `core-client-requests`: the percent-encoding requirement also covers application type schemas (`bucket`/`path`) and file-sourced schema `name`.

## Impact

- Code: new `internal/provider/application_type_schema_resource.go`, `application_type_schema_data_source.go`, `file_application_type_schema_data_source.go`; new client methods in `internal/client/client.go`; registration in `internal/provider/provider.go`.
- Tests: client and provider unit tests, an acceptance test (`TestAcc*`), a manifest in `tests/integration/`.
- Docs: generated `docs/` pages, `examples/` snippets, README, `AGENTS.md` and a `CHANGELOG.md` entry under `Unreleased` (existing `public-repo-readiness` rule, no requirement change).
- Not breaking: no change to existing resources, state or schemas. `stub/models/` is untouched.
- Out of scope: a metaschema data source (`/v1/application_type_schemas/meta_schema`), a listing data source, and a file-config catalog schema data source (`/v1/admin/config/file/catalog_schemas/{name}`), which is a candidate follow-up.
