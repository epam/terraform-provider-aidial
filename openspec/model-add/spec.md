# model-add Specification

## Purpose

Define the `Create` behavior of the `aidial_model` resource.

## Requirements

### Requirement: Create behavior

`Create` SHALL:

1. Read plan
2. Convert plan to model payload
3. Call `SaveModel`
4. Re-read using `GetModel`
5. Write state from API response

If save succeeds but subsequent read returns missing model, create SHALL fail with diagnostics.

#### Scenario: Successful create
- **WHEN** valid model config is applied
- **THEN** model is saved and state reflects remote object

#### Scenario: Invalid planned model
- **WHEN** plan-to-payload conversion fails (e.g. malformed `user_roles`)
- **THEN** create returns diagnostics and does not call API save

#### Scenario: Save succeeded but model missing
- **WHEN** PUT succeeds but GET returns not found
- **THEN** create fails with a post-create-not-found diagnostic
