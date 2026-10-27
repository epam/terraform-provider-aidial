# route-read Specification

## Purpose

Define the `Read` behavior of the `aidial_route` resource.

## Requirements

### Requirement: Read behavior

`Read` SHALL fetch by `(bucket, path)` and refresh full state from API response.

On `404`, resource SHALL be removed from state.

#### Scenario: Successful refresh
- **WHEN** route exists remotely
- **THEN** state is updated to current remote representation

#### Scenario: External deletion
- **WHEN** route was deleted outside Terraform
- **THEN** refresh removes resource from state

#### Scenario: Read failure
- **WHEN** upstream read fails with non-2xx
- **THEN** read returns diagnostics
