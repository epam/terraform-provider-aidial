# route-state-consistency Specification

## Purpose

Define state consistency and idempotency guarantees for the `aidial_route` resource.

## Requirements

### Requirement: State consistency and idempotency

Implementation SHALL avoid perpetual diffs and unnecessary updates.

It SHALL:

- re-read remote state after create/update
- never overwrite a known `name` (configured, or already stored after an earlier read) from an API response; expose the API's own value through the `Computed`-only `api_name` attribute, and when `name` is not configured set it from the API response only on create or import, then keep it with `UseStateForUnknown`
- normalize map/object-like fields to stable Terraform representation
- handle API omission of optional fields consistently
- preserve idempotent plan/apply behavior

#### Scenario: Repeated plan with no config changes
- **WHEN** no local or remote changes occurred
- **THEN** `terraform plan` returns no changes

#### Scenario: Repeated apply
- **WHEN** same configuration is applied twice
- **THEN** second apply is a no-op

#### Scenario: Remote drift
- **WHEN** route is modified outside Terraform
- **THEN** next refresh/plan reflects actionable drift

#### Scenario: Optional field omitted by API
- **WHEN** API omits optional empty fields
- **THEN** state normalization prevents repeated diff churn

#### Scenario: API canonicalizes the route name
- **WHEN** the API returns a `name` that differs from the configured value
- **THEN** Terraform does not report a provider error (`.name: was ..., but now ...`), because `name` in state is never rewritten from the API response once known

#### Scenario: Omitted name causes no drift
- **WHEN** `name` is not configured and the resource was applied once
- **THEN** the next plan shows no changes, and no provider-consistency error is reported for `name`

## Implementation Notes

- Route implementation follows role architecture: API methods in `client.go`, Terraform resource in `route_resource.go`, resource registration in provider resource list.
- Diagnostics and error shape mirror existing role implementation.
- Resource identity remains path-based (`bucket/path`) for parity and predictable imports.
