# Design

## Context

See proposal.md for motivation. Observed state:

- `aidial_model` is the closest template. Its `features` and `interfaces` schema, plan-to-API conversion and state mapping are written inline in `model_resource.go` (`modelFromPlan`, `setModelState`) rather than as reusable functions. Free-form objects use `model_defaults.go` (`freeFormObjectFromPlan` / `freeFormObjectToState`). Write-only secrets are preserved from prior state (`aidial_key`, upstream `key` and `secret_extra_data`).
- In `openapicore/open_api_core.yaml`, `ToolSet`, `Model`, `Application` and `Alias` all reference the same `Features` schema, whose 24 fields equal the client's `ModelFeatures`. `ToolSet.interfaces` and `Model.interfaces` both use `DeploymentInterface`.
- `ToolSet` has no `required` list at all, and the spec does not say which buckets Core accepts for `/v1/toolsets/{bucket}/{toolset_path}`. The other resources are `platform`-only, an observed Core behavior recorded in `resource-identity`.
- The file-sourced read is `GET /v1/admin/config/file/toolsets/{name}`, the analogue of the file model and file schema endpoints.

## Goals / Non-Goals

**Goals:**
- A toolset behaves like the other API-managed resources: same identity, import, read-after-write, 404 and null-vs-empty rules.
- One implementation of `features` and `interfaces` shared by models and toolsets, with no change to model behavior.

**Non-Goals:**
- Any change to the model resource's schema, state or plan behavior. The extraction is a pure refactor.
- Per-interface `translator`, `defaults` and nested `features`, which `aidial_model` also does not implement.
- Runtime and OAuth session endpoints.

## Decisions

1. **Generic endpoint `/v1/toolsets/{bucket}/{path}`.** The separate `/v1/toolsets/platform/{path}` variant would only work for one bucket, and `bucket` is a free string here. Same `getResource`, `saveResource` and `deleteResource` helpers as routes, `bucket` and `path` percent-encoded as single segments like every other type.
2. **Bucket validation is "non-empty string".** A hard-coded list (`platform`, `public`) would go stale and cannot be verified from the OpenAPI document; Core's own 4xx surfaces at apply. Alternative considered: `OneOf(platform, public)`. Rejected for that reason. `bucket` and `path` still force replacement. The identity validators are declared inline per schema (there is no shared helper), so toolset schemas just declare a non-empty validator. The other resources are not touched.
3. **Extract, then reuse, the model's `features` and `interfaces` code.** Move the schema attribute builders, expand (plan to client struct) and flatten (client struct to state, with null normalization) for those two attributes into shared helpers, and call them from both resources. Alternative: copy the code. Rejected, because the two would drift on every Core feature addition, and the shapes are identical in the OpenAPI document. A model-resource regression test run before and after proves the refactor is behavior-neutral.
4. **`auth_settings` is configuration only.** Omitted: `app_level_auth_status`, `global_auth_status`, `user_level_auth_status` (runtime session state that would show up as drift), `code_verifier` (transient PKCE value) and `client_secret_hint` (read-only). `client_secret` is `Sensitive`, optional and not computed. Read-back keeps the prior value, like `aidial_key.key`; after import it is null. The data sources omit it, since Core never returns it.
5. **Typed attributes, with free-form JSON only where the schema is free-form.** `defaults`, `responses_defaults` and `catalog_properties` are `types.Dynamic` via the existing `model_defaults.go` functions. Everything else, including `allowed_tools` and the other string lists (sets, like on models), is typed.
6. **Names.** `name` is optional and computed and sent only when configured, with `api_name` holding Core's answer, as in the `make-name-optional` change. `display_name` and `intro` are plain strings; sending a string where Core stores a locale map is the same limitation `aidial_model` has.
7. **`provider` is exposed as `provider_name`.** The framework rejects `provider` as a top-level attribute name (Terraform reserves it as a resource meta-argument), so the API's `provider` field cannot keep its name. No other `ToolSet` field collides.
8. **Optional, computed where Core defaults.** Plain scalars and sets are optional only, as on most `aidial_model` attributes. Attributes Core fills in on its own are optional and computed (see the findings), because an echoed default would otherwise be an "inconsistent result" error. `name`, `api_name`, `interfaces` and `default_headers` are computed, like on models.
9. **Omitted server fields.** `createdAt` and `updatedAt` change on every write, so exposing them would add plan noise for no use. Not modelled on the resource or data sources.
10. **File data source** mirrors `aidial_file_model`: lookup by `name` only, the same attribute set minus identity and secrets, error on 404.
11. **Spec placement.** New capabilities `toolset-resource` (client, resource, API data source) and `file-toolset-data-source`, following the layout used for application type schemas. The identity exception is an ADDED requirement in `resource-identity`, because the existing "Required bucket" requirement lists the covered resources and does not include toolsets.

## Findings from a real Core (task 5.1, Core 0.50.1)

Probed against the ephemeral Core with throwaway toolsets, now deleted. The OpenAPI document is wrong or silent on all of these:

- **Wire format is snake_case.** GET returns `forward_auth_token`, `auth_settings`, `description_keywords`, `responses_defaults` and so on, and PUT accepts them. camelCase bodies are rejected (`Missing required property 'responsesEndpoint'`). The client therefore uses snake_case tags. Nested `interfaces` entries keep the model's mixed keys (`base_url`, `overridePaths`, `defaultHeaders`), so reusing `ModelInterface` was right.
- **Paths keep their slashes.** `PUT /v1/toolsets/public/team/tool` works; `team%2Ftool` is a 400. Toolset URLs escape each segment and keep the `/` separators (`core-client-requests` delta).
- **Any bucket string reaches Core.** `public` works for PUT, GET and DELETE, and `platform` works too. The `GET` name differs by bucket (`toolsets/public/x` against `x` for `platform`) and is only kept in `api_name`.
- **Core fills in defaults:** `reference` (a new UUID on every PUT that omits it), `author` (the caller, whatever is sent), `transport` HTTP, `forward_auth_token` and `forward_per_request_key` false, `max_retry_attempts` 1, empty `allowed_tools`, `dependencies`, `description_keywords` and `interceptors`, empty `defaults` and `responses_defaults`, and `auth_settings` of type `NONE` with status fields. These are optional and computed with `UseStateForUnknown` (decision 8 is revised accordingly), and `author` is computed only. Free-form `defaults` stay optional: `{}` reads back as null.
- **Minimum payload:** `{}` is accepted. `endpoint` is not required, but Core checks reachability of an `endpoint` that is set (`Connection failed ... invalid or unreachable`) and validates OAuth settings (`authorization_endpoint` required; `client_secret` at least 8 characters). These surface as apply errors. For the `platform` bucket, and for incomplete OAuth settings in any bucket, Core also probes the endpoints when saving (`Connection failed`); a complete OAuth block (`redirect_uri`, `resource`, `scopes_supported`, `code_challenge_method`, `dynamically_registered`, `token_endpoint_auth_method`) in the `public` bucket skips the probe, which is what the tests use.
- **`code_challenge` is derived by Core** and overwrites a sent value, so it is not modelled.
- **`client_secret` is write-only:** GET returns only `client_secret_hint`, as designed.
- **Not verified:** a file-sourced toolset (the ephemeral Core defines none, `GET /v1/admin/config/file/toolsets` returns an empty list), and its JSON key style (assumed snake_case like resource-storage toolsets).

## Risks / Trade-offs

- **Core's minimum payload is unknown** (`ToolSet` has no required fields) → during implementation, PUT `{}`-like bodies and a typical MCP toolset (`endpoint`, `transport`) against the ephemeral Core, then make attributes required in the schema only if Core demands them, and record the result in the spec.
- **`public` bucket may not accept writes with every credential** → it works with the ephemeral Core's key (see the findings), so the integration manifest includes a `public` case; other credentials may differ.
- **The OpenAPI contract and real Core disagree on toolsets** → the client follows observed behavior; the next OpenAPI refresh may change the document, and the integration test against a real Core is what keeps the client honest.
- **File-sourced toolsets are untested against a real Core** → unit tests cover the client and data source; an acceptance test for `aidial_file_toolset` needs a Core whose static config defines a toolset.
- **Slashes in `path`** → resolved, see the findings: toolsets need raw slashes.
- **Extraction touches `aidial_model`** → keep it mechanical, run the full existing model unit, null-vs-empty and drift tests before adding toolset code.
- **`client_secret` may be returned masked by some Core versions** → treat any GET value as non-authoritative: always keep prior state; revisit if Core ever returns it in clear.
- **Schema drift against newer Core** (`ToolSet` gains fields) → unknown JSON fields are ignored on read; the next OpenAPI refresh adds attributes.
