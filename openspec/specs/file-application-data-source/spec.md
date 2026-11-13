# file-application-data-source Specification

## Purpose

Read applications defined in DIAL Core's static configuration through `aidial_file_application`.

## Requirements

### Requirement: aidial_file_application data source

The data source `aidial_file_application` SHALL take a required, non-empty `name`, read `GET /v1/admin/config/file/applications/{name}`, and expose the same computed attributes as `aidial_application` minus `bucket`, `path` and `id`. `name` SHALL be percent-encoded as one path segment. A 404 SHALL be an error naming the application.

#### Scenario: Found
- **WHEN** a static application with the given name exists
- **THEN** its attributes are populated

#### Scenario: Missing
- **WHEN** Core returns 404
- **THEN** the data source fails with an error naming `name`

#### Scenario: Empty name
- **WHEN** `name` is empty
- **THEN** validation fails at plan time
