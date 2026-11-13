# Spec Delta

## ADDED Requirements

### Requirement: Interceptor identity follows the platform-only rule

`aidial_interceptor` and the data source `aidial_interceptor` SHALL require `bucket` and `path` with no defaults, and `bucket` SHALL accept only `platform`, like models, roles, routes, keys and catalog schemas. `path` SHALL be non-empty, both SHALL force replacement on change, `id` SHALL be `<bucket>/<path>`, and import SHALL take `<bucket>/<path>`.

#### Scenario: Interceptor in another bucket
- **WHEN** an `aidial_interceptor` sets `bucket = "public"`
- **THEN** validation fails at plan time

#### Scenario: Interceptor bucket or path omitted or empty
- **WHEN** `bucket` or `path` is omitted, or `path` is an empty string
- **THEN** validation fails at plan time
