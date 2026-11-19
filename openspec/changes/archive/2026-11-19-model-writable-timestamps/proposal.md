# Proposal

## Why

`aidial_model` exposes `created_at` and `updated_at` as computed-only, so a configuration that sets them fails at plan time with "Invalid Configuration for Read-Only Attribute". Models exported from an existing DIAL Core carry these values (for example `updated_at = 1630454400000`), and users want to apply them as written. Probing Core 0.50.2 confirmed that `PUT /v1/models/{bucket}/{path}` accepts `createdAt` and `updatedAt` and returns them unchanged on read. Core never sets them itself: a PUT without them stores `null`.

## What Changes

- `aidial_model.created_at` and `aidial_model.updated_at` become optional, non-computed `Int64` attributes (Unix epoch milliseconds) with an at-least-zero validator.
- Create and update send each value as `createdAt` / `updatedAt` only when it is configured. Omitting one sends nothing, so Core stores `null`.
- Read-back keeps the API value in state. A value set outside Terraform that is not in the configuration shows as drift, and the next apply clears it.
- `created_at` loses its keep-prior-value plan modifier, because it is no longer computed.
- **BREAKING (minor):** a model whose `created_at` or `updated_at` was set outside Terraform now plans an update that clears it, unless the configuration states the value. Models created by this provider are unaffected, because Core returns `null` for them.
- The `aidial_model` and `aidial_file_model` data sources stay computed-only. No other resource changes.
- Attribute descriptions, the resource docs and `CHANGELOG.md` (Unreleased) state the clearing behavior.

## Capabilities

### New Capabilities

- `model-timestamps`: configurable `created_at` and `updated_at` on `aidial_model`: schema, request mapping, read-back and drift behavior.

### Modified Capabilities

None as delta specs. The older per-resource specs `openspec/model-schema/spec.md` and `openspec/model-update/spec.md` live outside `openspec/specs/`, where deltas cannot target them, so they are edited directly during apply (see tasks).

## Impact

- Code: `internal/provider/model_resource.go` (schema, plan struct, request builder, read mapping). `internal/client/client.go` already has `CreatedAt`/`UpdatedAt` pointers with `omitempty` on `ModelRequest`.
- Tests: `internal/provider/model_resource_test.go`; a new manifest in `tests/integration/`.
- Docs: regenerated `docs/resources/model.md`; `CHANGELOG.md`; `openspec/model-schema/spec.md`.
- Out of scope: data sources, other resources, the read-only `status`, validating that `updated_at >= created_at`.
