# model-delete Specification

## Purpose

Define the `Delete` behavior of the `aidial_model` resource.

## Requirements

### Requirement: Delete behavior

`Delete` SHALL call `DeleteModel(bucket, path)` and surface diagnostics on non-idempotent failures.

`404` from API SHALL be treated as success.

#### Scenario: Successful delete
- **WHEN** model exists
- **THEN** delete completes successfully

#### Scenario: Model already deleted
- **WHEN** API responds `404 Not Found`
- **THEN** Terraform delete still succeeds

#### Scenario: Upstream delete error
- **WHEN** API responds non-2xx other than 404
- **THEN** delete fails with diagnostics
