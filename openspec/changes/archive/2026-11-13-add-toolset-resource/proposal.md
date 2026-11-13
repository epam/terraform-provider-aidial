# Proposal

## Why

DIAL Core documents toolsets (`/v1/toolsets/{bucket}/{toolset_path}`, schema `ToolSet`): MCP-style tool servers that applications and models can use, with their endpoint, transport and upstream authentication. The provider manages models, routes, roles, keys and schemas but has no way to create, read or look up toolsets, nor to read the ones defined in Core's static configuration. Users must manage them out of band.

## What Changes

- Add resource `aidial_toolset` (create, read, update, delete, import) over `/v1/toolsets/{bucket}/{path}`, with typed attributes mapped one-to-one from the OpenAPI `ToolSet` schema in snake_case. Free-form objects (`defaults`, `responses_defaults`, `catalog_properties`) are Terraform dynamic attributes with the same semantic-equality behavior as `aidial_model`.
- Add data source `aidial_toolset` that looks up an API-managed toolset by `bucket` and `path`.
- Add data source `aidial_file_toolset` that reads a static-config toolset by `name` only (`/v1/admin/config/file/toolsets/{name}`).
- Reuse the model's `features` and `interfaces` implementation: `ToolSet`, `Model` and `Application` reference the same OpenAPI `Features` schema, which matches the provider's existing `ModelFeatures` field for field, and `interfaces` is the same `DeploymentInterface` map.
- `auth_settings` exposes configuration only. `client_secret` is sensitive and kept from prior state, because GET returns only a read-only hint.
- **Identity exception**: for toolsets `bucket` is any non-empty string, not only `platform`, because Core does not restrict it for this type (for example `public`). `path` is non-empty, both force replacement, `id` is `<bucket>/<path>`, import is `<bucket>/<path>`. The other resources keep their `platform`-only validation.
- Add examples, regenerated `docs/`, README and `CHANGELOG.md` updates, and an integration manifest.

## Capabilities

### New Capabilities

- `toolset-resource`: client methods, the `aidial_toolset` resource and the API-managed data source of the same name.
- `file-toolset-data-source`: the `aidial_file_toolset` data source for static-config toolsets.

### Modified Capabilities

- `resource-identity`: documents toolsets as an exception to the `platform`-only `bucket` rule.
- `sensitive-attributes`: `auth_settings.client_secret` is sensitive wherever it appears.
- `provider-identity`: the registered resource and data source type list gains the three new types.
- `core-client-requests`: the percent-encoding requirement also covers toolsets (`bucket`/`path`) and file-sourced toolset `name`.

## Impact

- Code: new `internal/provider/toolset_resource.go`, `toolset_data_source.go`, `file_toolset_data_source.go`; new toolset types and methods in `internal/client/client.go`; registration in `internal/provider/provider.go`; shared extraction of the model `features` and `interfaces` schema, expand and flatten code so both resources use it.
- Tests: client and provider unit tests, an acceptance test (`TestAcc*`, skipped without `TF_ACC`), a manifest in `tests/integration/`.
- Docs: generated `docs/` pages, `examples/` snippets, README, `CHANGELOG.md` under `Unreleased`.
- Not breaking: no change to existing resources, state or schemas. `stub/models/` is untouched.
- Out of scope: the locale-map form of `display_name` and `intro` (plain strings only, as on `aidial_model`), the read-only runtime endpoints (`/openai/toolsets`, `/v1/toolset/{name}/...`, `/v1/ops/toolset/signin`), `createdAt`/`updatedAt`, the OAuth runtime status fields and `code_verifier`, relaxing `bucket` validation on other resources, and a toolset listing data source.
