# Proposal

## Why

DIAL Core documents interceptors (`/v1/interceptors/{bucket}/{path}`, schema `Interceptor`): services that sit in the request pipeline and inspect or modify chat completions, referenced by name from a model's `interceptors` list. The provider manages models, routes, roles, keys, schemas and toolsets, and already lets a model name interceptors, but it cannot create, read or look up the interceptors themselves, nor read those defined in Core's static configuration. Users must manage them out of band.

## What Changes

- Add resource `aidial_interceptor` (create, read, update, delete, import) over `/v1/interceptors/{bucket}/{path}`, with typed attributes mapped from the OpenAPI `Interceptor` schema in snake_case: `endpoint` (required), `description`, `display_name`, `display_version`, `icon_url`, `forward_auth_token`, `max_retry_attempts`, `user_roles`, `features`, `interfaces`, `defaults`, `default_headers`, `catalog_properties` and `catalog_schema_id`. `defaults` and `catalog_properties` are Terraform dynamic attributes with the same semantic-equality behavior as `aidial_model`.
- Add data source `aidial_interceptor` that looks up an API-managed interceptor by `bucket` and `path`.
- Add data source `aidial_file_interceptor` that reads a static-config interceptor by `name` only (`/v1/admin/config/file/interceptors/{name}`).
- Reuse the shared `features` and `interfaces` helpers from `deployment_shared.go`: `Interceptor` references the same OpenAPI `Features` and `FeaturesData` schemas and the same `DeploymentInterface` map as models and toolsets.
- Identity follows the model rule: `bucket` and `path` are required with no defaults, `bucket` accepts only `platform` (pending the real-Core check in design.md), both force replacement, `id` is `<bucket>/<path>`, import is `<bucket>/<path>`.
- Add examples, regenerated `docs/`, README and `CHANGELOG.md` updates, and an integration manifest.

## Capabilities

### New Capabilities

- `interceptor-resource`: client methods, the `aidial_interceptor` resource and the API-managed data source of the same name.
- `file-interceptor-data-source`: the `aidial_file_interceptor` data source for static-config interceptors.

### Modified Capabilities

- `resource-identity`: the `platform`-only `bucket` rule also covers `aidial_interceptor` and its data source.
- `provider-identity`: the registered resource and data source type list gains the three new types.
- `core-client-requests`: the percent-encoding rule also covers interceptor `bucket`/`path` and file-sourced interceptor `name`.

## Impact

- Code: new `internal/client/interceptor.go`; new `internal/provider/interceptor_resource.go`, `interceptor_data_source.go`, `file_interceptor_data_source.go`; registration in `internal/provider/provider.go`. The shared helpers are reused as they are.
- Tests: client and provider unit tests, an acceptance test (`TestAcc*`, skipped without `TF_ACC`), a manifest in `tests/integration/`.
- Docs: generated `docs/` pages, `examples/` snippets, README, `CHANGELOG.md` under `Unreleased`, and the `AGENTS.md` architecture and identity bullets.
- Not breaking: no change to existing resources, state or schemas. `stub/models/` is untouched.
- Out of scope: wiring interceptors into `aidial_model` (it already accepts an `interceptors` list of names), the fields `author`, `createdAt`, `updatedAt`, `dependencies`, `description_keywords`, `descriptionKeywords`, `input_attachment_types`, `max_input_attachments`, `responses_defaults`, `responses_endpoint`, `base_url`, `override_name`, `reference`, `intro` and the nested `interceptors` list, the locale-map form of `display_name`, the read-only runtime endpoints, and an interceptor listing data source.
