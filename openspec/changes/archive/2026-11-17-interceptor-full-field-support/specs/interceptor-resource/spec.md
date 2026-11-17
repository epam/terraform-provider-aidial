# Spec Delta

## MODIFIED Requirements

### Requirement: aidial_interceptor attributes

The resource SHALL expose `endpoint` (required) and the optional attributes `description`, `display_name`, `display_version`, `icon_url`, `catalog_schema_id`, `forward_auth_token`, `max_retry_attempts`, `user_roles`, `default_headers`, `features`, `interfaces`, `defaults`, `catalog_properties`, `dependencies`, `description_keywords`, `input_attachment_types`, `max_input_attachments`, `intro`, `reference`, `override_name`, `base_url`, `responses_endpoint`, `responses_defaults` and `interceptors`. `features` and `interfaces` have the same shape as on `aidial_model`, and the free-form objects (`defaults`, `responses_defaults`, `catalog_properties`) are dynamic attributes.

#### Scenario: Minimal interceptor
- **WHEN** a config sets only `bucket`, `path` and `endpoint`
- **THEN** the interceptor is created with just that body, and every other attribute reads back as computed or null

#### Scenario: Endpoint required
- **WHEN** `endpoint` is omitted
- **THEN** validation fails at plan time

#### Scenario: Features and interfaces reuse
- **WHEN** `features` or `interfaces` is configured on an interceptor and on a model with the same values
- **THEN** both are validated, sent and read back identically

#### Scenario: Free-form values keep their form
- **WHEN** Core returns JSON equal to the configured `defaults`, `responses_defaults` or `catalog_properties`
- **THEN** the configured value and its types are kept in state

#### Scenario: New attributes round trip
- **WHEN** a config sets `dependencies`, `description_keywords`, `input_attachment_types`, `max_input_attachments`, `intro`, `reference`, `override_name`, `base_url`, `responses_endpoint` and `interceptors`
- **THEN** they are sent in the PUT with camelCase keys and read back with the configured values

### Requirement: Attributes DIAL Core fills in

`forward_auth_token` (false), `max_retry_attempts` (1), `dependencies`, `description_keywords` and `interceptors` (empty) are filled in by DIAL Core when a request leaves them out, so they SHALL be optional and computed, keep their value across updates, and not be sent until known. `input_attachment_types`, `max_input_attachments`, `intro`, `reference`, `override_name`, `base_url`, `responses_endpoint` and `responses_defaults` SHALL stay plain optional, because Core leaves them unset.

#### Scenario: Minimal config after apply
- **WHEN** a config sets only `bucket`, `path` and `endpoint`
- **THEN** after apply `forward_auth_token` is false, `max_retry_attempts` is 1 and `dependencies`, `description_keywords` and `interceptors` are empty sets, as Core returned, and a re-plan shows no change

#### Scenario: Unset optional attributes
- **WHEN** a config omits `input_attachment_types`, `max_input_attachments`, `reference` and the other plain optional attributes
- **THEN** they read back as null and a re-plan shows no change

#### Scenario: Configured empty set
- **WHEN** `input_attachment_types = []` is configured
- **THEN** it is sent as an empty list and stays an empty set in state

## ADDED Requirements

### Requirement: Unmanaged interceptor fields

Fields the resource does not model, namely `author`, `created_at` and `updated_at`, SHALL NOT be exposed or sent. The resource description SHALL say that fields it does not model are dropped when the interceptor is saved, because Core replaces the whole document on PUT.

#### Scenario: Description states the limit
- **WHEN** the generated docs for `aidial_interceptor` are read
- **THEN** they say that unmodelled fields are dropped on apply
