# Spec Delta

## MODIFIED Requirements

### Requirement: Path parameters are encoded as single segments
Each path parameter the client places in a request URL (`bucket` and `path` for roles, keys, models, routes, catalog schemas and application type schemas, and `name` for file-sourced models and file-sourced application type schemas) SHALL be percent-encoded as one URL path segment, so reserved characters including `/`, `?`, `#`, `%` and space never change the request's path or query.

#### Scenario: Slash in a path
- **WHEN** a role is read with bucket `platform` and path `team/admin`
- **THEN** the request path is `/v1/roles/platform/team%2Fadmin`

#### Scenario: Query delimiter in a path
- **WHEN** a model is read with path `a?b`
- **THEN** the request path is `/v1/models/platform/a%3Fb` and the request has no query string

#### Scenario: File model name
- **WHEN** a file-sourced model named `a/b` is read
- **THEN** the request path is `/v1/admin/config/file/models/a%2Fb`

#### Scenario: Application type schema path
- **WHEN** an application type schema is read with bucket `platform` and path `apps/quick`
- **THEN** the request path is `/v1/schemas/platform/apps%2Fquick`

#### Scenario: File application type schema name
- **WHEN** a file-sourced application type schema named `a/b` is read
- **THEN** the request path is `/v1/admin/config/file/schemas/a%2Fb`

#### Scenario: Plain names unchanged
- **WHEN** a parameter contains only letters, digits, `-`, `_`, `.` or `:`
- **THEN** it appears in the URL exactly as given
