# core-client-requests Specification

## Purpose

Defines how the provider's DIAL Core client turns resource addresses (bucket, path, name) into request URLs, so every value reaches Core as the single path parameter the API contract declares.

## Requirements

### Requirement: Path parameters are encoded as single segments
Each path parameter the client places in a request URL (`bucket` and `path` for roles, keys, models, routes and catalog schemas, and `name` for file-sourced models) SHALL be percent-encoded as one URL path segment, so reserved characters including `/`, `?`, `#`, `%` and space never change the request's path or query.

#### Scenario: Slash in a path
- **WHEN** a role is read with bucket `platform` and path `team/admin`
- **THEN** the request path is `/v1/roles/platform/team%2Fadmin`

#### Scenario: Query delimiter in a path
- **WHEN** a model is read with path `a?b`
- **THEN** the request path is `/v1/models/platform/a%3Fb` and the request has no query string

#### Scenario: File model name
- **WHEN** a file-sourced model named `a/b` is read
- **THEN** the request path is `/v1/admin/config/file/models/a%2Fb`

#### Scenario: Plain names unchanged
- **WHEN** a parameter contains only letters, digits, `-`, `_`, `.` or `:`
- **THEN** it appears in the URL exactly as given
