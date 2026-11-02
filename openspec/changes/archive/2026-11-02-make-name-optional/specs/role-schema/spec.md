# Spec Delta

## MODIFIED Requirements

### Requirement: Role Terraform resource

The provider SHALL expose a resource named `<provider>_role` (e.g. `aidial_role`) implemented in `role_resource.go`.

The schema SHALL include:

- `id` (computed)
- `bucket` (required, must be `platform`, replace on change; see `resource-identity`)
- `path` (required, non-empty, replace on change; see `resource-identity`)
- `name` (optional, computed; role name sent to the API when configured, otherwise derived by DIAL Core from `path` and read back; see Name handling below)
- `api_name` (computed, canonical name returned by the API)
- `description`, `display_name` (optional, computed)
- `limits` (optional, map of limit names to JSON objects encoded as strings; DIAL Core models each value as a `Limit` object, but the provider treats each value as opaque JSON so it also accepts non-`Limit` shapes such as `{"enabled": true}` per-model toggles observed in production)
- `share` (optional, map of share names to JSON objects encoded as strings, per DIAL Core `ShareRule`)
- `cost_limit` (optional, computed, single nested object with optional, computed float attributes `day`, `minute`, `month`, `week`; sent to the API as `costLimit`)
- `topics` (optional, computed, set of strings)
- `granted_keys` (computed, set of strings; API keys granted this role, managed outside this resource)
- `created_at`, `updated_at` (computed)
- `status` (computed, when returned by API)

`id` SHALL be composed as `<bucket>/<path>`.

#### Scenario: Resource registration
- **WHEN** provider resources are registered
- **THEN** `aidial_role` is included in the resource list

#### Scenario: Immutable identity attributes
- **WHEN** `bucket` or `path` changes
- **THEN** Terraform plans replacement

#### Scenario: Stable resource ID
- **WHEN** role exists remotely
- **THEN** `id` is persisted as `<bucket>/<path>` across refreshes

#### Scenario: API canonicalizes the configured name
- **WHEN** the API returns a `name` value that differs from the configured `name` (e.g. DIAL Core strips a `roles/<bucket>/` prefix)
- **THEN** the resource preserves the user-configured `name` in state and surfaces the API's value only via `api_name`

#### Scenario: Identity attributes required
- **WHEN** `bucket` or `path` is omitted from configuration
- **THEN** validation fails at plan time

Name handling: DIAL Core addresses a role only by `bucket` and `path` and ignores the `name` in the request body, so `name` carries no identity. When `name` is configured the provider sends it and keeps the configured value in state; when omitted the provider sends none, and `name` is taken from the API response on create, then kept stable across plans.

#### Scenario: Name omitted
- **WHEN** a config sets `bucket` and `path` but not `name`
- **THEN** the role is created, no `name` is sent, and `name` in state equals `api_name`

#### Scenario: Name configured
- **WHEN** a config sets `name`
- **THEN** `name` is sent and the configured value is kept in state, whatever the API returns

#### Scenario: Name added or removed later
- **WHEN** `name` is added to, changed in or removed from a config with `bucket` and `path` unchanged
- **THEN** the plan updates in place or shows no change, and never plans replacement
