# Spec Delta

## ADDED Requirements

### Requirement: Translator path parameters are encoded as single segments

The client SHALL percent-encode `bucket` and `path` of a translator, and `name` of a file-sourced translator, as one URL path segment each, like roles and models, so `/` and other reserved characters never change the request path. Core rejects `/` in a translator path in either form, so the error surfaces at apply.

#### Scenario: Slash in a translator path
- **WHEN** a translator is read with bucket `platform` and path `team/openai`
- **THEN** the request path is `/v1/translators/platform/team%2Fopenai`

#### Scenario: File translator name
- **WHEN** a file-sourced translator named `a/b` is read
- **THEN** the request path is `/v1/admin/config/file/translators/a%2Fb`
