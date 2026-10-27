# Spec Delta

## Purpose

Define the names under which the provider is addressed by Terraform users and by the Terraform Registry, so every resource, data source, binary and source address is consistent and publishable.

## ADDED Requirements

### Requirement: Provider type name
The provider SHALL register the type name `aidial`, so every resource and data source type starts with `aidial_`.

#### Scenario: Resource types
- **WHEN** the provider schema is exported
- **THEN** the resources are `aidial_role`, `aidial_key`, `aidial_model` and `aidial_route`, and the data sources are `aidial_bucket`, `aidial_role` and `aidial_file_model`

#### Scenario: Old names absent
- **WHEN** a configuration uses `dial_role` or any other `dial_*` type
- **THEN** Terraform reports an unsupported resource or data source type

### Requirement: Registry source address
The provider SHALL be served at the address `registry.terraform.io/alexey-ban/aidial`, and documentation and examples SHALL use `source = "alexey-ban/aidial"`.

#### Scenario: Required providers block
- **WHEN** a user declares `aidial = { source = "alexey-ban/aidial" }` in `required_providers`
- **THEN** `terraform init` resolves the provider from the Registry

#### Scenario: Old address
- **WHEN** a configuration references `dial/dial`
- **THEN** no file in the repository documents it as a valid source

### Requirement: Module path and binary name
The Go module path SHALL be `github.com/alexey-ban/terraform-provider-aidial` and the built binary SHALL be named `terraform-provider-aidial`.

#### Scenario: Build
- **WHEN** `go build -o terraform-provider-aidial` runs
- **THEN** it succeeds and no source file imports `github.com/dial/terraform-provider-dial`

### Requirement: Behavior unchanged by the rename
Renaming SHALL NOT change attribute schemas, API semantics, identity (`<bucket>/<path>`) or null-vs-empty handling of any resource or data source.

#### Scenario: Existing tests
- **WHEN** the unit and integration suites run with the renamed types
- **THEN** they pass with no assertion changed other than type and address names
