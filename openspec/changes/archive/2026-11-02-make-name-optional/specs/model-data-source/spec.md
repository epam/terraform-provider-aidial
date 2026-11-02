# Spec Delta

## MODIFIED Requirements

### Requirement: aidial_model data source arguments

The provider SHALL expose data source `aidial_model` with `bucket` (required, must be `platform`) and `path` (required, non-empty), and `name` (optional, computed; not used for the lookup). The model is looked up in resource storage at `<bucket>/<path>`. The data source MUST NOT read file-sourced models; those remain the job of `aidial_file_model`.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_model` is included alongside `aidial_file_model`

#### Scenario: Lookup without name
- **WHEN** only `bucket` and `path` are configured
- **THEN** the model is read from that bucket and path and `name` is populated from the API

#### Scenario: Explicit bucket and path
- **WHEN** `bucket` and `path` are configured
- **THEN** the model is read from that bucket and path, regardless of any configured `name`
