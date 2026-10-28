# Proposal

## Why

Some environments' model JSON carries `catalogSchemaId` and `catalogProperties` (provider, vendor, license, knowledge cutoff, parameter count, ...). `aidial_model` does not map them, so such models cannot be managed or round-tripped by the provider. `catalogSchemaId` points at a DIAL Core *catalog schema* (a JSON Schema document stored at `/v1/catalog_schemas/{bucket}/{path}`), and the provider has no way to manage those schemas either.

## What Changes

- `aidial_model` gains `catalog_schema_id` (string, URI) and `catalog_properties` (free-form object, sent as `catalogProperties`), both optional (not computed, like `defaults` and `override_name`), with null-vs-empty handling consistent with the existing model attributes. Both are also exposed by the `aidial_file_model` data source.
- New resource `aidial_catalog_schema` managing a catalog schema document (`PUT/GET/DELETE /v1/catalog_schemas/{bucket}/{path}`), and a matching `aidial_catalog_schema` data source.
- Client gains `GetCatalogSchema`, `SaveCatalogSchema`, `DeleteCatalogSchema`; `Model` gains the two new fields.
- Docs, examples, and the stale "not implemented" notes in `openspec/model-schema` are updated.

Out of scope: validating `catalog_properties` against the referenced schema client-side, resolving or validating that `catalog_schema_id` matches an existing schema's `$id`, the other unmapped model fields (`embedding_dimensions`, `base_url`, `responses_defaults`, `created_at`, `updated_at`), the file-sourced catalog schema admin endpoints, and the read-only `/v1/catalog_schemas/{schema,schemas,meta_schema}` endpoints.

## Capabilities

### New Capabilities
- `model-catalog-metadata`: `catalog_schema_id` and `catalog_properties` on `aidial_model` and `aidial_file_model`: schema, payload mapping, state consistency.
- `catalog-schema-resource`: `aidial_catalog_schema` resource and data source, plus the catalog schema client methods.

### Modified Capabilities
<!-- None as OpenSpec deltas: the existing model-* capabilities live in legacy top-level openspec/<capability>/spec.md files, not under openspec/specs/, so they are updated by hand in tasks.md. -->

## Impact

- Code: `internal/client/client.go`, `internal/provider/model_resource.go`, `file_model_data_source.go`, new `catalog_schema_resource.go` / `catalog_schema_data_source.go`, `provider.go` registration, reuse of `model_defaults.go` JSON helpers.
- Docs: regenerated `docs/` (`make docs`), new `examples/` snippets.
- Tests: unit tests per package; integration manifest in `tests/integration/` and drift check for null vs empty.
- Non-breaking: new optional attributes. Existing state upgrades with the new attributes null/computed; no schema version bump expected.
