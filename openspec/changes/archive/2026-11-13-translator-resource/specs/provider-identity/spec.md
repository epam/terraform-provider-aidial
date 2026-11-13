# Spec Delta

## ADDED Requirements

### Requirement: Translator types are registered

The provider SHALL register the resource `aidial_translator` and the data sources `aidial_translator` and `aidial_file_translator`, in addition to the types listed in "Provider type name".

#### Scenario: Translator types
- **WHEN** the provider schema is exported
- **THEN** it contains the resource `aidial_translator` and the data sources `aidial_translator` and `aidial_file_translator`
