# Spec Delta

## ADDED Requirements

### Requirement: Interceptor path parameters are encoded as single segments

The client SHALL percent-encode `bucket` and `path` of an interceptor, and `name` of a file-sourced interceptor, as one URL path segment each, like roles and models, so `/` and other reserved characters never change the request path. Core rejects `/` in an interceptor path in either form, so the error surfaces at apply.

#### Scenario: Slash in an interceptor path
- **WHEN** an interceptor is read with bucket `platform` and path `team/filter`
- **THEN** the request path is `/v1/interceptors/platform/team%2Ffilter`

#### Scenario: File interceptor name
- **WHEN** a file-sourced interceptor named `a/b` is read
- **THEN** the request path is `/v1/admin/config/file/interceptors/a%2Fb`
