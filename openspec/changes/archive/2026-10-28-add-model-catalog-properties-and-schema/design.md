## Context

The OpenAPI contract (`openapicore/open_api_core.yaml`) defines `Model.catalogSchemaId` (`string`, `format: uri`) and `Model.catalogProperties` (`MapStringObject`, i.e. arbitrary JSON values). A catalog schema is `CatalogSchema`: a JSON Schema with a required non-blank `$id` plus arbitrary extra keys (including `dial`-prefixed extensions). It is CRUD-managed at `/v1/catalog_schemas/{bucket}/{path}` (`x-preview: true`); PUT returns only `{"name": ...}`; GET returns the schema body merged with `EntityMetadata` (`name`, `status`). Existing resources follow the same bucket/path identity, GET-404-is-absent, DELETE-404-is-noop, re-read-after-write conventions.

## Goals / Non-Goals

**Goals:** manage `catalogSchemaId`/`catalogProperties` on models without plan churn; manage the referenced schema as a first-class resource so a model and its schema can live in one configuration.

**Non-Goals:** client-side validation of properties against the schema; ETag handling (same known gap as other resources); the other unmapped model fields.

## Decisions

1. **`catalog_properties` is a dynamic attribute**, reusing the `defaults` helpers in `model_defaults.go` (`defaultsFromPlan`/`defaultsToState` semantics: keep the prior value when the API JSON is semantically equal; unset stays null when the API returns `{}`/omits it; configured `{}` stays `{}`). Rationale: values are schema-defined and may be non-strings (numbers, arrays), and the user's sample uses only strings today, so a `map(string)` would block legitimate schemas. Alternative considered: `map(string)`; rejected for that reason. The helpers should be generalised/renamed rather than copied.
2. **`catalog_schema_id` is a plain optional string** (not computed, matching `override_name`/`defaults`: unset stays null), no cross-resource reference enforcement. Users may reference `aidial_catalog_schema.x.schema_id` to get implicit dependency ordering.
3. **`aidial_catalog_schema` identity**: `bucket` (default `platform`) and required `path`, both force-replace, `id = <bucket>/<path>`. The schema has no `name`, so `path` is required, like `aidial_key` (this is a documented exception to the "path defaults to name" rule in `resource-identity-defaults`).
4. **Schema body is a JSON string attribute `schema`** (intended for `jsonencode(...)`), required. Typed attributes are impractical because the document is open-ended JSON Schema. Plan-time validation: valid JSON object with non-blank string `$id`. Semantic JSON equality on read keeps the configured text when the API returns equivalent JSON (key order/whitespace). Computed `schema_id` mirrors `$id`.
5. **Read strips `EntityMetadata`**: the GET body includes `name` and `status` injected by Core. These are removed before comparison/storage unless they were part of the configured schema (JSON Schema has no `name`/`status` keywords, so they are stripped unconditionally; to be confirmed against the ephemeral Core in the integration job).
6. **Data source `aidial_catalog_schema`** takes `bucket` (default `platform`) and `path`, returns `schema` (JSON string) and `schema_id`; a 404 is an error for a data source, matching `aidial_role`.
7. Specs for the model attributes are new capability files; the legacy `openspec/model-*/spec.md` are edited by hand to drop the "not implemented" notes for these two fields.

## Risks / Trade-offs

- `/v1/catalog_schemas/{bucket}/{path}` is `x-preview`; the contract may change. Mitigation: integration test against real Core in CI.
- Older Core versions may lack the endpoint or reject the model fields. Mitigation: fields are optional; resource errors surface API status/body.
- Core may validate `catalogProperties` against the schema on save and reject mismatches; this is surfaced as an API error diagnostic (assumption to confirm in integration test).
- Whether Core echoes `catalogSchemaId` back with a default for models that omit it is unverified; an unset attribute would then show drift; revisit making it computed if the integration run shows Core adding a default.

## Verified Against Core (CI integration job)

- Saving a model whose `catalogSchemaId` has no matching schema fails with `400 Catalog properties validation failed: Catalog schema not found: <id>`, so the schema must exist first. Docs and the integration manifest therefore reference `aidial_catalog_schema.<name>.schema_id`, which orders creation.
- Core resolves `catalogSchemaId` by the schema's `$id`, not by its bucket/path: a schema stored at an unrelated path was found by its `$id`.
- A model without `catalog_schema_id`/`catalog_properties` is created and read back with no plan churn (the attributes stay null), so optional-only is correct.
