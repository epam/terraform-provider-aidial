# core-client-requests Specification

## Purpose

Defines how the provider's DIAL Core client turns resource addresses (bucket, path, name) into request URLs, so every value reaches Core as the single path parameter the API contract declares.

## Requirements

### Requirement: Path parameters are encoded as single segments
Each path parameter the client places in a request URL (`bucket` and `path` for roles, keys, models, routes, catalog schemas and application type schemas, `bucket` for toolsets, and `name` for file-sourced models, file-sourced application type schemas and file-sourced toolsets) SHALL be percent-encoded as one URL path segment, so reserved characters including `/`, `?`, `#`, `%` and space never change the request's path or query.

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

#### Scenario: File toolset name
- **WHEN** a file-sourced toolset named `a/b` is read
- **THEN** the request path is `/v1/admin/config/file/toolsets/a%2Fb`

#### Scenario: Plain names unchanged
- **WHEN** a parameter contains only letters, digits, `-`, `_`, `.` or `:`
- **THEN** it appears in the URL exactly as given

### Requirement: Toolset paths keep their slashes
A toolset `path` SHALL be sent with its `/` separators intact, with each segment between them percent-encoded on its own, because DIAL Core reads the path as folders and rejects an encoded slash.

#### Scenario: Toolset path
- **WHEN** a toolset is read with bucket `public` and path `team/tool`
- **THEN** the request path is `/v1/toolsets/public/team/tool`

#### Scenario: Reserved characters in a toolset path segment
- **WHEN** a toolset is read with path `a b/c?d`
- **THEN** the request path is `/v1/toolsets/public/a%20b/c%3Fd` and the request has no query string

### Requirement: Interceptor path parameters are encoded as single segments

The client SHALL percent-encode `bucket` and `path` of an interceptor, and `name` of a file-sourced interceptor, as one URL path segment each, like roles and models, so `/` and other reserved characters never change the request path. Core rejects `/` in an interceptor path in either form, so the error surfaces at apply.

#### Scenario: Slash in an interceptor path
- **WHEN** an interceptor is read with bucket `platform` and path `team/filter`
- **THEN** the request path is `/v1/interceptors/platform/team%2Ffilter`

#### Scenario: File interceptor name
- **WHEN** a file-sourced interceptor named `a/b` is read
- **THEN** the request path is `/v1/admin/config/file/interceptors/a%2Fb`
