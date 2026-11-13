# Spec Delta

## ADDED Requirements

### Requirement: Alias path parameters are encoded as single segments

The client SHALL percent-encode `bucket` and `path` of an alias, and `name` of a file-sourced alias, as one URL path segment each, like roles and models, so `/` and other reserved characters never change the request path.

#### Scenario: Slash in an alias path
- **WHEN** an alias is read with bucket `public` and path `team/fast`
- **THEN** the request path is `/v1/aliases/public/team%2Ffast`

#### Scenario: File alias name
- **WHEN** a file-sourced alias named `a/b` is read
- **THEN** the request path is `/v1/admin/config/file/aliases/a%2Fb`
