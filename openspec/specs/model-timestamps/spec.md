# model-timestamps Specification

## Purpose

Lets `aidial_model` declare the `createdAt` and `updatedAt` timestamps that DIAL Core stores verbatim, so models migrated from another Core keep their original values.

## Requirements

### Requirement: Timestamp attributes are configurable

`aidial_model` SHALL accept `created_at` and `updated_at` as optional, non-computed integers holding Unix epoch milliseconds. Each value SHALL be at least 0; a negative value SHALL fail validation at plan time. The `aidial_model` and `aidial_file_model` data sources SHALL keep both attributes computed-only.

#### Scenario: Value in configuration is accepted
- **WHEN** a configuration sets `updated_at = 1630454400000`
- **THEN** plan succeeds without a read-only attribute error

#### Scenario: Negative value is rejected
- **WHEN** a configuration sets `created_at = -1`
- **THEN** plan fails with a validation error naming `created_at`

### Requirement: Timestamps are sent only when configured

Create and update SHALL send `createdAt` and `updatedAt` to Core only for the attributes that are configured. An unset attribute SHALL be omitted from the request body, so Core stores `null`.

#### Scenario: Configured values round-trip
- **WHEN** a model is applied with `created_at = 1111` and `updated_at = 1630454400000`
- **THEN** Core returns `createdAt` 1111 and `updatedAt` 1630454400000, and state holds the same values with no diff on the next plan

#### Scenario: Removing a value clears it
- **WHEN** `updated_at` is removed from the configuration of an existing model and the plan is applied
- **THEN** the PUT omits `updatedAt`, Core returns `null`, and state holds null

### Requirement: Read-back reflects Core

Read SHALL store `createdAt` and `updatedAt` from Core as-is, with null when Core omits or nulls them. A value present in Core but absent from the configuration SHALL plan as an update to null.

#### Scenario: Value set outside Terraform
- **WHEN** `updatedAt` is set on a model directly in Core and the configuration does not set `updated_at`
- **THEN** plan shows `updated_at` changing to null, and apply clears it in Core

#### Scenario: Import
- **WHEN** a model is imported
- **THEN** `created_at` and `updated_at` take the values Core returns, or null
