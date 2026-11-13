# Design

## Context

See proposal.md. Observed state:

- `aidial_interceptor` (archived change `add-interceptor-resource`) is the template: same identity model, `deployment_shared.go` helpers, `model_defaults.go` free-form handling, client in `internal/client/interceptor.go`.
- `Application` in `openapicore/open_api_core.yaml` is camelCase for most fields but mixes snake_case (`application_properties`, `application_type_schema_id`, `editor_url`, `viewer_url`, `external_services`, `interfaces`). The document is known to be wrong in places (toolsets), so casing is verified against Core.
- Endpoints: `GET/PUT` on `/v1/applications/{bucket}/{application_path}`, `GET/PUT/DELETE` on `/v1/applications/platform/{path}` (marked `x-preview`), and read-only `GET /v1/admin/config/file/applications/{name}`. The generic path has no documented DELETE, which must be checked.
- `aidial_application_type_schema` already exists; `application_type_schema_id` references its `$id`. No Terraform-level reference is added.

## Goals / Non-Goals

**Goals:**
- Applications behave like other API-managed resources: identity, import, read-after-write, 404 and null-vs-empty rules.
- Reuse shared helpers without changing model, toolset or interceptor behavior.

**Non-Goals:**
- Code-based applications (`function`), deployment lifecycle and logs, MCP, nested routes and external services.
- Validating that `interceptors` names or `application_type_schema_id` refer to existing resources.

## Decisions

1. **Endpoint** `/v1/applications/{bucket}/{path}` with the generic `getResource`/`saveResource`/`deleteResource` helpers (task 1.1: GET, PUT and DELETE all work there).
2. **Bucket follows the toolset rule** (any non-empty string, `path` keeps `/`), because Core accepts `public` (task 1.1).
3. **Wire format is snake_case**, as Core returns it (task 1.1), like toolsets.
4. **Attribute subset** is the list in proposal.md; unexposed fields are ignored on read. `endpoint` is optional: Core requires exactly one of `endpoint` and `application_type_schema_id` (or MCP or function, out of scope), so the resource validates that exactly one of the two is set.
5. **Names.** `name` optional and computed, `api_name` holds Core's answer as is, as for toolsets. For `public` Core returns `applications/public/<path>`; it is not shortened.
6. **Core-defaulted attributes** (`forward_auth_token`, `max_retry_attempts`, probably others) are optional and computed with `UseStateForUnknown`; the exact set comes from task 1.1.
7. **`interceptors`** is a set of names, same as on `aidial_model`.
8. **File data source** mirrors `aidial_file_interceptor`.
9. **Spec placement** mirrors interceptors: two new capabilities and ADDED requirements in three existing ones.

## Findings from a real Core (task 1.1, Core 0.50.1)

Probed against the local Core with throwaway applications and one schema, now deleted.

- **Both `/v1/applications/{bucket}/{path}` verbs work**: GET, PUT, DELETE.
- **Buckets `platform` and `public` work**; anything else is a 400 ("invalid bucket"). So the platform-only rule of interceptors does not apply.
- **Slashes**: `platform/team%2Fx` is a 400 (invalid entity name segment); `public/team/probe8` with a raw slash works and returns `parentPath: team`.
- **Casing**: GET returns snake_case (`display_name`, `max_retry_attempts`, `user_roles`, `default_headers`, `catalog_properties`, `application_type_schema_id`, `application_properties`, `editor_url`, `viewer_url`). Core accepts both snake_case and camelCase on write.
- **Minimum payload**: `{}` is a 400 ("At least application endpoint, MCP endpoint, function or interface must be provided"). An `endpoint` together with `application_type_schema_id` is a 400 ("Neither application endpoint, MCP or function must be set for schema based application"). `application_type_schema_id` plus `application_properties` is accepted. An unknown `catalog_schema_id` is a 400, so catalog attributes need a real catalog schema.
- **Core fills in**: `forward_auth_token` false, `max_retry_attempts` 1, `defaults` `{}`, `responses_defaults` `{}`, `routes` `{}`, `interceptors` `[]`, `description_keywords` `[]`, `dependencies` `[]`, `reference` (a UUID), and `status` `valid` on `platform`. `default_headers` is omitted when unset.
- **`name`**: `platform` returns the bare name and ignores a `name` in the body; `public` returns `applications/public/<path>` and also `author`, `created_at` and `updated_at`.
- **DELETE** of a missing `platform` application is a 404 (success); `public` DELETE answered 200.
- **No static application** is defined (`GET /v1/admin/config/file/applications` returns an empty list), so `aidial_file_application` has no acceptance test.

## Risks / Trade-offs

- **OpenAPI disagrees with real Core** → already probed (see Findings); the specs follow Core.
- **Dropped unexposed fields**: a PUT replaces the document, so out-of-band `function`, `mcp`, `routes` etc. are lost on the next apply → document in the README.
- **A static application may not exist on the ephemeral Core** → no acceptance test for the file data source unless one ships.
- **Schema drift against newer Core** → unknown JSON fields are ignored on read.
