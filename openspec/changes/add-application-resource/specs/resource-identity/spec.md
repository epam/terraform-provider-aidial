# Spec Delta

## ADDED Requirements

### Requirement: Application identity follows the toolset rule

`aidial_application` and the data source `aidial_application` SHALL require `bucket` and `path` with no defaults. `bucket` SHALL be any non-empty string, like toolsets, because Core accepts `public` as well as `platform` for applications. `path` SHALL be non-empty and MAY contain `/`. Both SHALL force replacement on change, `id` SHALL be `<bucket>/<path>`, and import SHALL take `<bucket>/<path>`, splitting at the first `/`.

#### Scenario: Application in the public bucket
- **WHEN** an `aidial_application` sets `bucket = "public"`
- **THEN** validation passes

#### Scenario: Application bucket or path omitted or empty
- **WHEN** `bucket` or `path` is omitted or an empty string
- **THEN** validation fails at plan time
