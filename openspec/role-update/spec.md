# role-update Specification

## Purpose

Define the `Update` behavior of the `aidial_role` resource.

## Requirements

### Requirement: Update behavior

`Update` SHALL:

1. Read plan
2. Convert plan to role payload
3. Call `SaveRole`
4. Re-read with `GetRole`
5. Persist refreshed state from API response, preserving the configured `name` and storing the API's canonical name in `api_name`

#### Scenario: Successful update
- **WHEN** role attributes change
- **THEN** save succeeds and refreshed state is persisted

#### Scenario: Post-update missing object
- **WHEN** save succeeds but GET returns missing
- **THEN** update fails with diagnostics

#### Scenario: Unconfigured optional attributes are not clobbered
- **WHEN** `description`, `display_name`, or `topics` are left unconfigured on an update to a role that already has values for them
- **THEN** the plan carries forward the prior state value for those attributes (via `UseStateForUnknown`) instead of clearing them on the API
