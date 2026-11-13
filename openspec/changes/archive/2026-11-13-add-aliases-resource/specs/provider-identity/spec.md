# Spec Delta

## ADDED Requirements

### Requirement: Alias types are registered

The provider SHALL register the resource `aidial_alias` and the data sources `aidial_alias` and `aidial_file_alias`, in addition to the types listed in "Provider type name".

#### Scenario: Alias types
- **WHEN** the provider schema is exported
- **THEN** it contains the resource `aidial_alias` and the data sources `aidial_alias` and `aidial_file_alias`
