# provider-identity Specification

## Purpose

Define the names under which the provider is addressed by Terraform users and by the Terraform Registry, so every resource, data source, binary and source address is consistent and publishable.

## Requirements

### Requirement: Provider type name
The provider SHALL register the type name `aidial`, so every resource and data source type starts with `aidial_`.

#### Scenario: Resource types
- **WHEN** the provider schema is exported
- **THEN** the resources are `aidial_role`, `aidial_key`, `aidial_model`, `aidial_route`, `aidial_catalog_schema`, `aidial_application_type_schema` and `aidial_toolset`, and the data sources are `aidial_bucket`, `aidial_role`, `aidial_model`, `aidial_file_model`, `aidial_catalog_schema`, `aidial_application_type_schema`, `aidial_file_application_type_schema`, `aidial_toolset` and `aidial_file_toolset`

#### Scenario: Old names absent
- **WHEN** a configuration uses `dial_role` or any other `dial_*` type
- **THEN** Terraform reports an unsupported resource or data source type

### Requirement: Registry source address
The provider SHALL be served at the address `registry.terraform.io/epam/aidial`, and documentation and examples SHALL use `source = "epam/aidial"`.

#### Scenario: Required providers block
- **WHEN** a user declares `aidial = { source = "epam/aidial" }` in `required_providers`
- **THEN** `terraform init` resolves the provider from the Registry

#### Scenario: Old address
- **WHEN** a configuration references `dial/dial`
- **THEN** no file in the repository documents it as a valid source

### Requirement: Module path and binary name
The Go module path SHALL be `github.com/epam/terraform-provider-aidial` and the built binary SHALL be named `terraform-provider-aidial`.

#### Scenario: Build
- **WHEN** `go build -o terraform-provider-aidial` runs
- **THEN** it succeeds and no source file imports `github.com/dial/terraform-provider-dial`

### Requirement: Behavior unchanged by the rename
Renaming SHALL NOT change attribute schemas, API semantics, identity (`<bucket>/<path>`) or null-vs-empty handling of any resource or data source.

#### Scenario: Existing tests
- **WHEN** the unit and integration suites run with the renamed types
- **THEN** they pass with no assertion changed other than type and address names

### Requirement: Interceptor types are registered

The provider SHALL register the resource `aidial_interceptor` and the data sources `aidial_interceptor` and `aidial_file_interceptor`, in addition to the types listed in "Provider type name".

#### Scenario: Interceptor types
- **WHEN** the provider schema is exported
- **THEN** it contains the resource `aidial_interceptor` and the data sources `aidial_interceptor` and `aidial_file_interceptor`

### Requirement: Application types are registered

The provider SHALL register the resource `aidial_application` and the data sources `aidial_application` and `aidial_file_application`, in addition to the types listed in "Provider type name".

#### Scenario: Application types
- **WHEN** the provider schema is exported
- **THEN** it contains the resource `aidial_application` and the data sources `aidial_application` and `aidial_file_application`
