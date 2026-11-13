# resource-identity Specification

## Purpose

System-wide rules for how `bucket` and `path` (the resource identity) are specified across all `aidial_*` resources and their data sources. Replaces `resource-identity-defaults`: identity is always written in configuration and never defaulted. Per-resource schema specs reference this spec instead of restating it.

## Requirements

### Requirement: Required bucket

`bucket` SHALL be required on `aidial_model`, `aidial_role`, `aidial_route`, `aidial_key` and `aidial_catalog_schema` and on the data sources `aidial_role`, `aidial_model` and `aidial_catalog_schema`. It SHALL have no default, SHALL NOT be computed, and SHALL accept only `platform`, the one bucket DIAL Core stores these types in.

#### Scenario: Bucket omitted
- **WHEN** a resource or data source config does not set `bucket`
- **THEN** validation fails at plan time

#### Scenario: Unsupported bucket
- **WHEN** `bucket` is set to any value other than `platform`
- **THEN** validation fails at plan time with a diagnostic naming the accepted value, before any API call

#### Scenario: Bucket still forces replacement
- **WHEN** the configured `bucket` changes on a resource
- **THEN** Terraform plans replacement

### Requirement: Required path

`path` SHALL be required and non-empty on the same resources and data sources, with no default and no relation to `name`. It MAY contain `/` to nest. A change of `name` alone SHALL NOT change the identity or plan a replacement.

#### Scenario: Path omitted
- **WHEN** a config sets `name` but not `path`
- **THEN** validation fails at plan time

#### Scenario: Empty path
- **WHEN** `path` is an empty string
- **THEN** validation fails at plan time

#### Scenario: Name changes alone
- **WHEN** only `name` changes and `bucket` and `path` are unchanged
- **THEN** the resource is updated in place, not replaced

#### Scenario: Path still forces replacement
- **WHEN** the configured `path` changes on a resource
- **THEN** Terraform plans replacement, because DIAL Core has no move operation for these resource types

### Requirement: Identity id and import

`id` SHALL be `<bucket>/<path>`. Import SHALL accept `<bucket>/<path>` and take both attributes from the import id.

#### Scenario: Resolved id
- **WHEN** a config sets `bucket = "platform"` and `path = "gpt-4o"`
- **THEN** `id` is `platform/gpt-4o`

#### Scenario: Import
- **WHEN** a resource is imported as `<bucket>/<path>`
- **THEN** both attributes are taken from the import id

### Requirement: Resources without bucket and path

`aidial_file_model` and `aidial_bucket` SHALL NOT take `bucket` or `path` inputs.

#### Scenario: File model lookup
- **WHEN** `aidial_file_model` is read
- **THEN** it is keyed by `name` only

### Requirement: Toolset bucket is not restricted to platform

For `aidial_toolset` and its data source, `bucket` SHALL be required and non-empty but SHALL NOT be restricted to `platform`, because DIAL Core accepts other buckets (for example `public`) for toolsets. The rules for `path`, `id` (`<bucket>/<path>`), replacement on change and import (`<bucket>/<path>`) are unchanged. The `platform`-only rule for the other resources and data sources is unchanged.

#### Scenario: Toolset in a non-platform bucket
- **WHEN** an `aidial_toolset` sets `bucket = "public"`
- **THEN** validation passes, and `id` is `public/<path>`

#### Scenario: Other resources unchanged
- **WHEN** `aidial_model` sets `bucket = "public"`
- **THEN** validation still fails at plan time

#### Scenario: Toolset bucket omitted or empty
- **WHEN** `aidial_toolset` omits `bucket` or sets it to an empty string
- **THEN** validation fails at plan time

### Requirement: Interceptor identity follows the platform-only rule

`aidial_interceptor` and the data source `aidial_interceptor` SHALL require `bucket` and `path` with no defaults, and `bucket` SHALL accept only `platform`, like models, roles, routes, keys and catalog schemas. `path` SHALL be non-empty, both SHALL force replacement on change, `id` SHALL be `<bucket>/<path>`, and import SHALL take `<bucket>/<path>`.

#### Scenario: Interceptor in another bucket
- **WHEN** an `aidial_interceptor` sets `bucket = "public"`
- **THEN** validation fails at plan time

#### Scenario: Interceptor bucket or path omitted or empty
- **WHEN** `bucket` or `path` is omitted, or `path` is an empty string
- **THEN** validation fails at plan time

## Verification

- Unit: `TestResourceIdentityRequired` and `TestDataSourceIdentityRequired` (schema shape, `platform`-only and non-empty validators, replace-on-change).
- Acceptance: `TestAccModel_nameChangeUpdatesInPlace` (a `name` change alone is an in-place update); every other `TestAcc*` sets `bucket` and `path` explicitly.
- Observed Core behavior behind the `platform`-only rule: `PUT /v1/{models,roles,keys,routes,catalog_schemas}/<bucket>/<path>` returns 200 for `platform` and 404 for any other bucket, including the caller's own bucket from `GET /v1/bucket`. This is not stated in `openapicore/open_api_core.yaml`.
