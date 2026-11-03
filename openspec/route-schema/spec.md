# route-schema Specification

## Purpose

Define the Terraform resource shape for `aidial_route`, implemented in `route_resource.go`.

## Requirements

### Requirement: Route Terraform resource

The provider SHALL expose a resource named `<provider>_route` (e.g. `aidial_route`) implemented in `route_resource.go`, matching `role_resource.go` conventions.

The schema SHALL include:

- `id` (computed)
- `bucket` (required, must be `platform`, replace on change; see `resource-identity`)
- `path` (required, non-empty, replace on change; see `resource-identity`)
- `name` (optional, computed; route name sent to the API when configured, otherwise derived by DIAL Core from `path` and read back; see Name handling below)
- `api_name` (computed, canonical name returned by the API)
- `order` (optional, computed, int)
- `methods` (optional, computed, set of strings)
- `paths` (optional, computed, set of strings; DIAL Core models each entry as an untyped `Pattern` object, represented here as a raw pattern string)
- `permissions` (optional, computed, set of strings, per DIAL Core `ResourceAccessType`: `READ`, `WRITE`, `SHARE`)
- `user_roles` (optional, computed, set of strings)
- `max_retry_attempts` (optional, computed, int)
- `rewrite_path` (optional, computed, bool)
- `response` (optional, computed, nested object: `body`, `status`, per DIAL Core `RouteResponse`)
- `attachment_paths` (optional, computed, nested object: `request_body`, `response_body` sets of strings, per DIAL Core `RouteAttachmentPath`)
- `upstreams` (optional, computed, list of nested objects: `endpoint` (required), `responses_endpoint`, `id`, `extra_data` (map of string to string), `key`, `secret_extra_data`, `tier` (required), `weight` (required), `base_url`, `auth_type` (`API_KEY` or `BEARER`; both optional and computed); reuses DIAL Core's `Upstream` schema, same Go types as `aidial_model`'s `upstreams`, but with different required-ness: route's `endpoint`/`tier`/`weight` are `Required`, whereas `aidial_model`'s are `Optional`+`Computed`); `key` and `secret_extra_data` are optional, sensitive and not computed: DIAL Core never returns them (`writeOnly`), so state keeps the configured values by list position (an upstream without `key` plans as unset, not "(known after apply)"); after import they are null; `extra_data` is decoded from either an object (DIAL Core 0.48.1) or a string holding a JSON object (newer Core, matching the OpenAPI `string` type); it is always sent as an object, and non-string values are kept as their JSON text.
- `status` (computed, when returned by API)

`id` SHALL be composed as `<bucket>/<path>`.

#### Scenario: Static response route
- **WHEN** a route configures `response` instead of `upstreams`
- **THEN** the schema accepts a route with no upstreams and a populated `response` object

#### Scenario: API canonicalizes the configured name
- **WHEN** the API returns a `name` value that differs from the configured `name`
- **THEN** the resource preserves the user-configured `name` in state and surfaces the API's value only via `api_name`

#### Scenario: Resource registration
- **WHEN** provider resources are registered
- **THEN** `aidial_route` is included in the resource list

#### Scenario: Immutable identity attributes
- **WHEN** `bucket` or `path` changes
- **THEN** Terraform plans replacement

#### Scenario: Stable resource ID
- **WHEN** route exists remotely
- **THEN** `id` is persisted as `<bucket>/<path>` across refreshes

#### Scenario: Identity attributes required
- **WHEN** `bucket` or `path` is omitted from configuration
- **THEN** validation fails at plan time

Name handling: DIAL Core addresses a route only by `bucket` and `path` and ignores the `name` in the request body, so `name` carries no identity. When `name` is configured the provider sends it and keeps the configured value in state; when omitted the provider sends none, and `name` is taken from the API response on create, then kept stable across plans.

#### Scenario: Name omitted
- **WHEN** a config sets `bucket` and `path` but not `name`
- **THEN** the route is created, no `name` is sent, and `name` in state equals `api_name`

#### Scenario: Name configured
- **WHEN** a config sets `name`
- **THEN** `name` is sent and the configured value is kept in state, whatever the API returns

#### Scenario: Name added or removed later
- **WHEN** `name` is added to, changed in or removed from a config with `bucket` and `path` unchanged
- **THEN** the plan updates in place or shows no change, and never plans replacement

## Implementation Notes

- `name` is optional and computed (`UseStateForUnknown`). A known `name` is never overwritten from the API response in `setRouteState`; when it is not configured it is set from the response once, on create or import. Only `api_name` (`Computed`-only) reflects what the API actually returned. This mirrors `aidial_role`'s and `aidial_model`'s `name`/`api_name` split, added after both hit a Terraform provider-consistency failure (`.name: was cty.StringVal(...), but now cty.StringVal(...)`) when the API echoed back a canonicalized name; `aidial_route` carried the same latent bug (unconditional overwrite, no `api_name`) until it was caught by a unit test exercising this exact scenario.
- Nested attributes (`response`, `attachment_paths`, `upstreams`) are represented as Terraform object/list types built and unpacked via `types.ObjectValue`/`types.ListValue` helpers, mirroring `model_resource.go`.
- `upstreams` reuses the same `modelUpstreamModel`/`client.ModelUpstream` types as `aidial_model`, since DIAL Core's `Upstream` schema is shared between `Model` and `Route` — but reusing the Go types does not mean the schemas are identical: route's `endpoint`/`tier`/`weight` are `Required`, while `aidial_model`'s are `Optional`+`Computed`. `extra_data` is a `map(string)` in both (matching `modelUpstreamModel.ExtraData`'s `types.Map`), corrected from a stray `schema.StringAttribute` that briefly diverged from the underlying Go type after `aidial_model`'s upstream schema changed `extra_data` from a string to a map — route's schema wasn't updated to match at the time, which would have rejected any non-empty `extra_data` config at plan time.
- Schema verified against the DIAL Core OpenAPI spec (`openapicore/open_api_core.yaml`, `Route`, `RouteResponse`, `RouteAttachmentPath`, `ResourceAccessType`, `Pattern` schemas). Current known gaps in `route_resource.go` versus that contract:
  - `paths[]` is modeled as a set of plain strings; DIAL Core's `Pattern` schema is an untyped object with no declared properties, so the API's actual pattern-matching structure (if any beyond a string) is not represented.
  - `upstreams[].interfaces` is not implemented, matching the same gap already tracked for `aidial_model`. `upstreams[].base_url` and `upstreams[].auth_type` are implemented.
