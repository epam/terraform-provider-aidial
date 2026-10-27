# role-import Specification

## Purpose

Define the import behavior of the `aidial_role` resource.

## Requirements

### Requirement: Import behavior

Role resource SHALL support import using:

`<bucket>/<path>`

Import SHALL populate:

- `id`
- `bucket`
- `path`

Invalid import IDs SHALL return diagnostics with expected format.

#### Scenario: Valid import
- **WHEN** user imports `aidial_role.example` with `bucket/path`
- **THEN** import succeeds and sets identity attributes

#### Scenario: Invalid import format
- **WHEN** user imports without `<bucket>/<path>`
- **THEN** import fails with usage guidance diagnostic

## Implementation Notes

- Import only sets `id`, `bucket`, and `path`; `name` and all other attributes are populated by the subsequent `Read`. Because `name` is `Required` (not `Computed`), `setRoleState` falls back to the API's returned name only when `name` is not yet known in state (i.e. right after import); once a `name` value exists in state, it is preserved as-is and only `api_name` reflects subsequent API responses. This lets `terraform import` produce a usable `name` immediately while still protecting configured resources from being perturbed by the API's canonicalization (see `role-schema`).
