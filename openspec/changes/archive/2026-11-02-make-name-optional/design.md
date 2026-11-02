# Design

## Context

See proposal.md - Why. Today `name` is `Required` on model, role and route and is always sent (`json:"name"` without `omitempty`). `set*State` sets `api_name` from the response and sets `name` only when it is null or unknown, so a configured `name` is never rewritten. Core derives the returned name from the path: the bare path for models and roles, `routes/<bucket>/<path>` for routes (observed, not in the OpenAPI contract).

## Goals / Non-Goals

**Goals:**
- Let configurations omit `name` without drift or provider-consistency errors.
- Keep every existing configuration and state valid and unchanged.

**Non-Goals:**
- Removing `name`, renaming `api_name`, or any change to `aidial_key` and `aidial_catalog_schema`.

## Decisions

**Optional+Computed with `UseStateForUnknown`.** On create `name` is unknown at plan time and becomes known after apply from the API response; afterwards the plan modifier keeps the stored value. The existing "set only when null or unknown" logic in `set*State` already implements this, so the read path needs no change. Alternative: default `name` to `path` at plan time. Rejected: it recreates the `name`/`path` coupling just removed, and for routes Core returns a different value (`routes/<bucket>/<path>`), which would cause a provider-consistency error.

**Configured `name` is still sent and still never rewritten.** This preserves current behavior. When the config later sets a `name` different from state, the plan is an in-place update and state takes the configured value, even though Core ignores it. Alternative: stop sending `name`. Rejected: Core versions that honor it keep working, and it keeps the change minimal.

**Client omits unset `name`.** `Name` tags become `json:"name,omitempty"` on `Model`, `Role` and `Route`. Core accepted bodies without `name` for all three on 2026-11-02. Alternative: send `path` as the name. Rejected for routes, where Core reports a prefixed name.

**`aidial_model` data source `name` optional+computed.** The lookup uses `bucket` and `path` only, so requiring `name` there has the same defect. Assumption recorded here, not confirmed with the user: it follows the resource.

**No schema version bump.** Required to optional+computed does not change the stored type, and state already holds a `name` value. Verified by a unit test that decodes prior state against the new schema.

## Risks / Trade-offs

- [A configured `name` that Core ignores can differ from `api_name`] -> Already the case today; `api_name` is the authoritative value and is documented as such.
- [Older Core might require `name`] -> Only the current Core was tested; a configured `name` is still sent, and omitting it is the user's choice.
- [`name` unknown after create shows as "(known after apply)" in plans] -> Expected for a computed attribute; stable after the first apply.
- [Spec layout] -> As in `require-bucket-path-identity`: the schema, state-consistency and read specs live at `openspec/<capability>/` and are edited by hand; archive with `--skip-specs`.
