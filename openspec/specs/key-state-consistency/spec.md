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

### Known gap: allowedIpAddressRanges is not exposed
The `aidial_key` resource SHALL NOT expose DIAL Core's `allowedIpAddressRanges` while Core cannot round-trip it through the API. Core accepts a JSON array of CIDR strings on `PUT /v1/keys` but persists and serves the `IpAddressRanges` object (`ranges[]` of base64 `mask` and `maskedBaseIp`), which its own deserializer rejects on reload. Verified against Core 0.49.3: a key created with ranges reads back as valid until Core drops its in-memory snapshot, then reports `status: "invalid"` with a validation warning, and `DELETE` fails with HTTP 500 ("Stored key entity is unreadable"). Exposing the field would let Terraform create keys it can no longer read or delete. Revisit when Core fixes the round trip.

