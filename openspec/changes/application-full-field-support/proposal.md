# Proposal

## Why

`aidial_application` (0.3.0, 0.3.1) models only part of DIAL Core's Application. A PUT replaces the whole document, so fields the resource does not model are rejected by Terraform as unsupported arguments, or silently dropped when the application is managed some other way and then applied. Users cannot describe applications that use `input_attachment_types`, `description_keywords`, `dependencies`, `mcp`, `routes` or `external_services` through Terraform. This was verified against DIAL Core 0.50.1: a PUT that omits these fields resets them to Core defaults.

## What Changes

- `aidial_application`, and the `aidial_application` and `aidial_file_application` data sources, gain the optional attributes `input_attachment_types`, `description_keywords`, `dependencies`, `max_input_attachments`, `responses_endpoint`, `responses_defaults`, `intro`, `reference`, `override_name`, `base_url` and `allow_user_external_services`.
- New typed attribute `mcp` (`endpoint`, `transport`, `allowed_tools`, `config_delivery`, `forward_per_request_key`, `mcp_apps.domain_override`).
- New typed map attribute `routes`: route name to a route body (the application's own copy of the route schema, without `id` and `name`).
- New typed map attribute `external_services`: name to `display_name`, `description` and `auth_settings`. `client_secret` and `code_verifier` are sensitive and kept from prior state, because Core never returns them.
- The client sends and reads `mcp` and `routes` with camelCase keys inside the otherwise snake_case body, as Core requires.
- The resource description and docs state that `function` applications and any other field not modelled are still dropped by a PUT, so `function` applications must not be managed with this resource.
- Purely additive; no change to existing state. Released as 0.3.2.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `application-resource`: new attributes and their defaults, the camelCase exception in the client body, nested `mcp`, `routes` and `external_services` behavior including secret handling, and the documented set of unmanaged fields. The `aidial_application` data source and `file-application-data-source` already expose "the resource's attributes", so they pick up the new attributes without a separate delta.

## Impact

- `internal/client/application.go` (struct fields and nested types), `internal/provider/application_resource.go`, `application_data_source.go` and the file application data source mapping.
- Tests: client request-body tests, provider unit tests including null-versus-empty and secret retention, and an acceptance test against the local Core.
- `docs/` regenerated, a new `examples/` snippet, a CHANGELOG entry and `openspec/specs/application-resource` updated at archive.
- No new dependencies and no change to other resources.
