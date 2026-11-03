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

### Requirement: Allowed IP ranges are a set of CIDRs
The `aidial_key` resource SHALL expose DIAL Core's `allowedIpAddressRanges` as the optional `allowed_ip_address_ranges` set of CIDR strings (IPv4 or IPv6). Configuration SHALL reject values that are not valid CIDRs. The resource SHALL send the set to Core as a JSON array of CIDR strings, the only form Core accepts, and SHALL read it back from Core's `IpAddressRanges` object (`ranges[]` of `mask` and `maskedBaseIp` bytes), decoding bytes given as a base64 string or an array of numbers. An unset attribute SHALL be omitted from the request, which lets any client use the key; an empty set SHALL be sent as `[]`, which lets no client use it. Core stores each range with its host bits cleared, so state SHALL keep the prior (planned or stored) set whenever Core's ranges are equivalent after normalization, and SHALL take Core's ranges otherwise. When Core's ranges cannot be decoded, state SHALL keep the prior value; without one it SHALL be null. Core omitting the ranges SHALL be stored as null.

#### Scenario: Unset means any client
- **WHEN** a key is created without `allowed_ip_address_ranges` and Core returns no ranges
- **THEN** the request omits the field, state holds null, and the next plan shows no changes

#### Scenario: Empty set means no client
- **WHEN** the configuration sets `allowed_ip_address_ranges = []`
- **THEN** the request sends `[]`, state holds an empty set, and the next plan shows no changes

#### Scenario: Core normalizes the range
- **WHEN** the configuration sets `["198.51.100.14/24"]` and Core returns the range `198.51.100.0/24`
- **THEN** state keeps `198.51.100.14/24` and the next plan shows no changes

#### Scenario: Changed outside Terraform
- **WHEN** state holds `["10.0.0.0/8"]` and Core now returns `192.0.2.0/24`
- **THEN** refresh stores `["192.0.2.0/24"]` and the plan shows the configured ranges being restored

#### Scenario: Invalid CIDR
- **WHEN** the configuration contains a value that is not a CIDR
- **THEN** validation fails at plan time
