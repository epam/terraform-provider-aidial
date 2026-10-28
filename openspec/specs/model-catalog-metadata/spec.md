# model-catalog-metadata Specification

## Purpose

Let `aidial_model` carry catalog metadata (`catalog_schema_id`, `catalog_properties`) and keep its state consistent with DIAL Core.

## Requirements

### Requirement: Model catalog attributes

The `aidial_model` resource SHALL expose:

- `catalog_schema_id` (optional, string; sent as `catalogSchemaId`): URI of the catalog schema describing the model's catalog properties.
- `catalog_properties` (optional, free-form object as a Terraform dynamic attribute; sent as `catalogProperties`): arbitrary JSON values keyed by property name, with keys not renamed. It MUST be an object.

The `aidial_file_model` data source SHALL expose both as computed attributes.

#### Scenario: Properties sent as configured
- **WHEN** `catalog_properties = { provider = "Provider", vendor = "Vendor", license = "License", knowledgeCutoffDate = "2026-08-17", parameters = "100B" }` and `catalog_schema_id = "https://dial.epam.com/catalog-schemas/model"` are configured
- **THEN** the PUT payload contains `catalogSchemaId` and `catalogProperties` with the same keys and values

#### Scenario: Non-string property values
- **WHEN** a property value is a number, boolean, list or object
- **THEN** it is sent as that JSON type and read back without change

#### Scenario: Non-object catalog_properties
- **WHEN** `catalog_properties` is configured as a non-object value
- **THEN** the provider returns a diagnostic and does not call the API

#### Scenario: Referenced schema does not exist
- **WHEN** `catalog_schema_id` references a schema that DIAL Core does not have
- **THEN** create/update fails with the API error (400, "Catalog schema not found") surfaced as a diagnostic

### Requirement: Catalog attribute state consistency

State handling for the new attributes SHALL follow the model null-vs-empty rules.

#### Scenario: Unset stays null
- **WHEN** `catalog_properties` is not configured and the API omits it or returns `{}`
- **THEN** state holds null and the next plan shows no changes

#### Scenario: Configured empty object preserved
- **WHEN** `catalog_properties = {}` is configured
- **THEN** state keeps `{}` and the next plan shows no changes

#### Scenario: Equivalent JSON keeps configured value
- **WHEN** the API returns JSON semantically equal to the configured value (different key order or number formatting)
- **THEN** the configured value is kept in state

#### Scenario: Remote drift
- **WHEN** `catalogProperties` or `catalogSchemaId` is changed outside Terraform
- **THEN** the next refresh stores the API value and plan shows the drift

#### Scenario: Import
- **WHEN** an existing model with these fields is imported
- **THEN** the subsequent read populates both attributes from the API
