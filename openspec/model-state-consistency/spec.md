# model-state-consistency Specification

## Purpose

Define state consistency and idempotency guarantees for the `aidial_model` resource.

## Requirements

### Requirement: State consistency and idempotency

Implementation SHALL avoid perpetual diffs and unnecessary updates.

It SHALL:

- re-read remote state after create/update
- normalize nested/object-like fields (`features`, `limits`, `pricing`, `upstreams`, `defaults`) to a stable Terraform representation, including explicit null objects/lists when the API omits them
- normalize set-like fields to a stable Terraform representation when empty — but not uniformly to null: `user_roles`, `dependencies`, `description_keywords`, `input_attachment_types`, `interceptors`, and `fields_hashing_order` normalize to an **explicit empty set** (not null) when the API returns none, while `reasoning_efforts` (nested inside `features`) is the one field that normalizes to an explicit **null** set when empty
- handle API omission of optional fields consistently
- preserve idempotent plan/apply behavior
- preserve the user-configured `name` in state when the API does not distinguish it from `api_name`

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

### Requirement: Null and empty values stay distinct

DIAL Core treats a missing value (`null`) and an empty one (`""`, `[]`) as different settings. The provider SHALL keep them apart in both directions.

- Attributes that DIAL Core leaves `null` when they are not sent are optional and **not computed**: `description`, `display_name`, `endpoint`, `responses_endpoint`, `override_name`, `display_version`, `icon_url`, `intro`, `reference`, `tokenizer_model`, `max_input_attachments`, `user_roles`, `input_attachment_types`, `features`, `limits`, `pricing`. Leaving one unset means "null in DIAL Core".
- Attributes that DIAL Core fills with a default stay optional and computed, so the default is accepted: `forward_auth_token` (`false`), `max_retry_attempts` (`5`), `defaults` (`{}`), `interceptors`, `description_keywords`, `dependencies`, `upstreams` (`[]`), `fields_hashing_order` (`["prefix.body.tools", "prefix.body.messages"]`).
- Lists: a configured `[]` is sent as `[]`; an unset list is not sent. On read, `null` stays null and `[]` stays an empty set.
- Strings: `""` is sent and read back as `""`; an unset string is not sent and reads back as null.
- An upstream without `endpoint` is sent without the field, not as `""`.

#### Scenario: Empty string set outside Terraform
- **WHEN** a model's config leaves `description` unset and the value in DIAL Core is changed to `""`
- **THEN** `terraform plan` shows `description = "" -> null`, and apply restores `null`

#### Scenario: Empty list versus unset list
- **WHEN** `user_roles = []` is configured
- **THEN** DIAL Core receives `"userRoles": []`; when `user_roles` is unset, the field is not sent and DIAL Core keeps `null`

#### Scenario: Core default accepted
- **WHEN** a config omits `max_retry_attempts` and DIAL Core stores its default `5`
- **THEN** the next plan shows no changes

## Implementation Notes

- Model implementation follows role architecture: API methods in `client.go`, Terraform resource in `model_resource.go`, resource registration in provider resource list.
- Diagnostics and error shape mirror existing role implementation.
- Resource identity remains path-based (`bucket/path`) for parity and predictable imports.
- The split between null-default and Core-default attributes was taken from a real DIAL Core response to a minimal model PUT (`name`, `type`, `displayName` only). The client models optional lists as `*[]string` and strings as `*string`, so nil (omitted) and empty stay distinct; `stringSetFromAPI` / `stringSetToAPI` in `helpers.go` do the list mapping. Unit tests: `model_null_vs_empty_test.go`.
