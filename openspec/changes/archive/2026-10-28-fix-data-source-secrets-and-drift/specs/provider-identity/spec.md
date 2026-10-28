# Spec Delta

## MODIFIED Requirements

### Requirement: Provider type name
The provider SHALL register the type name `aidial`, so every resource and data source type starts with `aidial_`.

#### Scenario: Resource types
- **WHEN** the provider schema is exported
- **THEN** the resources are `aidial_role`, `aidial_key`, `aidial_model`, `aidial_route` and `aidial_catalog_schema`, and the data sources are `aidial_bucket`, `aidial_role`, `aidial_model`, `aidial_file_model` and `aidial_catalog_schema`

#### Scenario: Old names absent
- **WHEN** a configuration uses `dial_role` or any other `dial_*` type
- **THEN** Terraform reports an unsupported resource or data source type
