# translator-resource Specification

## Purpose

Manage DIAL Core translators (`/v1/translators/{bucket}/{path}`) through the `aidial_translator` resource and data source, so they can be created, imported and looked up from Terraform.

## ADDED Requirements

### Requirement: Translator client methods

`internal/client/translator.go` SHALL provide `GetTranslator(ctx, bucket, path)`, `SaveTranslator(ctx, bucket, path, translator)` and `DeleteTranslator(ctx, bucket, path)` against `/v1/translators/{bucket}/{path}`, following the existing client conventions. The body SHALL use the camelCase JSON keys of the OpenAPI `Translator` schema (`baseUrl`, `in`, `out`); fields Core returns beyond those, such as `name` and `status`, SHALL be ignored.

#### Scenario: Request conventions
- **WHEN** any translator method calls the API
- **THEN** it sends the `Api-Key` header and does not use ETag, `If-Match` or `If-None-Match` headers

#### Scenario: API error
- **WHEN** a request returns a non-2xx status other than the GET/DELETE 404 cases
- **THEN** the error includes the status and response body

#### Scenario: PUT response
- **WHEN** `SaveTranslator` receives a successful response
- **THEN** the body is treated as `{"name": ...}` only

#### Scenario: CamelCase body
- **WHEN** a translator with `base_url`, `in` and `out` is saved
- **THEN** the request body uses the keys `baseUrl`, `in` and `out`

#### Scenario: Translator not found
- **WHEN** `GetTranslator` receives `404`
- **THEN** it returns `nil, nil`

#### Scenario: Idempotent delete
- **WHEN** `DeleteTranslator` receives `404`
- **THEN** it returns success

### Requirement: aidial_translator identity

The provider SHALL expose resource `aidial_translator` with `id` (computed, `<bucket>/<path>`), `bucket` (required, only `platform`, replace on change) and `path` (required, non-empty, replace on change). See `resource-identity`.

#### Scenario: Resource registration
- **WHEN** provider resources are registered
- **THEN** `aidial_translator` is included

#### Scenario: Unsupported bucket
- **WHEN** `bucket` is not `platform`
- **THEN** validation fails at plan time

#### Scenario: Empty or omitted identity
- **WHEN** `bucket` or `path` is omitted, or `path` is an empty string
- **THEN** validation fails at plan time

#### Scenario: Immutable identity
- **WHEN** `bucket` or `path` changes
- **THEN** Terraform plans replacement

### Requirement: aidial_translator attributes

The resource SHALL expose `base_url`, `in` and `out`, all required strings. `in` and `out` SHALL accept only `openaiChatCompletions`, `openaiEmbeddings`, `openaiResponses` and `anthropicMessages`. The resource SHALL NOT expose `name` or `api_name`.

#### Scenario: Valid translator
- **WHEN** a config sets `bucket`, `path`, `base_url`, `in = "openaiChatCompletions"` and `out = "anthropicMessages"`
- **THEN** the translator is created with exactly those three fields in the body, and a re-plan shows no change

#### Scenario: Required attribute omitted
- **WHEN** `base_url`, `in` or `out` is omitted
- **THEN** validation fails at plan time

#### Scenario: Unknown interface type
- **WHEN** `in` or `out` is not one of the four interface types
- **THEN** validation fails at plan time and names the accepted values

### Requirement: Translator create, read, update, delete

Create and update SHALL PUT the translator and then re-read it with GET to obtain state. A GET `404` during refresh SHALL remove the resource from state. Delete SHALL treat `404` as success. Update SHALL be in place for any attribute other than `bucket` and `path`.

#### Scenario: Create reads back
- **WHEN** a translator is created
- **THEN** state reflects the GET result

#### Scenario: Drift removal
- **WHEN** the translator no longer exists in Core during refresh
- **THEN** it is removed from state

#### Scenario: In-place update
- **WHEN** `base_url`, `in` or `out` changes
- **THEN** the translator is updated in place with PUT

#### Scenario: Delete already gone
- **WHEN** delete receives `404`
- **THEN** it succeeds

### Requirement: Translator state consistency

After apply, state SHALL equal the planned values for every attribute. Where Core returns a configured attribute empty or omits it on read, the read SHALL keep the value as Core returned it rather than invent one, so drift is visible in the next plan.

#### Scenario: Changed out of band
- **WHEN** Core returns a different `base_url` than the one in state during refresh
- **THEN** state holds Core's value and the next plan shows the change

### Requirement: Translator import

The resource SHALL support import by `<bucket>/<path>`, taking both attributes from the import id, and then read the translator.

#### Scenario: Import
- **WHEN** `terraform import aidial_translator.x platform/openai-to-claude` runs
- **THEN** `bucket` is `platform`, `path` is `openai-to-claude` and the attributes are read from Core

#### Scenario: Malformed import id
- **WHEN** the import id has no `/` or an empty bucket or path
- **THEN** import fails with a diagnostic naming the expected format

### Requirement: aidial_translator data source

The provider SHALL expose data source `aidial_translator` with `bucket` (required, only `platform`) and `path` (required, non-empty), computed `id`, `base_url`, `in` and `out`. A translator that Core does not have SHALL produce an error diagnostic naming the bucket and path. File-sourced translators are the job of `aidial_file_translator`.

#### Scenario: Data source registration
- **WHEN** provider data sources are registered
- **THEN** `aidial_translator` is included

#### Scenario: Existing translator
- **WHEN** the translator exists
- **THEN** `id`, `base_url`, `in` and `out` are populated, equal to what the resource would hold

#### Scenario: Missing translator
- **WHEN** the API returns 404
- **THEN** the data source fails with a not-found diagnostic

#### Scenario: API error
- **WHEN** the API returns a non-2xx status other than 404
- **THEN** the data source fails with a diagnostic carrying the status and body
