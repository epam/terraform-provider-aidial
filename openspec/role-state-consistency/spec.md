# role-state-consistency Specification

## Purpose

Define state consistency and idempotency guarantees for the `aidial_role` resource.

## Requirements

### Requirement: State consistency and idempotency

Implementation SHALL avoid perpetual diffs, unnecessary updates, and Terraform provider-consistency errors.

It SHALL:

- re-read remote state after create/update
- never overwrite the configured, `Required` `name` attribute from an API response once it is known; expose the API's own value only through the `Computed`-only `api_name` attribute
- normalize map-like fields (`limits`, `share`) to a stable Terraform representation via JSON-string passthrough
- keep the configured string for a `limits`/`share` element when the API value decodes to the same JSON (whitespace, key order and number formatting do not matter)
- keep a configured empty `limits`/`share` map as `{}` and an unset one as null, whether the API returns `{}` or omits the field
- drop limit fields that DIAL Core filled with the int64-max "unlimited" sentinel when the configured value for that key did not set them
- normalize set-like fields (`topics`, `granted_keys`) to explicit null sets when empty
- decode `cost_limit` fields tolerantly regardless of whether the API encodes them as bare numbers or quoted numeric strings
- preserve unconfigured optional/computed attributes (`description`, `display_name`, `topics`) across updates via `UseStateForUnknown`, rather than clearing them when left out of config
- handle API omission of optional fields consistently
- preserve idempotent plan/apply behavior

#### Scenario: Repeated plan with no config changes
- **WHEN** no local or remote changes occurred
- **THEN** `terraform plan` returns no changes

#### Scenario: Repeated apply
- **WHEN** same configuration is applied twice
- **THEN** second apply is a no-op

#### Scenario: Remote drift
- **WHEN** role is modified outside Terraform
- **THEN** next refresh/plan reflects actionable drift

#### Scenario: Optional field omitted by API
- **WHEN** API omits optional empty fields (e.g. `costLimit`, `topics`, `grantedKeys`, `description`, `displayName`)
- **THEN** state normalization prevents repeated diff churn

#### Scenario: API canonicalizes the role name
- **WHEN** the API returns a `name` that differs from the configured value (e.g. DIAL Core strips a `roles/<bucket>/` prefix)
- **THEN** Terraform does not report a provider error (`.name: was ..., but now ...`), because `name` in state is never rewritten from the API response once known

#### Scenario: Cost limit sentinel value
- **WHEN** `costLimit.day`/`minute`/`month`/`week` is returned as a quoted numeric string (e.g. the int64-max "unlimited" sentinel)
- **THEN** `Read` succeeds instead of failing with a JSON decode error

#### Scenario: Hand-written limit JSON
- **WHEN** a `limits` value is configured as `{"minute": 64000000, "day": 100000000}` (e.g. in a `.tf.json` manifest) and the API returns `{"day":100000000,"minute":64000000}`
- **THEN** state keeps the configured string and apply does not fail with "inconsistent result after apply"

#### Scenario: Limit value changed remotely
- **WHEN** the API returns a `limits` value that decodes to different JSON than the configured one
- **THEN** state takes the API value, so the change shows up as drift

#### Scenario: Server-filled limit fields
- **WHEN** a `limits` value is configured as `{"day": 10}` and DIAL Core returns it with the other fields (`minute`, `week`, `month`, `requestHour`, `requestDay`) set to the int64-max sentinel
- **THEN** those sentinel fields are dropped and state stays `{"day":10}`

#### Scenario: Empty maps
- **WHEN** `limits` or `share` is configured as `{}`, or left unset, and the API returns `{}` or omits it
- **THEN** state matches the config (`{}` or null) and the next plan shows no changes

## Implementation Notes

- Role implementation follows the model/route architecture: API methods in `client.go`, Terraform resource in `role_resource.go`, resource registration in provider resource list.
- Diagnostics and error shape mirror the model/route implementations.
- Resource identity remains path-based (`bucket/path`) for parity and predictable imports.
- `setRoleState` applies, in order: `dropServerDefaultLimits` (sentinel fields), `mapOrPlannedEmpty` (null vs `{}`), then `preserveEquivalentJSON` (configured formatting), all in `helpers.go`. The API value is still canonicalized (compact, sorted keys) when no equivalent prior value exists, e.g. on import.
- Unit tests: `TestSetRoleStateKeepsEquivalentLimitsFormatting`, `TestSetRoleStateDropsServerDefaultLimitFields`, `TestSetRoleStateKeepsConfiguredEmptyMaps`.
- The `name`/`api_name` split and `FlexibleFloat64` cost-limit decoding were both added after production use of `aidial_role` against a real DIAL Core deployment surfaced them as apply-blocking bugs (see `role-schema` and `role-client` for the underlying schema/client details).
