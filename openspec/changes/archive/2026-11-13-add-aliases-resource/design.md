# Design

## Context

See proposal.md for motivation. Observed state:

- `aidial_interceptor` is the closest template: an API-managed resource with the generic `getResource`, `saveResource` and `deleteResource` client helpers, an API data source and a `file_` data source. Its client lives in `internal/client/interceptor.go`.
- In `openapicore/open_api_core.yaml`, `Alias` is camelCase with no `required` list and carries the whole deployment field set. `presentAs` is `AliasPresentation` (`model`, `application`, `toolset`); `target` is `AliasTarget` with `id` and `version` of no declared type and `path` a string; `displayName`, `description` and `intro` are `LocalizedValue` (string or locale map).
- The path description says `bucket` "Must be public", and the PUT 400 text lists the accepted fields as `presentAs`, `target`, `userRoles` and the display fields, and rejects a "target that does not fit presentAs".
- Endpoints are marked `x-preview`: `GET/PUT/DELETE /v1/aliases/{bucket}/{path}` and the read-only `GET /v1/admin/config/file/aliases/{name}`. GET returns `status` (`valid` or `invalid`) and may carry `validationWarnings`.

## Goals / Non-Goals

**Goals:**
- An alias behaves like the other API-managed resources: same identity, import, read-after-write, 404 and null-vs-empty rules.

**Non-Goals:**
- Validating that `target` refers to an existing deployment, or adding a reference between resources.
- The `Alias` fields listed as out of scope in proposal.md.

## Decisions

1. **Generic endpoint `/v1/aliases/{bucket}/{path}`** with the generic helpers, `bucket` and `path` percent-encoded as single segments like roles and models. Revisit only if the real-Core check (task 1.1) shows Core needs raw slashes, as it did for toolsets.
2. **Bucket is `public` or `platform`.** The OpenAPI text says `public`, but a real Core accepts both and 404s any other bucket (see the findings), so the validator is `OneOf("public", "platform")` and both are real use cases.
3. **Wire format is camelCase**, as in the OpenAPI document. The toolset change found the document wrong for toolsets, so task 1.1 verifies it before the client is written.
4. **Attribute subset.** Exposed: `present_as`, `target_path`, `user_roles`, `display_name`, `description`, `display_version`, `icon_url`, `intro`, `description_keywords` (a set, as on models). Everything else is ignored on read, so a future Core field or an out-of-band value causes no drift.
5. **`present_as`** is a required string validated with `OneOf("model", "application", "toolset")`.
6. **`target_path`** is a required string sent as `target.path`. Core rejects `target.id` and `target.version`, so a nested object would only expose fields that always fail. Validators: a regex for `models/`, `applications/` or `toolsets/` plus a name, and a cross-attribute check that `toolset` needs a `toolsets/` path while `model` and `application` must not have one. Other combinations (such as nonexistent targets) are left to Core.
7. **Display fields as plain strings.** `display_name` and `description` are `LocalizedValue` in the API. The provider sends and reads the plain string form; Core accepts a locale map on write but returns a plain string (the probe), so the read never sees the map form.
8. **Names.** `name` is optional and computed and sent only when configured, with `api_name` holding Core's answer, as in `make-name-optional`.
9. **Server-owned fields.** `author`, `createdAt`, `updatedAt` and `status` are not modelled. Alias `status`/`validationWarnings` is a read-time evaluation of the target, so exposing it would make plans depend on the target's current state.
10. **No Core-defaulted attributes.** Core fills in only `name` and `status`, so nothing is computed beyond `api_name`.
11. **File data source** mirrors `aidial_file_interceptor`: lookup by `name` only, the same attribute set minus identity, error on 404.
12. **Spec placement.** New capabilities `alias-resource` and `file-alias-data-source`; identity, provider-identity and encoding changes are ADDED requirements in the existing capabilities, so no existing requirement text is rewritten.

## Findings from a real Core (task 1.1, Core 0.50.1)

Probed against the local Core with throwaway aliases, an application, a toolset and a model, all deleted afterwards. The decisions and specs above were updated to match.

- **Buckets: `public` and `platform` both work** (PUT and GET); any other bucket (`foo`) is a 404. The OpenAPI text "Must be public" is not enforced, which contradicts decision 2.
- **`target` has only `path`.** `target.id` and `target.version` fail with `Failed to parse Alias at "target.id": unrecognized field` (same for `version`), whatever their type. `path` is required (`{}` and a missing `target` are a 400 with `validationWarnings`) and must be `models/<name>`, `applications/<name>` or `toolsets/<name>` (or "a resource url"; a full `http://.../v1/applications/...` URL and `models/platform/<name>` were rejected).
- **`present_as` and the target kind must fit** (400 with `validationWarnings`): `model` accepts `applications/x` and `models/x` but not `toolsets/x`; `application` accepts `applications/x` and `models/x` but not `toolsets/x`; `toolset` accepts only `toolsets/x`.
- **The target need not exist**: `applications/nonexistent` is accepted and GET reports `status: "valid"`.
- **Wire format is camelCase** (`presentAs`, `userRoles`, `displayName`, `displayVersion`, `iconUrl`); Core also accepts the snake_case spellings on write.
- **Fields rejected on PUT (400, "not supported for a stored alias"):** `endpoint`, `features`, `catalogProperties`, `catalogSchemaId`, `author`, `createdAt`. **Accepted:** `intro` and `descriptionKeywords` (read back as written), `displayName`, `description`, `displayVersion`, `iconUrl`, `userRoles`.
- **Core fills nothing in** beyond `name` and `status` (`valid`), so no attribute needs to be computed. `userRoles: []` and `displayName: ""` read back as written; unset fields are omitted.
- **A locale map is accepted for `displayName` on write and read back as a plain string** (`{"en":"E"}` returned `"E"`).
- **A body `name` other than the path name is a 400** (`name 'other' is not the name of the alias in the path`); a matching one is accepted.
- **Slashes in `path` are not supported** (raw and `%2F` are both a 400 `Invalid entity name segment`).
- **DELETE of a missing alias is a 404**, handled as success. **No static alias** is defined (`GET /v1/admin/config/file/aliases` returns an empty list), so `aidial_file_alias` has no acceptance test.

## Risks / Trade-offs

- **The OpenAPI contract disagreed with real Core** (buckets, target fields) and the endpoints are `x-preview` → the specs follow the probe; a newer Core may change the rules, which the next OpenAPI refresh and probe would catch.
- **Core may reject an alias whose target does not exist** (the 400 "target that does not fit presentAs") → the error surfaces at apply with Core's message; ordering against the target resource is the user's job via references or `depends_on`.
- **Locale-map display values** set out of band are flattened to a string by Core, so a locale map is not representable → documented in the README.
- **The ignored fields are silently dropped on update.** A PUT replaces the document, so an out-of-band value in an unexposed field is lost on the next apply → document it in the README, as the interceptor change did.
- **A file-sourced alias may not exist on the ephemeral Core** → unit tests cover the client and the data source; no acceptance test for `aidial_file_alias` unless the Core ships one.

## Open Questions

None.
