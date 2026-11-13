# Proposal

## Why

DIAL Core documents translators (`/v1/translators/{bucket}/{path}`, schema `Translator`): services that convert a request from one interface (for example `openaiChatCompletions`) to another (for example `anthropicMessages`) so a deployment can serve a protocol it does not speak natively. The provider manages models, routes, roles, keys, schemas, toolsets, interceptors, applications and aliases, but it cannot create, read or look up translators, nor read those defined in Core's static configuration. Users must manage them out of band.

## What Changes

- Add resource `aidial_translator` (create, read, update, delete, import) over `/v1/translators/{bucket}/{path}`, with three attributes mapped from the OpenAPI `Translator` schema: `base_url`, `in` and `out`. All three are required (Core answers 400 when any is missing), and `in` and `out` accept only the `InterfaceType` values `openaiChatCompletions`, `openaiEmbeddings`, `openaiResponses` and `anthropicMessages`.
- Add data source `aidial_translator` that looks up an API-managed translator by `bucket` and `path`.
- Add data source `aidial_file_translator` that reads a static-config translator by `name` only (`/v1/admin/config/file/translators/{name}`).
- Identity follows the interceptor rule: `bucket` and `path` are required with no defaults, `bucket` accepts only `platform` (confirmed against Core 0.50.1, which answers `public` with 404), `path` is non-empty, both force replacement, `id` is `<bucket>/<path>`, import is `<bucket>/<path>`.
- Add examples, regenerated `docs/`, README and `CHANGELOG.md` updates, and an integration manifest.

## Capabilities

### New Capabilities

- `translator-resource`: client methods, the `aidial_translator` resource and the API-managed data source of the same name.
- `file-translator-data-source`: the `aidial_file_translator` data source for static-config translators.

### Modified Capabilities

- `resource-identity`: the `platform`-only `bucket` rule also covers `aidial_translator` and its data source.
- `provider-identity`: the registered resource and data source type list gains the three new types.
- `core-client-requests`: the percent-encoding rule also covers translator `bucket`/`path` and file-sourced translator `name`.

## Impact

- Code: new `internal/client/translator.go`; new `internal/provider/translator_resource.go`, `translator_data_source.go`, `file_translator_data_source.go`; registration in `internal/provider/provider.go`. The `modelInterfaceTypes` list is reused for validation.
- Tests: client and provider unit tests, an acceptance test (`TestAcc*`, skipped without `TF_ACC`), a manifest in `tests/integration/`.
- Docs: generated `docs/` pages, `examples/` snippets, README, `CHANGELOG.md` under `Unreleased`, and the `AGENTS.md` architecture and identity bullets.
- Not breaking: no change to existing resources, state or schemas. `stub/models/` is untouched.
- Out of scope: referencing translators from the `translator` entry (`TranslatorRef`: `name`, `inline`, `definition`) of a model or toolset interface, which stays "not modeled" and is a follow-up; `name`/`api_name` attributes (the `Translator` schema has no name field, and the name is the `path`); the read-only runtime endpoints (`/v1/metadata/translators/...`); and a translator listing data source.
