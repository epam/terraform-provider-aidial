# key-state-consistency Specification

## Purpose

Defines how the `aidial_key` resource maps DIAL Core responses into Terraform state so that refreshes produce no spurious differences and omitted values stay distinct from empty ones.

## Requirements

### Requirement: Omitted project and role are null
When DIAL Core returns a key without `project` or `role`, the `aidial_key` resource SHALL store null for that attribute. The only exception is when the planned value (on create or update) or the prior state value (on refresh) is the empty string; then the empty string SHALL be kept, because Core cannot echo it back.

#### Scenario: API omits project and role
- **WHEN** a key is created without `project` or `role` in configuration and Core returns neither
- **THEN** state holds null for both, and the next plan shows no changes

#### Scenario: Explicit empty string
- **WHEN** the configuration sets `project = ""`
- **THEN** state holds `""` after apply and after refresh, and the next plan shows no changes

#### Scenario: Existing empty-string state
- **WHEN** a key whose state holds `project = ""` from an earlier provider version is refreshed
- **THEN** state keeps `""`, and the plan shows no changes

#### Scenario: Removed outside Terraform
- **WHEN** state holds `role = "basic"` and Core now returns the key without `role`
- **THEN** refresh stores null, and the plan shows the configured `role` being restored

#### Scenario: Value returned
- **WHEN** Core returns `role = "basic"`
- **THEN** state holds `"basic"`
