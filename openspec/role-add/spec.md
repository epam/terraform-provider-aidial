# role-add Specification

## Purpose

Define the `Create` behavior of the `aidial_role` resource.

## Requirements

### Requirement: Create behavior

`Create` SHALL:

1. Read plan
2. Convert plan to role payload
3. Call `SaveRole`
4. Re-read using `GetRole`
5. Write state from API response, preserving the configured `name` and storing the API's canonical name in `api_name`

If save succeeds but subsequent read returns missing role, create SHALL fail with diagnostics.

#### Scenario: Successful create
- **WHEN** valid role config is applied
- **THEN** role is saved and state reflects remote object

#### Scenario: Invalid planned role
- **WHEN** plan-to-payload conversion fails (e.g. malformed `limits`, `share`, or `topics`)
- **THEN** create returns diagnostics and does not call API save

#### Scenario: Save succeeded but role missing
- **WHEN** PUT succeeds but GET returns not found
- **THEN** create fails with a post-create-not-found diagnostic
