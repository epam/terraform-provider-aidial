# Design

## Context

See proposal.md for motivation. Current state: `client.Application` and `applicationBody` model the snake_case top-level fields; the shared `features` and `interfaces` mapping lives in `deployment_shared.go`; `aidial_toolset` already maps `dependencies`, `description_keywords` and `input_attachment_types` (with a `defaultedSet` helper for Core-filled sets); `aidial_route` maps the route body in `route_resource.go`.

Probe findings against Core 0.50.1 (PUT then GET, throwaway `platform` and `public` applications):

- A PUT that omits fields resets them: Core replaces the whole document. `_metadata` is rejected on write and absent on read for applications, and `function` is rejected in `platform` and needs a real source folder elsewhere. Nothing hidden preserves unmodelled fields.
- `mcp` is accepted and returned in camelCase only (snake_case `forward_per_request_key` is rejected). Core fills `configDelivery: META`, `forwardPerRequestKey: true`, `allowedTools: []` and `mcpApps: null`.
- `routes` entries are camelCase and Core fills `upstreams: []`, `order`, `permissions: []` and `attachmentPaths`. `routes` reads `{}` when unset, as `dependencies` and `description_keywords` read `[]`.
- `external_services` is snake_case. Core never returns `client_secret`, only `client_secret_hint` plus the runtime `user_level_auth_status` and `app_level_auth_status`. It requires `client_secret` of at least 8 characters and forbids `code_verifier` for `OAUTH`.
- `input_attachment_types`, `max_input_attachments` and `allow_user_external_services` stay absent or null when unset.

## Goals / Non-Goals

**Goals:**
- Every application field users reported, plus the other modelled-by-Core simple fields, round trips without plan churn.
- Existing state and configs keep working unchanged.

**Non-Goals:**
- Managing `function` applications, or preserving unmodelled fields by read-modify-write.
- Changing `aidial_route` or sharing code with it.
- Exposing runtime-only external service status.

## Decisions

1. **Typed nested attributes for `mcp`, `routes` and `external_services`** over dynamic JSON. Plan-time validation and generated docs outweigh the extra mapping code, and the shapes are small and stable. Alternative: free-form dynamic attribute like `defaults`; rejected because typos would reach Core.
2. **Copy the route body schema into the application** instead of sharing it with `aidial_route`. The application entries have no `id`, `name` or `status`, and a shared type would couple two resources that evolve separately. Cost: some duplicated schema and mapping, accepted. Route `upstreams[].key` and `secret_extra_data` follow the same retention rules the route resource uses.
3. **Optional+Computed with `UseStateForUnknown` only where Core fills a value**, plain Optional elsewhere, so null stays null for fields Core leaves unset. Nested `mcp` fields are computed individually, which keeps `mcp = { endpoint = ..., transport = ... }` free of churn.
4. **External service secrets follow `aidial_key`**: sensitive, prior state value kept on read-back, null after import. Only `client_secret` length is validated at plan time; Core's `code_verifier` rule is left to Core so the two cannot drift.
5. **Wire casing handled in the client types** with explicit JSON tags: top level snake_case, `mcp` and route bodies camelCase, `external_services` snake_case. The spec states the exception so tests pin it.
6. **Unmanaged fields stay dropped and documented.** Merging the stored raw document on write would make drift semantics unclear, and `function` applications are not suitable for Terraform. The resource description gets an explicit sentence.
7. **Plan-time source check replaces `ExactlyOneOf`.** Testing real application configs against Core showed that an application with only an `mcp` endpoint is valid (Core requires an endpoint, MCP endpoint, function or interface). So the check is now "not both `endpoint` and `application_type_schema_id`" plus "at least one of `endpoint`, `application_type_schema_id`, `mcp`, `interfaces`", in `ValidateConfig`. Unknown values count as set.
8. **Runtime and unknown nested attributes stay unexposed.** `user_level_auth_status` and `app_level_auth_status` of an external service are run-time state: Core ignores `app_level_auth_status` on write (a PUT with `SIGNED_IN` reads back `SIGNED_OUT`). Terraform silently drops unknown attributes of an object literal such as these, and a `name` inside a route, so the docs tell users to leave them out.
9. **Name mapping:** Terraform names are snake_case versions of the Core names (`responses_endpoint`, `override_name`, `base_url`, `mcp_apps.domain_override`).

## Risks / Trade-offs

- [Route upstream `key` may not be returned by Core in an application, causing a diff] → Verify against the local Core in the acceptance test and keep the prior value if masked, as the route resource does.
- [Core may add computed fields to `mcp` or route entries in later versions] → Unknown JSON keys are ignored on read; new computed fields would only show up once the schema is extended.
- [`routes` Optional+Computed means removing the block from config cannot clear routes] → Set `routes = {}` to clear; document it.
- [Copied route schema can drift from `aidial_route`] → Acceptable; the spec pins the application's own shape and a test checks it round trips against Core.
- [`reference` and `override_name` may be rejected or rewritten by Core in some buckets] → Cover with the acceptance test and mark as computed if Core generates a value (observed: `reference` is generated when unset in `platform`).

## Migration Plan

Additive schema only. Existing applications read back the new attributes as Core defaults (`[]`, `{}`, null) that are Computed or null, so the first plan after upgrading shows no diff. Release as 0.3.2; rollback is installing 0.3.1, which ignores the new attributes (and drops them again on its next PUT).
