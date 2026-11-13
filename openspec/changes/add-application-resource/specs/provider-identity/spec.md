# Spec Delta

## ADDED Requirements

### Requirement: Application types are registered

The provider SHALL register the resource `aidial_application` and the data sources `aidial_application` and `aidial_file_application`, in addition to the types listed in "Provider type name".

#### Scenario: Application types
- **WHEN** the provider schema is exported
- **THEN** it contains the resource `aidial_application` and the data sources `aidial_application` and `aidial_file_application`
