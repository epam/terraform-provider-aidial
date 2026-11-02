# Design

## Context

See proposal.md - Why. Today `bucket` is optional+computed (a plan modifier on model, role and route; a `stringdefault` on key and catalog schema) and `path` is optional+computed with a `name` fallback on model, role and route. `aidial_key` and `aidial_catalog_schema` already require `path`. Data sources: `aidial_role` already requires both; `aidial_model` defaults both; `aidial_catalog_schema` defaults `bucket`.

DIAL Core exposes `PUT/GET/DELETE /v1/{models,roles,keys,routes,catalog_schemas}/{bucket}/{path}`. `/v1/ops/resource/move` is tagged only for files, conversations, prompts, applications and toolsets.

## Goals / Non-Goals

**Goals:**
- One identity rule for every resource and its data source: both inputs written in configuration, validated at plan time.
- Remove the implicit `path = name` coupling.

**Non-Goals:**
- Supporting non-`platform` buckets, a move/rename operation, or any attribute change beyond `bucket`/`path`.

## Decisions

**Required, not defaulted.** `bucket` and `path` become `Required` (neither `Optional` nor `Computed`). Alternative considered: keep the defaults and add a warning. Rejected: the point is explicit identity, and a warning cycle was declined.

**`bucket` validated with `OneOf("platform")`.** Verified on 2026-11-02 against a local DIAL Core (`epam/ai-dial-core:latest`, key `ci-check-key`): a PUT of a model, role, key, route and catalog schema returned 200 for `/v1/<type>/platform/probe` and 404 for `/v1/<type>/other/probe`. This restriction is not documented in `openapicore/open_api_core.yaml`, whose `bucket` parameter reads only "The target bucket." and which has no `platform` path variant for these types. Required rather than removed or defaulted, so the shape matches every other bucket-scoped API and a future non-`platform` bucket is a one-line validator change. Alternatives: required with no validator (failure appears only at apply time as 404); keep the default (contradicts explicit identity).

**`path` validated with `LengthAtLeast(1)` only.** A path may contain `/` to nest, and Core rejects other invalid values.

**Keep `RequiresReplace` on both.** No move operation exists for these types, so a changed identity is a delete plus create. Import is unchanged.

**Data sources align with their resources** (`aidial_role` already does). The `Read` fallbacks to `defaultBucket` and `name` are removed.

**Delete the default helpers outright** (`bucketDefaultsToPlatform`, `pathDefaultsToName`, `defaultBucket`, the `stringdefault`s) and their tests, rather than leaving dead code that contradicts the spec. The `bucket` validator can use a shared constant for `"platform"` if one is still useful.

**Spec layout.** Per-resource specs and `resource-identity-defaults` live at `openspec/<capability>/` (see AGENTS.md), while the OpenSpec CLI manages `openspec/specs/`. The deltas here use the capability names so they can be reviewed and validated; during apply the edits are made directly in the existing files, and `resource-identity-defaults` is moved with `git mv` to `resource-identity`. `catalog-schema-resource` is the only affected capability under `openspec/specs/`.

## Risks / Trade-offs

- [Existing configurations that omit `bucket` or `path` fail validation] -> Intended; documented in the proposal and generated docs, no release note.
- [Validator on `bucket` blocks a future non-`platform` bucket] -> One-line change; the restriction is Core behavior verified today, not part of its published contract.
- [`add-model-data-source` is complete but not archived, so `model-data-source` has no main spec yet] -> Archive it before applying this change, or fold the `model-data-source` delta into that spec directly.
- [Fixtures omitting bucket/path elsewhere] -> The tasks include a scan of `.tf` and `.tf.json` files under `stub/`, `tests/` and `examples/`; the rest of the CI integration job catches stragglers.
