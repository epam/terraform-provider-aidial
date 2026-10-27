# route-update Specification

## Purpose

Define the `Update` behavior of the `aidial_route` resource.

## Requirements

### Requirement: Update behavior

`Update` SHALL:

1. Read plan
2. Convert plan to route payload
3. Call `SaveRoute`
4. Re-read with `GetRoute`
5. Persist refreshed state from API response

#### Scenario: Successful update
- **WHEN** route attributes change
- **THEN** save succeeds and refreshed state is persisted

#### Scenario: Post-update missing object
- **WHEN** save succeeds but GET returns missing
- **THEN** update fails with diagnostics
