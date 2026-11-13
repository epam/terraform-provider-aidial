# Proposal

## Why

DIAL Core documents applications (`/v1/applications/{bucket}/{path}`, schema `Application`): deployments that expose a chat endpoint, optionally typed by an application type schema and referencing interceptors and routes. The provider manages models, routes, roles, keys, schemas, toolsets and interceptors, and already manages the `aidial_application_type_schema` that applications point at, but it cannot create, read or look up the applications themselves, nor read those defined in Core's static configuration. Users must manage them out of band.

## What Changes

- Add resource `aidial_application` (create, read, update, delete, import) over `/v1/applications/{bucket}/{path}`, with typed attributes mapped from the OpenAPI `Application` schema: `endpoint` (optional; exactly one of it and `application_type_schema_id`), `description`, `display_name`, `display_version`, `icon_url`, `forward_auth_token`, `max_retry_attempts`, `user_roles`, `interceptors`, `features`, `interfaces`, `defaults`, `default_headers`, `application_type_schema_id`, `application_properties`, `catalog_properties`, `catalog_schema_id`, `editor_url`, `viewer_url`. Free-form objects are dynamic attributes with the same semantic-equality behavior as `aidial_model`.
- Add data source `aidial_application` that looks up an API-managed application by `bucket` and `path`.
- Add data source `aidial_file_application` that reads a static-config application by `name` only (`/v1/admin/config/file/applications/{name}`).
- Reuse the shared `features` and `interfaces` helpers from `deployment_shared.go`.
- Identity follows the toolset rule: `bucket` and `path` required, no defaults, `bucket` any non-empty string (Core accepts `platform` and `public`), `path` may contain `/`, both force replacement, `id` is `<bucket>/<path>`, import is `<bucket>/<path>`.
- Add examples, regenerated `docs/`, README and `CHANGELOG.md` updates, and an integration manifest.

## Capabilities

### New Capabilities

- `application-resource`: client methods, the `aidial_application` resource and the API-managed data source of the same name.
- `file-application-data-source`: the `aidial_file_application` data source for static-config applications.

### Modified Capabilities

- `resource-identity`: the toolset `bucket`/`path` rule also covers `aidial_application` and its data source.
- `provider-identity`: the registered type list gains the three new types.
- `core-client-requests`: application `path` keeps its `/` separators, like toolsets.

## Impact

- Code: new `internal/client/application.go`; new `internal/provider/application_resource.go`, `application_data_source.go`, `file_application_data_source.go`; registration in `internal/provider/provider.go`.
- Tests: client and provider unit tests, an acceptance test (skipped without `TF_ACC`), a manifest in `tests/integration/`.
- Docs: generated `docs/`, `examples/`, README, `CHANGELOG.md` under `Unreleased`, `AGENTS.md` architecture and identity bullets.
- Not breaking: no change to existing resources or state.
- Out of scope: the nested `function` (code-based apps and their deploy/undeploy/logs ops), `mcp`, `routes`, `external_services`, `allow_user_external_services`, `app_identity`, `invalid`, the fields `author`, `createdAt`, `updatedAt`, `dependencies`, `descriptionKeywords`, `inputAttachmentTypes`, `maxInputAttachments`, `reference`, `responsesDefaults`, `responsesEndpoint`, `overrideName`, `baseUrl`, `intro`, the locale-map form of `display_name`, the `/v1/applications/{appId}/external-services` endpoints, runtime `/openai/applications` endpoints, and an application listing data source.
