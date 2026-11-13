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
- `pricing` (optional, computed, nested object: `unit` and the rates `prompt`, `completion`, `cache_read`, `cache_write`). Each rate is an optional, computed nested object mapping DIAL Core's `PricingRate`: `rate` (flat price string), or `test` (`field`, `operator` one of `==`, `!=`, `>`, `<`, `>=`, `<=`, `value`, all required) with `if_true` and `if_false` (flat price strings). `rate` conflicts with `test`/`if_true`/`if_false`; `test`, `if_true` and `if_false` require each other (plan-time validation). Terraform schemas cannot nest recursively, so a condition inside `if_true`/`if_false` is not modeled; reading one from Core fails with a clear error instead of dropping it. Plain strings are no longer accepted for a rate (breaking change; write `prompt = { rate = "0.000002" }`).
- `upstreams` (optional, computed, list of nested objects: `endpoint`, `responses_endpoint`, `id`, `extra_data`, `key`, `secret_extra_data`, `tier`, `weight`, `base_url`, `auth_type` (`API_KEY`, `BEARER` or `CUSTOM_HEADER`), `auth_header_name` (required with `CUSTOM_HEADER`, not allowed otherwise)); `secret_extra_data` is a write-only string per DIAL Core `Upstream`, not a map; `extra_data` is likewise documented as a write-only string in the spec but is modeled here as a map (see Implementation Notes — verified against real API behavior, not just the spec text); per-upstream `interfaces` is a field in DIAL Core's `Upstream` schema and is not implemented (distinct from the model-level `interfaces` below); `key` and `secret_extra_data` are optional, sensitive and not computed: DIAL Core never returns them (`writeOnly`), so state keeps the configured values by list position (an upstream without `key` plans as unset, not "(known after apply)"); after import they are null; `extra_data` is decoded from either an object (DIAL Core 0.48.1) or a string holding a JSON object (newer Core, matching the OpenAPI `string` type); it is always sent as an object, and non-string values are kept as their JSON text.
- `dependencies`, `description_keywords`, `input_attachment_types`, `interceptors`, `fields_hashing_order` (optional, computed, sets of strings)
- `override_name` (optional, computed, string override of the model name presented to clients)
- `base_url` (optional, string; model-level `baseUrl` that completes `interfaces` entries without their own `base_url`)
- `embedding_dimensions` (optional, integer; sent as `embeddingDimensions`): dimensions of the embeddings returned by an embedding model
- `responses_defaults` (optional, free-form object, a Terraform dynamic attribute; sent as `responsesDefaults`): same semantics as `defaults`, applied to Responses API requests
- `created_at`, `updated_at` (computed, integer Unix epoch as reported by DIAL Core; `created_at` keeps its prior value across plans)
- `default_headers` (optional, computed, map of string to string)
- `interfaces` (optional, computed, map keyed by DIAL Core interface type; the keys are validated against Core's `InterfaceType` enum (`openaiChatCompletions`, `openaiEmbeddings`, `openaiResponses`, `anthropicMessages`) and sent as written, in Core's camelCase, while the attributes inside each entry are snake_case; each entry models `base_url`, `mode` (`passthrough` or `translator`), `override_paths` (map of string, sent as `overridePaths`) and `default_headers` (map of string, sent as `defaultHeaders`), all optional; DIAL Core's `DeploymentInterface` also allows `translator`, `defaults` and a nested `features` block per interface, which are not implemented)
- `defaults` (optional, free-form object, a Terraform dynamic attribute): sent to DIAL Core as-is, matching the API's `MapStringObject` (e.g. `max_tokens`, `custom_fields`, `tools`). Keys inside it are not renamed. It must be an object. On read the configured value is kept when the API returns equivalent JSON; otherwise the API value is stored (objects as objects, arrays as tuples). An unset `defaults` stays null when the API returns `{}` or omits it, and a configured `{}` stays `{}`. Schema version 1: state written by schema version 0 (where `defaults` only modeled `tools = [{ type, static_function = { name } }]`) is upgraded automatically to the same JSON
- `tokenizer_model` (optional, computed, string; sent as `tokenizerModel`): model whose tokenizer DIAL Core uses for token counting, e.g. `gpt-4-0314`. Also exposed by the `aidial_file_model` data source
- `max_input_attachments` (optional, computed, integer; sent as `maxInputAttachments`): maximum number of attachments in one request. Also exposed by the `aidial_file_model` data source
- `catalog_schema_id` (optional, string; sent as `catalogSchemaId`): URI of the catalog schema (see `aidial_catalog_schema`) describing `catalog_properties`
- `catalog_properties` (optional, free-form object, a Terraform dynamic attribute; sent as `catalogProperties`): handled like `defaults` (arbitrary JSON values, keys not renamed, must be an object, configured value kept when the API returns equivalent JSON, unset stays null for `{}`/omitted)

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

#### Scenario: Flat pricing rate
- **WHEN** a config sets `pricing.prompt = { rate = "0.000002" }`
- **THEN** the rate is sent to DIAL Core as the plain string `"0.000002"` and read back as `rate` with `test`, `if_true` and `if_false` null

#### Scenario: Conditional pricing rate
- **WHEN** a config sets a rate with `test = { field = "promptTokens", operator = ">", value = "100000" }`, `if_true` and `if_false`
- **THEN** DIAL Core receives an object with `test` (a numeric `value` as a JSON number), `ifTrue` and `ifFalse`, and the next plan shows no change

#### Scenario: Invalid pricing rate
- **WHEN** a rate sets `rate` together with `test`, `if_true` or `if_false`, sets `test` without both branches, or sets a branch without `test`, or uses an unknown `operator`
- **THEN** validation fails at plan time

#### Scenario: Nested pricing condition from Core
- **WHEN** DIAL Core returns a condition inside `ifTrue` or `ifFalse`
- **THEN** the read fails with an error naming the attribute instead of dropping the condition
