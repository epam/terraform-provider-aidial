# Design

## Context

See proposal.md for motivation. Observed state:

- `aidial_interceptor` is the closest template: `internal/client/interceptor.go` uses the generic `getResource`, `saveResource` and `deleteResource` helpers with percent-encoded single segments, and the provider side has a resource, an API data source and a file data source.
- `Translator` in `openapicore/open_api_core.yaml` has exactly three properties: `baseUrl` (string), `in` and `out` (`InterfaceType`, an enum of `openaiChatCompletions`, `openaiEmbeddings`, `openaiResponses`, `anthropicMessages`). It has no `required` list and no name.
- Endpoints: `GET/PUT/DELETE /v1/translators/{bucket}/{path}`, the read-only `GET /v1/admin/config/file/translators` (list) and `GET /v1/admin/config/file/translators/{name}`.
- `modelInterfaceTypes` in `model_resource.go` already holds the four enum values and is used to validate `interfaces` keys.
- `TranslatorRef` (a `translator` entry with `name`, `inline`, `definition`) appears inside deployment interfaces. `deployment_shared.go` documents it as not modeled; this change leaves it so.

## Goals / Non-Goals

**Goals:**
- A translator behaves like the other API-managed resources: same identity, import, read-after-write, 404 and null-vs-empty rules.
- Validate `in` and `out` at plan time, so a typo fails before the apply.

**Non-Goals:**
- Linking translators to models or toolsets, or checking that a referenced name exists.
- Exposing `name` or `api_name`: the schema has no name field and the translator's name is its `path`. This differs from the interceptor, which has an optional `name` and a computed `api_name`.

## Decisions

1. **Generic endpoint `/v1/translators/{bucket}/{path}`** with the generic client helpers and percent-encoded single segments, as for interceptors.
2. **Bucket is `platform`-only**, confirmed by the real-Core probe below.
3. **Wire format is camelCase** (`baseUrl`, `in`, `out`), as in the OpenAPI document and confirmed against Core. Terraform attributes are `base_url`, `in` and `out`; `in` is an HCL keyword but a legal attribute name.
4. **All three attributes are required.** Core rejects a body without them, so making them optional would only move the error from plan to apply. `in` and `out` use `stringvalidator.OneOf(modelInterfaceTypes...)`.
5. **No computed or defaulted attributes.** Core fills in nothing the provider exposes, so state equals config after apply and there is no null-vs-empty question for this resource. The spec still states the rule for consistency.
6. **GET extras are ignored.** Core returns `name` and `status` next to the three fields; the client decodes only the three.
7. **File data source** mirrors `aidial_file_interceptor`: lookup by `name` only, the same attributes minus identity, error on 404.
8. **Spec placement.** New capabilities `translator-resource` (client, resource, API data source) and `file-translator-data-source`, mirroring the interceptor layout. The identity, provider-identity and encoding changes are ADDED requirements in the existing capabilities.

## Findings from a real Core (Core 0.50.1)

Probed against the local ephemeral Core with throwaway translators, now deleted.

- **Wire format is camelCase.** A PUT of `{"baseUrl","in","out"}` returned `200 {"name": ...}`; GET returned the three fields plus `name` and `status` (`valid`).
- **Only `platform` works.** `PUT /v1/translators/public/x` is a 404, and so is the GET.
- **All three fields are required.** `{}` is a 400 (`Missing required creator property 'out'`), and nothing is stored.
- **`in` and `out` are enum-checked.** An unknown value is a 400 listing `openaiResponses`, `openaiChatCompletions`, `openaiEmbeddings`, `anthropicMessages`.
- **Slashes in `path` are not supported.** `platform/dir/x` is a 400 (`Invalid entity name segment: must match ^[A-Za-z0-9._%:@\[\]()-]+$`). The client keeps percent-encoding single segments and the error surfaces at apply.
- **DELETE of a missing translator is a 404**, handled as success.
- **No static translator** is defined: `GET /v1/admin/config/file/translators` returns `{"items":[]}`, so `aidial_file_translator` has no acceptance test.

## Risks / Trade-offs

- **`in` as an attribute name** can surprise users who expect it to be reserved. HCL accepts it (`in = "openaiChatCompletions"`); the docs and examples show it. Mitigation: a unit test that plans and applies a config using `in`.
- **A file-sourced translator may not exist on the ephemeral Core** → unit tests cover the client and the data source; no acceptance test for `aidial_file_translator` unless the Core ships one.
- **Schema drift against newer Core** (`Translator` gains fields) → unknown JSON fields are ignored on read and the next OpenAPI refresh adds attributes. A PUT replaces the whole document, so a value set out of band in a field the provider does not expose is lost on the next apply; the README says so, as for interceptors.
