# Spec Delta

## MODIFIED Requirements

### Requirement: Route Terraform resource

The provider SHALL expose a resource named `<provider>_route` (e.g. `aidial_route`) implemented in `route_resource.go`, matching `role_resource.go` conventions.

The schema SHALL include:

- `id` (computed)
- `bucket` (required, must be `platform`, replace on change; see `resource-identity`)
- `path` (required, non-empty, replace on change; see `resource-identity`)
- `name` (required, route name sent to the API)
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
- `upstreams` (optional, computed, list of nested objects: `endpoint` (required), `responses_endpoint`, `id`, `extra_data` (map of string to string), `key`, `secret_extra_data`, `tier` (required), `weight` (required); reuses DIAL Core's `Upstream` schema, same Go types as `aidial_model`'s `upstreams`, but with different required-ness: route's `endpoint`/`tier`/`weight` are `Required`, whereas `aidial_model`'s are `Optional`+`Computed`); `key` and `secret_extra_data` are optional, sensitive and not computed: DIAL Core never returns them (`writeOnly`), so state keeps the configured values by list position (an upstream without `key` plans as unset, not "(known after apply)"); after import they are null; `extra_data` is decoded from either an object (DIAL Core 0.48.1) or a string holding a JSON object (newer Core, matching the OpenAPI `string` type); it is always sent as an object, and non-string values are kept as their JSON text.
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
