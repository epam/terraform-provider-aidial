# Spec Delta

## ADDED Requirements

### Requirement: Translator identity follows the platform-only rule

`aidial_translator` and the data source `aidial_translator` SHALL require `bucket` and `path` with no defaults, and `bucket` SHALL accept only `platform`, like models, roles, routes, keys, catalog schemas and interceptors. `path` SHALL be non-empty, both SHALL force replacement on change, `id` SHALL be `<bucket>/<path>`, and import SHALL take `<bucket>/<path>`.

#### Scenario: Translator in another bucket
- **WHEN** an `aidial_translator` sets `bucket = "public"`
- **THEN** validation fails at plan time

#### Scenario: Translator bucket or path omitted or empty
- **WHEN** `bucket` or `path` is omitted, or `path` is an empty string
- **THEN** validation fails at plan time
