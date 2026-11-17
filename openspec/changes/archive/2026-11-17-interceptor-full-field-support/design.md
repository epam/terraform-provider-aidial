# Design

## Context

See proposal.md for motivation. The `Interceptor` client type and `aidial_interceptor` resource model `endpoint`, display fields, retry and auth flags, roles, headers, features, interfaces, `defaults`, `catalog_schema_id` and `catalog_properties`. `aidial_toolset` already maps the set attributes added here, and `aidial_application` gained the same ones in `application-full-field-support`; this change follows both.

Probe against Core 0.50.1 (PUT then GET on a throwaway `platform` interceptor):

- A minimal interceptor reads back `descriptionKeywords: []`, `dependencies: []`, `interceptors: []`, `defaults: {}`, `responsesDefaults: {}`, `maxRetryAttempts: 1`, `forwardAuthToken: false`, and null for `inputAttachmentTypes`, `maxInputAttachments`, `reference`, `intro`, `overrideName`, `baseUrl` and `responsesEndpoint`. Unlike toolsets, Core does not generate a `reference` for interceptors.
- All eleven new fields are accepted with camelCase keys and read back unchanged; a configured empty `inputAttachmentTypes` stays `[]`.
- `author`, `createdAt` and `updatedAt` read back null and are managed by Core.

## Goals / Non-Goals

**Goals:**
- Every field Core accepts for an interceptor round trips without plan churn.
- Existing state and configs keep working.

**Non-Goals:**
- Exposing `author`, `created_at` or `updated_at`.
- Changing aliases or other resources: Core rejects extra fields on stored aliases, and the other clients already cover every accepted field.

## Decisions

1. **Same attribute names and defaults as `aidial_toolset`**, so users see one convention. Set attributes use the existing `stringSetToAPI` and `stringSetFromAPI` helpers.
2. **Optional+Computed with `UseStateForUnknown` for `dependencies`, `description_keywords` and `interceptors`**, because Core reads them back as `[]` when unset. Everything else Core leaves null stays plain optional, so null stays null.
3. **`reference` is plain optional**, unlike toolsets and applications, because the probe showed Core does not generate it for interceptors. The acceptance test re-checks this, and `reference` becomes computed if Core's behavior differs.
4. **`responses_defaults` is a dynamic attribute** handled by the same free-form helper as `defaults`.
5. **Unmanaged fields stay dropped and documented**, as for applications.

## Risks / Trade-offs

- [`interceptors` on an interceptor is an unusual setting and Core may later reject or ignore it] → It is accepted and returned on Core 0.50.1; the acceptance test pins that, and a rejection would show as a clear Core error.
- [Core may start generating `reference` for interceptors] → Plain optional would then show a diff; the acceptance test on the supported Core catches it before release.
- [Core versions older than 0.50.1 may not accept all fields] → Fields are only sent when configured, so users of older Cores are unaffected unless they set them.

## Migration Plan

Additive schema only. Existing interceptors read the new sets as empty and the other new attributes as null, so the first plan after upgrading shows no diff. Rollback is installing the previous version, which ignores the new attributes and drops them again on its next PUT.
