# file-translator-data-source Specification

## Purpose

Let Terraform configurations read a translator defined in DIAL Core's static configuration, which has no bucket or path and cannot be managed through the API.

## Requirements

### Requirement: File translator client method

`internal/client/translator.go` SHALL provide `GetFileTranslator(ctx, name)` against `GET /v1/admin/config/file/translators/{name}`. It SHALL return the translator, or `nil` when it does not exist (404). There is no save or delete method, because the endpoint is read-only.

#### Scenario: Translator found
- **WHEN** GET returns a `Translator` body
- **THEN** the method returns it decoded

#### Scenario: Translator not found
- **WHEN** the API returns 404
- **THEN** the method returns nil without an error

#### Scenario: Request conventions
- **WHEN** the method calls the API
- **THEN** it sends the `Api-Key` header and percent-encodes `name` as one path segment

### Requirement: aidial_file_translator data source

The provider SHALL expose data source `aidial_file_translator` with `name` (required, non-empty) and the computed attributes `base_url`, `in` and `out`. A missing translator SHALL produce an error diagnostic naming it.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_file_translator` is included alongside `aidial_translator`

#### Scenario: Existing translator
- **WHEN** the static-config translator exists
- **THEN** `base_url`, `in` and `out` are populated

#### Scenario: Missing translator
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: Name required
- **WHEN** `name` is omitted or empty
- **THEN** validation fails at plan time
