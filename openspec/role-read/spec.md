# role-read Specification

## Purpose

Define the `Read` behavior of the `aidial_role` resource.

## Requirements

### Requirement: Read behavior

`Read` SHALL fetch by `(bucket, path)` and refresh full state from API response, without overwriting the previously stored `name`.

On `404`, resource SHALL be removed from state.

#### Scenario: Successful refresh
- **WHEN** role exists remotely
- **THEN** state is updated to current remote representation

#### Scenario: External deletion
- **WHEN** role was deleted outside Terraform
- **THEN** refresh removes resource from state

#### Scenario: Read failure
- **WHEN** upstream read fails with non-2xx
- **THEN** read returns diagnostics

#### Scenario: API-canonicalized name does not perturb state
- **WHEN** the API returns a `name` value different from the one already stored in state (e.g. a stripped `roles/<bucket>/` prefix)
- **THEN** `name` in state is left unchanged and only `api_name` reflects the API's value
