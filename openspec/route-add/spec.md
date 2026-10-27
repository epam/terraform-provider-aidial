# route-add Specification

## Purpose

Define the `Create` behavior of the `aidial_route` resource.

## Requirements

### Requirement: Create behavior

`Create` SHALL:

1. Read plan
2. Convert plan to route payload
3. Call `SaveRoute`
4. Re-read using `GetRoute`
5. Write state from API response

If save succeeds but subsequent read returns missing route, create SHALL fail with diagnostics.

#### Scenario: Successful create
- **WHEN** valid route config is applied
- **THEN** route is saved and state reflects remote object

#### Scenario: Invalid planned route
- **WHEN** plan-to-payload conversion fails
- **THEN** create returns diagnostics and does not call API save

#### Scenario: Save succeeded but route missing
- **WHEN** PUT succeeds but GET returns not found
- **THEN** create fails with a post-create-not-found diagnostic
