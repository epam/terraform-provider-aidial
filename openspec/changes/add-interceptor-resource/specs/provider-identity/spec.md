# Spec Delta

## ADDED Requirements

### Requirement: Interceptor types are registered

The provider SHALL register the resource `aidial_interceptor` and the data sources `aidial_interceptor` and `aidial_file_interceptor`, in addition to the types listed in "Provider type name".

#### Scenario: Interceptor types
- **WHEN** the provider schema is exported
- **THEN** it contains the resource `aidial_interceptor` and the data sources `aidial_interceptor` and `aidial_file_interceptor`
