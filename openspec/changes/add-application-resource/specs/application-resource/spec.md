# application-resource Specification

## Purpose

Manage DIAL Core applications (`/v1/applications/{bucket}/{path}`) through the `aidial_application` resource and data source, so they can be created, imported and looked up from Terraform.

## ADDED Requirements

### Requirement: Application client methods

`internal/client/application.go` SHALL provide `GetApplication(ctx, bucket, path)`, `SaveApplication(ctx, bucket, path, application)` and `DeleteApplication(ctx, bucket, path)`, following the existing client conventions. The body SHALL use the snake_case keys DIAL Core 0.50.1 returns for applications (Core accepts the camelCase names of the OpenAPI document on write, but the provider sends one casing both ways) and omit unset fields. The `{path}` keeps its `/` separators, because Core treats them as folders.

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

The resource SHALL expose the optional attributes `endpoint`, `description`, `display_name`, `display_version`, `icon_url`, `editor_url`, `viewer_url`, `application_type_schema_id`, `catalog_schema_id`, `forward_auth_token`, `max_retry_attempts`, `user_roles`, `interceptors`, `default_headers`, `features`, `interfaces`, `defaults`, `application_properties` and `catalog_properties`. `features` and `interfaces` have the same shape as on `aidial_model`, and the free-form objects are dynamic attributes.

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
- **WHEN** neither `endpoint` nor `application_type_schema_id` is set
- **THEN** validation fails at plan time, because Core rejects an empty application

#### Scenario: Features and interfaces reuse
- **WHEN** `features` or `interfaces` is configured on an application and on a model with the same values
- **THEN** both are validated, sent and read back identically

#### Scenario: Free-form values keep their form
- **WHEN** Core returns JSON equal to the configured `defaults`, `application_properties` or `catalog_properties`
- **THEN** the configured value and its types are kept in state

### Requirement: Attributes DIAL Core fills in

Attributes that Core fills in when a request leaves them out (`forward_auth_token` false, `max_retry_attempts` 1 and `interceptors` empty) SHALL be optional and computed, keep their value across updates, and not be sent until known.

#### Scenario: Minimal config after apply
- **WHEN** a config sets only `bucket`, `path` and `endpoint`
- **THEN** a second plan shows no changes

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
