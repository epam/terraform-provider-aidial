# application-resource Specification

## Purpose

Manage DIAL Core applications (`/v1/applications/{bucket}/{path}`) through the `aidial_application` resource and data source, so they can be created, imported and looked up from Terraform.

## Requirements

### Requirement: Application client methods

`internal/client/application.go` SHALL provide `GetApplication(ctx, bucket, path)`, `SaveApplication(ctx, bucket, path, application)` and `DeleteApplication(ctx, bucket, path)`, following the existing client conventions. The body SHALL use the snake_case keys DIAL Core 0.50.1 returns for applications (Core accepts the camelCase names of the OpenAPI document on write, but the provider sends one casing both ways) and omit unset fields. The exceptions are the contents of `mcp` and of each `routes` entry, which Core only accepts and returns in camelCase. The `{path}` keeps its `/` separators, because Core treats them as folders.

#### Scenario: Request conventions
- **WHEN** any application method calls the API
- **THEN** it sends the `Api-Key` header and does not use ETag, `If-Match` or `If-None-Match` headers

#### Scenario: API error
- **WHEN** a request returns a non-2xx status other than the GET/DELETE 404 cases
- **THEN** the error includes the status and response body

#### Scenario: PUT response
- **WHEN** `SaveApplication` receives a successful response
- **THEN** the body is treated as `{"name": ...}` only

#### Scenario: Snake_case body
- **WHEN** an application with `display_name`, `forward_auth_token`, `application_type_schema_id` and `application_properties` is saved
- **THEN** the request body uses exactly those keys

#### Scenario: Nested camelCase
- **WHEN** an application with `mcp` and `routes` is saved
- **THEN** the `mcp` object uses `allowedTools`, `configDelivery`, `forwardPerRequestKey` and `mcpApps.domainOverride`, each route uses `rewritePath`, `maxRetryAttempts` and `attachmentPaths`, and `external_services` keeps snake_case keys

#### Scenario: Nested path
- **WHEN** an application is read with bucket `public` and path `team/app`
- **THEN** the request path is `/v1/applications/public/team/app`, with each segment percent-encoded on its own

#### Scenario: Application not found
- **WHEN** `GetApplication` receives `404`
- **THEN** it returns `nil, nil`

#### Scenario: Idempotent delete
- **WHEN** `DeleteApplication` receives `404`
- **THEN** it returns success

### Requirement: aidial_application identity

The provider SHALL expose resource `aidial_application` with `id` (computed, `<bucket>/<path>`), `bucket` (required, non-empty string, replace on change) and `path` (required, non-empty, replace on change). `bucket` SHALL NOT be restricted to `platform`; DIAL Core decides which buckets it accepts (`platform` and `public` in 0.50.1). See `resource-identity`.

#### Scenario: Non-platform bucket
- **WHEN** `bucket` is `public`
- **THEN** validation passes and requests go to `/v1/applications/public/<path>`

#### Scenario: Empty or omitted identity
- **WHEN** `bucket` or `path` is omitted or an empty string
- **THEN** validation fails at plan time

#### Scenario: Immutable identity
- **WHEN** `bucket` or `path` changes
- **THEN** Terraform plans replacement

### Requirement: Application name handling

`name` SHALL be optional and computed: sent to Core only when configured, otherwise derived by Core and read back. `api_name` SHALL be computed and hold the name Core returns as is (for the `public` bucket Core returns `applications/public/<path>`). `author`, `created_at` and `updated_at` SHALL NOT be exposed.

#### Scenario: Name omitted
- **WHEN** `name` is not configured
- **THEN** no name is sent, and `name` in state equals `api_name`

### Requirement: aidial_application attributes

The resource SHALL expose the optional attributes `endpoint`, `description`, `display_name`, `display_version`, `icon_url`, `editor_url`, `viewer_url`, `application_type_schema_id`, `catalog_schema_id`, `forward_auth_token`, `max_retry_attempts`, `user_roles`, `interceptors`, `default_headers`, `features`, `interfaces`, `defaults`, `application_properties`, `catalog_properties`, `input_attachment_types`, `description_keywords`, `dependencies`, `max_input_attachments`, `responses_endpoint`, `responses_defaults`, `intro`, `reference`, `override_name`, `base_url`, `allow_user_external_services`, `mcp`, `routes` and `external_services`. `features` and `interfaces` have the same shape as on `aidial_model`, and the free-form objects (`defaults`, `responses_defaults`, `application_properties`, `catalog_properties`) are dynamic attributes.

#### Scenario: Minimal endpoint application
- **WHEN** a config sets only `bucket`, `path` and `endpoint`
- **THEN** the application is created with just that body, and every other attribute reads back as computed or null

#### Scenario: Schema-based application
- **WHEN** a config sets `application_type_schema_id` and `application_properties` and no `endpoint`
- **THEN** the application is created and `endpoint` reads back as null

#### Scenario: Endpoint and schema together
- **WHEN** both `endpoint` and `application_type_schema_id` are set
- **THEN** validation fails at plan time, because Core rejects the combination

#### Scenario: Neither endpoint nor schema
- **WHEN** none of `endpoint`, `application_type_schema_id`, `mcp` and `interfaces` is set
- **THEN** validation fails at plan time, because Core rejects an application without an endpoint, MCP endpoint, function or interface

#### Scenario: MCP-only application
- **WHEN** a config sets `mcp` with an `endpoint` and sets neither `endpoint` nor `application_type_schema_id`
- **THEN** validation passes and the application is created

#### Scenario: Features and interfaces reuse
- **WHEN** `features` or `interfaces` is configured on an application and on a model with the same values
- **THEN** both are validated, sent and read back identically

#### Scenario: Free-form values keep their form
- **WHEN** Core returns JSON equal to the configured `defaults`, `responses_defaults`, `application_properties` or `catalog_properties`
- **THEN** the configured value and its types are kept in state

#### Scenario: Simple attributes round trip
- **WHEN** a config sets `input_attachment_types`, `description_keywords`, `dependencies`, `max_input_attachments`, `allow_user_external_services` and `override_name`
- **THEN** they are sent in the PUT and read back with the configured values

### Requirement: Attributes DIAL Core fills in

Attributes that Core fills in when a request leaves them out SHALL be optional and computed, keep their value across updates, and not be sent until known. These are `forward_auth_token` (false), `max_retry_attempts` (1), `interceptors`, `dependencies` and `description_keywords` (empty), `routes` (empty) and, once `mcp` is set, its `config_delivery` (`META`), `forward_per_request_key` (true) and `allowed_tools` (empty). `input_attachment_types`, `max_input_attachments` and `allow_user_external_services` SHALL stay plain optional, because Core leaves them unset.

#### Scenario: Minimal config after apply
- **WHEN** a config sets only `bucket`, `path` and `endpoint`
- **THEN** a second plan shows no changes

#### Scenario: Partial mcp after apply
- **WHEN** a config sets `mcp` with only `endpoint` and `transport`
- **THEN** `config_delivery` reads back as `META`, `forward_per_request_key` as true and `allowed_tools` as empty, and a second plan shows no changes

#### Scenario: Unset optional attributes
- **WHEN** a config omits `input_attachment_types`, `max_input_attachments` and `allow_user_external_services`
- **THEN** they read back as null and a second plan shows no changes

### Requirement: Application state consistency

Create and update SHALL PUT then re-read via GET to populate state. Out-of-band deletion SHALL remove the resource from state on refresh. Absent nested objects, lists and sets SHALL read back as explicit nulls, while configured empty collections stay empty.

#### Scenario: Create re-reads
- **WHEN** an application is created
- **THEN** state reflects the GET response, not the PUT response

#### Scenario: Out-of-band deletion
- **WHEN** the application no longer exists on refresh
- **THEN** it is removed from state

### Requirement: Application import

`terraform import` SHALL accept `<bucket>/<path>` and reject any other form with an error naming the expected format.

#### Scenario: Malformed import id
- **WHEN** the import id has no `/` separator or an empty part
- **THEN** import fails with a format error

### Requirement: aidial_application data source

The data source `aidial_application` SHALL take `bucket` and `path` (required, same validation as the resource) and expose the resource's attributes as computed. A missing application SHALL be an error naming bucket and path.

#### Scenario: Found
- **WHEN** the application exists
- **THEN** its attributes equal those the resource would hold

#### Scenario: Not found
- **WHEN** Core returns 404
- **THEN** the data source fails with an error naming bucket and path

### Requirement: Application mcp attribute

`mcp` SHALL be an optional single nested attribute with `endpoint`, `transport`, `allowed_tools` (set of strings), `config_delivery` (`HEADER` or `META`), `forward_per_request_key` (bool) and `mcp_apps` (object with `domain_override`). `transport` SHALL accept the values the `aidial_toolset` resource accepts. Removing `mcp` from the config SHALL remove it from the application, and an application without `mcp` SHALL read back `mcp` as null.

#### Scenario: Invalid config_delivery
- **WHEN** `config_delivery` is neither `HEADER` nor `META`
- **THEN** validation fails at plan time

#### Scenario: Invalid transport
- **WHEN** `transport` is not a value the toolset resource accepts
- **THEN** validation fails at plan time

#### Scenario: Unset mcp
- **WHEN** an application has no `mcp` and Core returns none
- **THEN** `mcp` is null in state

#### Scenario: Remove mcp
- **WHEN** `mcp` is removed from a config that had it
- **THEN** the update sends no `mcp` and the attribute reads back as null

### Requirement: Application routes attribute

`routes` SHALL be an optional map from route name to a route body with `order`, `methods`, `paths`, `permissions`, `user_roles`, `max_retry_attempts`, `rewrite_path`, `response`, `attachment_paths` and `upstreams`, in the shapes `aidial_route` uses. It SHALL NOT have `id`, `name` or `status`; the map key is the route name. `order`, `permissions` and `upstreams` SHALL be computed when Core fills them in. The schema SHALL be the application's own and SHALL NOT change `aidial_route`.

#### Scenario: Route with static response
- **WHEN** a config sets `routes` with a route that has `paths`, `methods` and a `response`
- **THEN** the application is created with that route and reads it back, with Core's defaults for the fields left out

#### Scenario: No routes configured
- **WHEN** `routes` is omitted
- **THEN** it reads back as an empty map and a second plan shows no changes

#### Scenario: Route removed from the map
- **WHEN** a route is removed from `routes`
- **THEN** the update sends the map without it

### Requirement: Application external_services attribute

`external_services` SHALL be an optional map from service name to `display_name`, `description` and `auth_settings`. `auth_settings` SHALL hold `authentication_type`, `client_id`, `client_secret`, `authorization_endpoint`, `token_endpoint`, `scopes_supported`, `api_key_header` and `code_verifier`. Runtime fields Core computes (`client_secret_hint`, `user_level_auth_status`, `app_level_auth_status`) SHALL NOT be exposed.

#### Scenario: Runtime fields hidden
- **WHEN** Core returns `client_secret_hint` and the auth status fields
- **THEN** they are ignored and cause no plan diff

### Requirement: External service secrets

`client_secret` and `code_verifier` SHALL be sensitive. Since Core never returns them, the prior state value SHALL be kept on read, and `client_secret` SHALL be at least 8 characters when set.

#### Scenario: Secret kept after apply
- **WHEN** an application is created with `client_secret` set and read back
- **THEN** state holds the configured secret and a second plan shows no changes

#### Scenario: Secret changed
- **WHEN** `client_secret` changes in the config
- **THEN** the update sends the new value

#### Scenario: Short secret
- **WHEN** `client_secret` is shorter than 8 characters
- **THEN** validation fails at plan time

#### Scenario: Invalid combination rejected by Core
- **WHEN** `code_verifier` is set with `authentication_type` `OAUTH`
- **THEN** Core rejects the PUT and the provider reports Core's error

#### Scenario: Import
- **WHEN** an application with external services is imported
- **THEN** `client_secret` and `code_verifier` are null and the next plan shows the diff

### Requirement: Unmanaged application fields

Fields the resource does not model, such as `function`, SHALL be dropped when the resource saves an application, because Core replaces the whole document on PUT. The resource description and docs SHALL say this and that `function` applications are not to be managed with this resource. The provider SHALL NOT merge unmodelled fields from the stored document.

#### Scenario: Description states the limit
- **WHEN** the generated docs for `aidial_application` are read
- **THEN** they name `function` as unmanaged and say it is dropped on apply

### Requirement: Application data source external services

The `aidial_application` and `aidial_file_application` data sources SHALL expose `mcp`, `routes`, `external_services` and the new simple attributes as computed. Because Core never returns them, `client_secret` and `code_verifier` SHALL be null on the data sources.

#### Scenario: Data source with external service
- **WHEN** the data source reads an application with an external service
- **THEN** the service's `display_name`, `description` and non-secret `auth_settings` are populated and `client_secret` is null

### Requirement: Stable api_name

`api_name` SHALL keep its state value across updates, because it is derived from `bucket` and `path`, which force replacement. A plan that changes another attribute SHALL NOT list `api_name` as unknown.

#### Scenario: Unrelated update
- **WHEN** an attribute other than `bucket` or `path` changes
- **THEN** the plan keeps `api_name` as in state
