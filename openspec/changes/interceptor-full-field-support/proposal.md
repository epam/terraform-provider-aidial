# Proposal

## Why

`aidial_interceptor` models only part of DIAL Core's Interceptor. Core accepts and stores more fields, and a PUT replaces the whole document, so anything the resource does not model is lost on every apply. This was verified against Core 0.50.1: `dependencies`, `descriptionKeywords`, `inputAttachmentTypes`, `maxInputAttachments`, `intro`, `reference`, `overrideName`, `baseUrl`, `responsesEndpoint`, `responsesDefaults` and `interceptors` are all accepted and read back, but Terraform rejects them as unsupported arguments. The same gap was just closed for `aidial_application` (see `application-full-field-support`).

## What Changes

- `aidial_interceptor`, and the `aidial_interceptor` and `aidial_file_interceptor` data sources, gain the optional attributes `dependencies`, `description_keywords`, `input_attachment_types`, `max_input_attachments`, `intro`, `reference`, `override_name`, `base_url`, `responses_endpoint`, `responses_defaults` and `interceptors`.
- `dependencies`, `description_keywords` and `interceptors` are optional and computed, because Core reads them back as empty lists when unset. The other new attributes stay plain optional, because Core leaves them null.
- `author`, `created_at` and `updated_at` stay unexposed: Core manages them.
- The resource description states that fields it does not model are dropped on apply.
- Purely additive; no change to existing state. Released in 0.3.2 together with `application-full-field-support`.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `interceptor-resource`: the attribute list, and the set of attributes Core fills in. The `aidial_interceptor` data source and `file-interceptor-data-source` already expose "every attribute the resource exposes", so they follow without a separate delta.

## Impact

- `internal/client/interceptor.go`, `internal/provider/interceptor_resource.go` and `interceptor_data_source.go`.
- Tests: client body test, provider unit tests (round trip, defaults, null-versus-empty, data sources) and an acceptance test against the local Core.
- `docs/` regenerated, `examples/` updated, one CHANGELOG entry in the existing "Unreleased" section, and `openspec/specs/interceptor-resource` updated at archive.
- No new dependencies and no change to other resources.
