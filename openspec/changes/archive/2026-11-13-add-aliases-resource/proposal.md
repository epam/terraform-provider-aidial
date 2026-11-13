# Proposal

## Why

DIAL Core documents aliases (`/v1/aliases/{bucket}/{path}`, schema `Alias`): an alternative invocation name that points at a model, application or toolset and can override how it is presented and who may use it. The provider manages models, applications, toolsets, interceptors, routes, roles, keys and schemas, but it cannot create, read or look up aliases, nor read those defined in Core's static configuration. Users must manage them out of band.

## What Changes

- Add resource `aidial_alias` (create, read, update, delete, import) over `/v1/aliases/{bucket}/{path}`, with typed attributes mapped from the OpenAPI `Alias` schema: `present_as` (required, `model`, `application` or `toolset`), `target_path` (required, `models/<name>`, `applications/<name>` or `toolsets/<name>`), `user_roles`, `display_name`, `description`, `display_version`, `icon_url`, `intro` and `description_keywords`.
- Add data source `aidial_alias` that looks up an API-managed alias by `bucket` and `path`.
- Add data source `aidial_file_alias` that reads a static-config alias by `name` only (`/v1/admin/config/file/aliases/{name}`).
- Identity follows what a real Core accepts (the OpenAPI note that `bucket` "must be public" is not enforced): `bucket` and `path` are required with no defaults, `bucket` accepts `public` and `platform`, both force replacement, `id` is `<bucket>/<path>`, import is `<bucket>/<path>`.
- `name` is optional and computed with `api_name` holding Core's answer, as in the `make-name-optional` change.
- Add examples, regenerated `docs/`, README and `CHANGELOG.md` updates, and an integration manifest.

## Capabilities

### New Capabilities

- `alias-resource`: client methods, the `aidial_alias` resource and the API-managed data source of the same name.
- `file-alias-data-source`: the `aidial_file_alias` data source for static-config aliases.

### Modified Capabilities

- `resource-identity`: adds the `public`-or-`platform` `bucket` rule for `aidial_alias` and its data source.
- `provider-identity`: the registered resource and data source type list gains the three new types.
- `core-client-requests`: the percent-encoding rule also covers alias `bucket`/`path` and file-sourced alias `name`.

## Impact

- Code: new `internal/client/alias.go`; new `internal/provider/alias_resource.go`, `alias_data_source.go`, `file_alias_data_source.go`; registration in `internal/provider/provider.go`.
- Tests: client and provider unit tests, an acceptance test (`TestAcc*`, skipped without `TF_ACC`), a manifest in `tests/integration/`.
- Docs: generated `docs/` pages, `examples/` snippets, README, `CHANGELOG.md` under `Unreleased`, and the `AGENTS.md` architecture and identity bullets.
- Not breaking: no change to existing resources, state or schemas. `stub/models/` is untouched.
- Out of scope: every other `Alias` field (`author`, `createdAt`, `updatedAt`, `status`, and the fields Core rejects on a stored alias: `endpoint`, `features`, `catalogProperties`, `catalogSchemaId`), `target.id` and `target.version` (Core rejects them), the locale-map form of `display_name` and `description`, the `status` and `validationWarnings` read-only fields, the `/v1/metadata/aliases` endpoint, and an alias listing data source.
