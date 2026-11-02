# Spec Delta

## MODIFIED Requirements

### Requirement: Model Terraform resource

The provider SHALL expose a resource named `<provider>_model` (e.g. `aidial_model`) implemented in `model_resource.go`, matching `role_resource.go` conventions.

The schema SHALL include:

- `id` (computed)
- `bucket` (required, must be `platform`, replace on change; see `resource-identity`)
- `path` (required, non-empty, replace on change; see `resource-identity`)
- `name` (optional, computed; model name sent to the API when configured, otherwise derived by DIAL Core from `path` and read back; see Name handling below)
- `api_name` (computed, canonical name returned by the API)
- `type` (required, one of `CHAT`, `COMPLETION`, `EMBEDDING`, per DIAL Core `ModelType`)
- `display_name`, `intro` (optional, computed; DIAL Core models these as `LocalizedValue`, i.e. either a plain string or a locale-keyed map of strings)
- `description`, `endpoint` (optional, computed)
- `user_roles` (optional, computed, set of strings)
- `responses_endpoint`, `display_version`, `icon_url`, `reference` (optional, computed)
- `forward_auth_token` (optional, computed, bool)
- `max_retry_attempts` (optional, computed, int)
- `author` (optional, computed)
- `status` (computed, when returned by API)
- `features` (optional, computed, nested object mapping DIAL model feature flags and endpoints)
- `limits` (optional, computed, nested object: `max_prompt_tokens`, `max_completion_tokens`, `max_total_tokens`)
- `pricing` (optional, computed, nested object: `unit`, `prompt`, `completion`, `cache_read`, `cache_write`; `cache_read`/`cache_write` are DIAL Core `PricingRate` objects (`rate`, `if_true`, `if_false`, `test`), not plain strings)
- `upstreams` (optional, computed, list of nested objects: `endpoint`, `responses_endpoint`, `id`, `extra_data`, `key`, `secret_extra_data`, `tier`, `weight`); `secret_extra_data` is a write-only string per DIAL Core `Upstream`, not a map; `extra_data` is likewise documented as a write-only string in the spec but is modeled here as a map (see Implementation Notes — verified against real API behavior, not just the spec text); `base_url` and `interfaces` are per-upstream fields in DIAL Core's `Upstream` schema and are not implemented (distinct from the model-level `interfaces` below); `key` and `secret_extra_data` are optional, sensitive and not computed: DIAL Core never returns them (`writeOnly`), so state keeps the configured values by list position (an upstream without `key` plans as unset, not "(known after apply)"); after import they are null; `extra_data` is decoded from either an object (DIAL Core 0.48.1) or a string holding a JSON object (newer Core, matching the OpenAPI `string` type); it is always sent as an object, and non-string values are kept as their JSON text.
- `dependencies`, `description_keywords`, `input_attachment_types`, `interceptors`, `fields_hashing_order` (optional, computed, sets of strings)
- `override_name` (optional, computed, string override of the model name presented to clients)
- `default_headers` (optional, computed, map of string to string)
- `interfaces` (optional, computed, map keyed by interface name, e.g. `openaiResponses`/`anthropicMessages`; only `base_url` is modeled per interface — DIAL Core's `DeploymentInterface` also allows `defaultHeaders`, `mode`, `translator`, `defaults`, and a nested `features` block per interface, none of which are implemented)
- `defaults` (optional, free-form object, a Terraform dynamic attribute): sent to DIAL Core as-is, matching the API's `MapStringObject` (e.g. `max_tokens`, `custom_fields`, `tools`). Keys inside it are not renamed. It must be an object. On read the configured value is kept when the API returns equivalent JSON; otherwise the API value is stored (objects as objects, arrays as tuples). An unset `defaults` stays null when the API returns `{}` or omits it, and a configured `{}` stays `{}`. Schema version 1: state written by schema version 0 (where `defaults` only modeled `tools = [{ type, static_function = { name } }]`) is upgraded automatically to the same JSON
- `tokenizer_model` (optional, computed, string; sent as `tokenizerModel`): model whose tokenizer DIAL Core uses for token counting, e.g. `gpt-4-0314`. Also exposed by the `aidial_file_model` data source
- `max_input_attachments` (optional, computed, integer; sent as `maxInputAttachments`): maximum number of attachments in one request. Also exposed by the `aidial_file_model` data source
- `catalog_schema_id` (optional, string; sent as `catalogSchemaId`): URI of the catalog schema (see `aidial_catalog_schema`) describing `catalog_properties`
- `catalog_properties` (optional, free-form object, a Terraform dynamic attribute; sent as `catalogProperties`): handled like `defaults` (arbitrary JSON values, keys not renamed, must be an object, configured value kept when the API returns equivalent JSON, unset stays null for `{}`/omitted)
- `embedding_dimensions`, `base_url` (model-level), `responses_defaults` (optional, computed; present in DIAL Core `Model` and currently unmapped)
- `created_at`, `updated_at` (computed; present in DIAL Core `Model` and currently unmapped)

`id` SHALL be composed as `<bucket>/<path>`.

#### Scenario: Resource registration
- **WHEN** provider resources are registered
- **THEN** `aidial_model` is included in the resource list

#### Scenario: Immutable identity attributes
- **WHEN** `bucket` or `path` changes
- **THEN** Terraform plans replacement

#### Scenario: Stable resource ID
- **WHEN** model exists remotely
- **THEN** `id` is persisted as `<bucket>/<path>` across refreshes

#### Scenario: Identity attributes required
- **WHEN** `bucket` or `path` is omitted from configuration
- **THEN** validation fails at plan time

Name handling: DIAL Core addresses a model only by `bucket` and `path` and ignores the `name` in the request body, so `name` carries no identity. When `name` is configured the provider sends it and keeps the configured value in state; when omitted the provider sends none, and `name` is taken from the API response on create, then kept stable across plans.

#### Scenario: Name omitted
- **WHEN** a config sets `bucket` and `path` but not `name`
- **THEN** the model is created, no `name` is sent, and `name` in state equals `api_name`

#### Scenario: Name configured
- **WHEN** a config sets `name`
- **THEN** `name` is sent and the configured value is kept in state, whatever the API returns

#### Scenario: Name added or removed later
- **WHEN** `name` is added to, changed in or removed from a config with `bucket` and `path` unchanged
- **THEN** the plan updates in place or shows no change, and never plans replacement
