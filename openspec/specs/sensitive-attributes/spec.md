# sensitive-attributes Specification

## Purpose

Defines which provider attributes carry secrets and requires them to be marked sensitive wherever they appear, so Terraform redacts them in plan, apply and console output.

## Requirements

### Requirement: Secret attributes are sensitive in every schema
The attributes `api_key` and `id_token` (provider), `key` (`aidial_key`), the upstream `key` and `secret_extra_data` (`aidial_model`, `aidial_route`, and the `aidial_model` and `aidial_file_model` data sources) and `auth_settings.client_secret` (`aidial_toolset`) SHALL be marked sensitive in every schema that exposes them, including read-only data sources. The `aidial_toolset` and `aidial_file_toolset` data sources do not expose `client_secret`.

#### Scenario: Data source upstream secrets redacted
- **WHEN** a plan reads an `aidial_model` or `aidial_file_model` data source whose model has upstreams with `key` or `secret_extra_data`
- **THEN** Terraform shows `(sensitive value)` for those attributes instead of their contents

#### Scenario: Schema export
- **WHEN** `terraform providers schema -json` is run
- **THEN** every listed attribute is reported with `"sensitive": true` in each resource and data source that has it

#### Scenario: Provider credentials
- **WHEN** `terraform providers schema -json` is run
- **THEN** the provider's `api_key` and `id_token` are reported with `"sensitive": true`

#### Scenario: Toolset client secret
- **WHEN** `terraform providers schema -json` is run
- **THEN** `auth_settings.client_secret` of `aidial_toolset` is reported with `"sensitive": true`, and the toolset data sources have no such attribute

#### Scenario: Non-secret siblings stay visible
- **WHEN** the same upstream is shown in plan output
- **THEN** `endpoint`, `responses_endpoint`, `id`, `extra_data`, `tier` and `weight` remain visible
