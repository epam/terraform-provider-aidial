# Design

## Context

See proposal.md for motivation. Observed state:

- `aidial_toolset` is the closest template: a recent resource built on `deployment_shared.go` (`featuresResourceAttribute`, `featuresFromObject`, `featuresToObject`, `interfacesResourceAttribute`, `interfacesFromMap`, `interfacesToMap`) and `model_defaults.go` (`freeFormObjectFromPlan`, `freeFormObjectToState`). Its client lives in `internal/client/toolset.go`.
- In `openapicore/open_api_core.yaml`, `Interceptor` references the same `Features` schema as `Model`, `Application` and `ToolSet` (nested `interfaces` entries use `FeaturesData` through `DeploymentInterface`). The shared helpers therefore apply unchanged, which closes the condition attached to reusing them.
- `Interceptor` is camelCase in the OpenAPI document and has no `required` list. Endpoints: `GET/PUT/DELETE /v1/interceptors/{bucket}/{path}` and the read-only `GET /v1/admin/config/file/interceptors/{name}`.
- `aidial_model` already has an `interceptors` set of names; nothing here changes it.

## Goals / Non-Goals

**Goals:**
- An interceptor behaves like the other API-managed resources: same identity, import, read-after-write, 404 and null-vs-empty rules.
- Reuse the shared helpers with no change to model or toolset behavior.

**Non-Goals:**
- Validating that a model's `interceptors` names refer to existing interceptors, or adding a reference between the resources.
- The `Interceptor` fields listed as out of scope in proposal.md.
- Per-interface `translator`, `defaults` and nested `features`, which `aidial_model` also does not implement.

## Decisions

1. **Generic endpoint `/v1/interceptors/{bucket}/{path}`** with the generic `getResource`, `saveResource` and `deleteResource` helpers, `bucket` and `path` percent-encoded as single segments like roles and models. Revisit only if the real-Core check (task 1.1) shows Core needs raw slashes, as it did for toolsets.
2. **Bucket is `platform`-only.** The user believes Core does not accept `public` for interceptors; task 1.1 confirms it against a local Core. If Core does accept other buckets, switch to the toolset rule (non-empty string) and update the `resource-identity` delta before implementing.
3. **Wire format is camelCase**, as in the OpenAPI document and like models. The toolset change found the document wrong for toolsets (Core uses snake_case), so this is verified in task 1.1 before the client is written. If Core disagrees, the client follows Core and the `interceptor-resource` client requirement is updated.
4. **Attribute subset.** Only the fields in the proposal are exposed; everything else is ignored on read, so a future Core field or an out-of-band value in an unexposed field causes no drift. `endpoint` is required in the schema because an interceptor without one cannot do anything; if Core accepts an empty body, the requirement stays because it is the useful minimum.
5. **Names.** `name` is optional and computed and sent only when configured, with `api_name` holding Core's answer, as in the `make-name-optional` change.
6. **Omitted server fields.** `author`, `createdAt` and `updatedAt` are not modelled, because they change on writes or are set by the caller and would add plan noise.
7. **Core-defaulted attributes.** `forward_auth_token` and `max_retry_attempts` are optional and computed with `UseStateForUnknown`, as in the toolset change (see the findings). Free-form `defaults` stay optional.
8. **File data source** mirrors `aidial_file_toolset`: lookup by `name` only, the same attribute set minus identity, error on 404.
9. **Spec placement.** New capabilities `interceptor-resource` (client, resource, API data source) and `file-interceptor-data-source`, mirroring the toolset layout. The identity, provider-identity and encoding changes are ADDED requirements in the existing capabilities, so no existing requirement text is rewritten.

## Findings from a real Core (task 1.1, Core 0.50.1)

Probed against the local Core with throwaway interceptors, now deleted.

- **Wire format is camelCase**, as the OpenAPI document says (`displayName`, `forwardAuthToken`, `maxRetryAttempts`, `userRoles`, `defaultHeaders`, `catalogProperties`). Core also accepts snake_case aliases on write, but the client sends camelCase. Nested `features` keep their snake_case keys and `interfaces` entries use `base_url`, as on models, so `ModelFeatures` and `ModelInterface` fit.
- **Only `platform` works.** `PUT /v1/interceptors/public/x` is a 404, so decision 2 stands.
- **Slashes in `path` are not supported at all.** Both `team/x` and `team%2Fx` are a 400 (`Invalid entity name segment: must match ^[A-Za-z0-9._%:@\[\]()-]+$`). The client keeps percent-encoding single segments, and the error surfaces at apply.
- **`{}` is accepted** as a body, so `endpoint` is required by this provider, not by Core.
- **Core fills in:** `forwardAuthToken` false, `maxRetryAttempts` 1, `defaults` `{}`, `responsesDefaults` `{}`, `interceptors` `[]`, `descriptionKeywords` `[]`, `dependencies` `[]` and `status` `valid`. `author`, `createdAt`, `updatedAt`, `reference`, `features`, `catalogProperties` and `catalogSchemaId` read back as null. `defaultHeaders` is omitted when unset. Of the exposed attributes, `forward_auth_token` and `max_retry_attempts` are therefore optional and computed; `defaults` `{}` reads back as `{}` and stays optional like on models (null-vs-empty handled by `freeFormObjectToState`).
- **DELETE of a missing interceptor is a 404**, handled as success.
- **No static interceptor** is defined (`GET /v1/admin/config/file/interceptors` returns an empty list), so `aidial_file_interceptor` has no acceptance test.

## Risks / Trade-offs

- **The OpenAPI contract may disagree with real Core** (key casing, path slashes, accepted buckets, minimum payload) → task 1.1 probes a local Core first and records the findings in this file; the specs are adjusted before the client and resource are written.
- **A file-sourced interceptor may not exist on the ephemeral Core** (as with toolsets) → unit tests cover the client and the data source; no acceptance test for `aidial_file_interceptor` unless the Core ships one.
- **The ignored fields are silently dropped on update.** A PUT replaces the document, so an out-of-band value in an unexposed field (for example `dependencies`) is lost on the next apply → document it in the README, as the toolset change did for its omitted fields.
- **Schema drift against newer Core** (`Interceptor` gains fields) → unknown JSON fields are ignored on read; the next OpenAPI refresh adds attributes.
