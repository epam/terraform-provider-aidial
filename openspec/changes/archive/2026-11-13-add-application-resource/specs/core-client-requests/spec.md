# Spec Delta

## ADDED Requirements

### Requirement: Application path keeps its folder separators

The client SHALL percent-encode each `/`-separated segment of an application `path`, and `bucket`, on its own, keeping the `/` between segments, like toolsets, because Core reads an application path as folders and rejects an encoded slash. The `name` of a file-sourced application SHALL be encoded as one segment.

#### Scenario: Nested application path
- **WHEN** an application is read with bucket `public` and path `team/app`
- **THEN** the request path is `/v1/applications/public/team/app`

#### Scenario: File application name
- **WHEN** a file-sourced application named `a/b` is read
- **THEN** the request path is `/v1/admin/config/file/applications/a%2Fb`
