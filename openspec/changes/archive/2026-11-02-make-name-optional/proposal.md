# Proposal

## Why

`name` is required on `aidial_model`, `aidial_role` and `aidial_route`, but DIAL Core does not use it. Verified against a live Core on 2026-11-02: a PUT to `/v1/{models,roles,routes}/platform/<path>` succeeds with no `name` in the body, and a role PUT with `name = "custom"` reads back with the name derived from the path. Identity is `bucket` and `path` alone (see `require-bucket-path-identity`), so a required `name` forces users to write a value that Core ignores and that can disagree with `path`.

## What Changes

- `name` becomes optional and computed on `aidial_model`, `aidial_role` and `aidial_route`. Existing configurations that set `name` stay valid and behave exactly as before.
- When `name` is configured, the provider sends it and keeps the configured value in state, never rewriting it from the API. `api_name` keeps the API's value.
- When `name` is omitted, the provider sends no `name` and `name` is computed from the API response on create or import, then kept stable across plans so it causes no drift.
- The `aidial_model` data source stops requiring `name` (the lookup is by `bucket` and `path`); it becomes optional and computed.
- The client stops sending an empty `name`: `name` is omitted from the request body when unset.
- `aidial_key`, `aidial_catalog_schema`, `aidial_role` data source and `aidial_file_model` are unchanged. `aidial_file_model` still looks up by `name`.
- No state migration and no release note; ships as a 0.2.x release. Changing `name` never replaces a resource.

## Capabilities

### New Capabilities

<!-- None. -->

### Modified Capabilities

- `model-schema`, `role-schema`, `route-schema`: `name` attribute definition and name handling.
- `model-state-consistency`, `role-state-consistency`, `route-state-consistency`: `name` is preserved when known and computed once when omitted.
- `role-read`: a still-unknown `name` is set from the API response.
- `model-data-source`: `name` no longer required (exists only once `add-model-data-source` is archived).

## Impact

- Code: `internal/provider/{model,role,route}_resource.go` (schema, request building, `set*State`), `model_data_source.go`, `internal/client/client.go` (`name` tags to `omitempty`).
- Tests: unit tests for omitted and configured `name`, a schema-upgrade check that existing state stays valid, acceptance tests with `name` omitted.
- Fixtures and docs: `name` may be dropped where redundant in `tests/integration`, `examples/` and the README; regenerate `docs/`.
- Depends on `require-bucket-path-identity`, which makes `path` required and removes the `name` fallback.
- Out of scope: removing `name` entirely; changing `aidial_key`.
