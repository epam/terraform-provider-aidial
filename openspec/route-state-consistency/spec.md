# route-state-consistency Specification

## Purpose

Define state consistency and idempotency guarantees for the `aidial_route` resource.

## Requirements

### Requirement: State consistency and idempotency

Implementation SHALL avoid perpetual diffs and unnecessary updates.

It SHALL:

- re-read remote state after create/update
- never overwrite the configured, `Required` `name` attribute from an API response once it is known; expose the API's own value only through the `Computed`-only `api_name` attribute
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

## Implementation Notes

- Route implementation follows role architecture: API methods in `client.go`, Terraform resource in `route_resource.go`, resource registration in provider resource list.
- Diagnostics and error shape mirror existing role implementation.
- Resource identity remains path-based (`bucket/path`) for parity and predictable imports.
