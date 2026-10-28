# Spec Delta

## Purpose

Lets Terraform configurations read an existing API-managed DIAL Core model without managing it, so other workspaces can reference models they did not create.

## ADDED Requirements

### Requirement: aidial_model data source arguments

The provider SHALL expose data source `aidial_model` with `name` (required), `bucket` (optional, default `platform`) and `path` (optional, default `name`). The model is looked up in resource storage at `<bucket>/<path>`. The data source MUST NOT read file-sourced models; those remain the job of `aidial_file_model`.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_model` is included alongside `aidial_file_model`

#### Scenario: Defaults applied
- **WHEN** only `name = "gpt-4o"` is configured
- **THEN** the model is read from bucket `platform` and path `gpt-4o`

#### Scenario: Explicit bucket and path
- **WHEN** `bucket` and `path` are configured
- **THEN** the model is read from that bucket and path, regardless of `name`

### Requirement: aidial_model data source attributes

The data source SHALL expose computed `id` (`<bucket>/<path>`), `api_name` and every model attribute that `aidial_file_model` exposes as computed, including the catalog attributes `catalog_schema_id` and `catalog_properties`. Attribute values SHALL match what the `aidial_model` resource would hold in state for the same model, including the null-vs-empty rules for nested objects, lists and sets.

#### Scenario: Existing model
- **WHEN** the model exists
- **THEN** `id`, `api_name` and the computed attributes are populated

#### Scenario: Omitted collections follow the resource
- **WHEN** the API omits nested objects, lists or sets
- **THEN** `features`, `limits`, `pricing`, `upstreams`, the string sets and `catalog_properties` are null, and `interfaces` and `default_headers` are empty maps, exactly as the `aidial_model` resource holds them

#### Scenario: Matches the resource
- **WHEN** a model is managed by an `aidial_model` resource and read by the data source
- **THEN** their attribute values are equal

### Requirement: aidial_model data source not found

A model that DIAL Core does not have SHALL produce an error diagnostic naming the bucket and path, and MUST NOT yield empty state.

#### Scenario: Missing model
- **WHEN** the API returns 404 for the model
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: API error
- **WHEN** the API returns a non-2xx status other than 404
- **THEN** the data source fails with a diagnostic carrying the status and body
