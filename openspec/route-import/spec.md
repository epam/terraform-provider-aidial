# route-import Specification

## Purpose

Define the import behavior of the `aidial_route` resource.

## Requirements

### Requirement: Import behavior

Route resource SHALL support import using:

`<bucket>/<path>`

Import SHALL populate:

- `id`
- `bucket`
- `path`

Invalid import IDs SHALL return diagnostics with expected format.

#### Scenario: Valid import
- **WHEN** user imports `aidial_route.example` with `bucket/path`
- **THEN** import succeeds and sets identity attributes

#### Scenario: Invalid import format
- **WHEN** user imports without `<bucket>/<path>`
- **THEN** import fails with usage guidance diagnostic
