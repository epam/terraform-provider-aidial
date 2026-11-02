# Proposal

## Why

`bucket` and `path` are optional today: `bucket` defaults to `platform` and `path` defaults to `name`. The implicit `path = name` coupling hides where a resource lands, and for `aidial_model`, `aidial_role` and `aidial_route` it ties the replace-on-change identity to the display `name`. The rules are also inconsistent: `aidial_key` and `aidial_catalog_schema` already require `path`, and the data sources differ from each other and from their resources. Verified against a live DIAL Core, every one of these resources exists only in the `platform` bucket (see design.md), so the identity is fully explicit once both attributes are written out.

## What Changes

- **BREAKING (validation)**: `bucket` and `path` become required, with no default and no computed fallback, on `aidial_model`, `aidial_role`, `aidial_route`, `aidial_key` and `aidial_catalog_schema`. A configuration that omits either one fails validation at plan time. This ships in a 0.2.x release with no separate breaking-change release note; the behavior is stated here, in the specs and in the generated docs.
- `bucket` accepts only `platform`. Any other value fails validation at plan time instead of surfacing as a 404 from DIAL Core at apply time.
- `path` must be non-empty.
- The data sources `aidial_role`, `aidial_model` and `aidial_catalog_schema` take the same two required inputs, with the same validation, as their resources. `aidial_model` stops defaulting `path` to `name`.
- `bucket` and `path` keep replace-on-change; DIAL Core has no move operation for these resource types. Import stays `<bucket>/<path>`.
- Remove the default machinery: `bucketDefaultsToPlatform`, `pathDefaultsToName`, the `defaultBucket` constant and the `stringdefault` on `aidial_key`/`aidial_catalog_schema`, with their tests.
- Rename the cross-resource spec `resource-identity-defaults` to `resource-identity` and rewrite it as required-identity rules.
- Update integration manifests, `stub/` models and `examples/` so every resource and data source sets `bucket` and `path`; delete the three `*_omitted.tf` integration manifests; regenerate `docs/`.
- `aidial_file_model` and `aidial_bucket` are unchanged (they have no bucket/path).

## Capabilities

### New Capabilities

- `resource-identity`: required `bucket`/`path` identity rules shared by all resources and data sources (replaces `resource-identity-defaults`).

### Modified Capabilities

- `model-schema`: `bucket`/`path` attribute definitions.
- `role-schema`: `bucket`/`path` attribute definitions.
- `route-schema`: `bucket`/`path` attribute definitions.
- `catalog-schema-resource`: `bucket` becomes required on the resource and the data source.
- `model-data-source`: `bucket`/`path` become required (this capability exists only once the in-flight `add-model-data-source` change is archived).

## Impact

- Code: `internal/provider/{model,role,route,key,catalog_schema}_resource.go`, `{role,model,catalog_schema}_data_source.go`, `helpers.go` (default helpers removed), plus their unit and acceptance tests.
- Tests and fixtures: `tests/integration/*.tf`, `stub/models/*`, `examples/`, `internal/provider/*_test.go`.
- Docs: `docs/` regenerated (`make docs`).
- Specs: new `resource-identity`; the five schema/data-source specs above; `resource-identity-defaults` removed.
- State: no schema version bump. State already holds both attributes, so existing state stays valid; only configuration must now be explicit.
- Out of scope: allowing buckets other than `platform`, a move/rename operation, and any other attribute changes.
