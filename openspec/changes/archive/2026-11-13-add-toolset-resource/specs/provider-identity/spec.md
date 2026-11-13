# Spec Delta

## MODIFIED Requirements

### Requirement: Provider type name
The provider SHALL register the type name `aidial`, so every resource and data source type starts with `aidial_`.

#### Scenario: Resource types
- **WHEN** the provider schema is exported
- **THEN** the resources are `aidial_role`, `aidial_key`, `aidial_model`, `aidial_route`, `aidial_catalog_schema`, `aidial_application_type_schema` and `aidial_toolset`, and the data sources are `aidial_bucket`, `aidial_role`, `aidial_model`, `aidial_file_model`, `aidial_catalog_schema`, `aidial_application_type_schema`, `aidial_file_application_type_schema`, `aidial_toolset` and `aidial_file_toolset`

#### Scenario: Old names absent
- **WHEN** a configuration uses `dial_role` or any other `dial_*` type
- **THEN** Terraform reports an unsupported resource or data source type
