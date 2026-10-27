# resource-identity-defaults Specification

## Purpose

System-wide rules for how `bucket` and `path` (the resource identity) are defaulted across all `aidial_*` resources. Per-resource schema specs (`model-schema`, `role-schema`, `route-schema`) reference this spec instead of restating it. Implemented in `helpers.go` (`defaultBucket`, `bucketDefaultsToPlatform`, `pathDefaultsToName`) and wired into each `*_resource.go` schema. `aidial_model`, `aidial_role` and `aidial_route` apply the bucket default through the `bucketDefaultsToPlatform` plan modifier; `aidial_key` uses a static schema default (`stringdefault.StaticString(defaultBucket)`). Both produce the same plan value.

## Requirements

### Requirement: Default bucket

`bucket` SHALL be optional and computed on `aidial_model`, `aidial_role`, `aidial_route` and `aidial_key`. When it is not configured, its value SHALL be `platform`. The defaulted value SHALL NOT be omitted: it is part of the plan, persisted in state, used in the API URL and in `id`.

#### Scenario: Bucket omitted
- **WHEN** a resource config does not set `bucket`
- **THEN** the plan, state and API request use `bucket = "platform"`

#### Scenario: Bucket set explicitly
- **WHEN** a resource config sets `bucket`
- **THEN** the configured value is used and the default is ignored

#### Scenario: Bucket still forces replacement
- **WHEN** the effective `bucket` changes (including to or from the default)
- **THEN** Terraform plans replacement

### Requirement: Default path from name

`path` SHALL be optional and computed on `aidial_model`, `aidial_role` and `aidial_route`. When it is not configured, its plan value SHALL equal the configured `name`. If `name` is not yet known at plan time, `path` SHALL stay unknown until it is.

`aidial_key` is excluded: its `name` is computed by the API, so `path` remains required there.

#### Scenario: Path omitted
- **WHEN** a resource config sets `name` but not `path`
- **THEN** `path` equals `name` in plan, state, API URL and `id`

#### Scenario: Path set explicitly
- **WHEN** a resource config sets `path`
- **THEN** the configured value is used, even if it differs from `name`

#### Scenario: Both omitted
- **WHEN** a config sets neither `bucket` nor `path`
- **THEN** `id` is `platform/<name>`

#### Scenario: Import
- **WHEN** a resource is imported as `<bucket>/<path>`
- **THEN** both are taken from the import id, never from the defaults

## Verification

- Unit: `TestPathDefaultsToName`, `TestModelSchemaBucketDefaultsToPlatform` (checks the schema shape and calls `bucketDefaultsToPlatform` directly).
- Integration manifests in `tests/integration/`: `tf_aidial_integration_model_bucket_path_omitted.tf`, `tf_aidial_integration_model_path_omitted.tf`, `tf_aidial_integration_role_bucket_path_omitted.tf` (each uses `check` blocks asserting the resolved values).
