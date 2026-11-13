# Spec Delta

## ADDED Requirements

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
