# Spec Delta

## ADDED Requirements

### Requirement: Alias identity accepts the public and platform buckets

`aidial_alias` and the data source `aidial_alias` SHALL require `bucket` and `path` with no defaults, and `bucket` SHALL accept only `public` and `platform`, the buckets DIAL Core accepts for aliases (any other bucket is a 404). `path` SHALL be non-empty, both SHALL force replacement on change, `id` SHALL be `<bucket>/<path>`, and import SHALL take `<bucket>/<path>`.

#### Scenario: Alias in another bucket
- **WHEN** an `aidial_alias` sets `bucket = "foo"`
- **THEN** validation fails at plan time

#### Scenario: Alias bucket or path omitted or empty
- **WHEN** `bucket` or `path` is omitted, or `path` is an empty string
- **THEN** validation fails at plan time
