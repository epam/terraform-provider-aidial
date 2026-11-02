# Spec Delta

## MODIFIED Requirements

### Requirement: State consistency and idempotency

Implementation SHALL avoid perpetual diffs and unnecessary updates.

It SHALL:

- re-read remote state after create/update
- normalize nested/object-like fields (`features`, `limits`, `pricing`, `upstreams`, `defaults`) to a stable Terraform representation, including explicit null objects/lists when the API omits them
- normalize set-like fields to a stable Terraform representation when empty — but not uniformly to null: `user_roles`, `dependencies`, `description_keywords`, `input_attachment_types`, `interceptors`, and `fields_hashing_order` normalize to an **explicit empty set** (not null) when the API returns none, while `reasoning_efforts` (nested inside `features`) is the one field that normalizes to an explicit **null** set when empty
- handle API omission of optional fields consistently
- preserve idempotent plan/apply behavior
- preserve a known `name` in state; when `name` is not configured, set it from `api_name` on create or import and keep it stable afterwards

#### Scenario: Repeated plan with no config changes
- **WHEN** no local or remote changes occurred
- **THEN** `terraform plan` returns no changes

#### Scenario: Repeated apply
- **WHEN** same configuration is applied twice
- **THEN** second apply is a no-op

#### Scenario: Remote drift
- **WHEN** model is modified outside Terraform
- **THEN** next refresh/plan reflects actionable drift

#### Scenario: Optional field omitted by API
- **WHEN** API omits optional empty fields (e.g. `features`, `limits`, `pricing`, `upstreams`, or optional sets)
- **THEN** state normalization prevents repeated diff churn

#### Scenario: Empty set-like field stored as empty, not null
- **WHEN** the API returns no values for `user_roles`, `dependencies`, `description_keywords`, `input_attachment_types`, `interceptors`, or `fields_hashing_order`
- **THEN** state stores an explicit empty set for that attribute, not null — configuring the attribute as `= []` must not produce a perpetual diff against a null value in state

#### Scenario: Omitted name causes no drift
- **WHEN** `name` is not configured and the model was applied once
- **THEN** the next plan shows no changes, and no provider-consistency error is reported for `name`
