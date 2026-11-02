# Proposal

## Why

Models created through the `aidial_model` resource cannot be read by other Terraform configurations or workspaces. Roles, buckets and catalog schemas already have data sources, and `aidial_file_model` covers only models from DIAL Core's static configuration. A workspace that needs to reference a model it did not create (for example to attach a role or route to it) has no way to look it up from the provider.

## What Changes

- Add data source `aidial_model` that reads an API-managed (resource-storage) model by `name`, with optional `bucket` (default `platform`) and `path` (default `name`), mirroring the resource's identity model.
- Expose the same computed attributes as `aidial_file_model`, plus `id` (`<bucket>/<path>`).
- A model that does not exist produces an error diagnostic.
- Keep `aidial_file_model` unchanged: the two data sources use different API paths and are not merged.
- Regenerate `docs/` and add an `examples/data-sources/aidial_model/data-source.tf` snippet.

## Capabilities

### New Capabilities

- `model-data-source`: the `aidial_model` data source, covering its lookup arguments, computed attributes, identity defaults and not-found behavior.

### Modified Capabilities

<!-- None: no existing requirement changes. -->

## Impact

- Code: new `internal/provider/model_data_source.go`, registration in `internal/provider/provider.go`; reuses `client.GetModel` and `setModelState` unchanged.
- Tests: unit tests for the data source, plus an acceptance test (`TestAcc*`) against the ephemeral Core.
- Docs: `docs/data-sources/model.md` (generated) and the new example.
- No API, state or schema changes to existing resources; not a breaking change.
- Out of scope: extracting a shared computed-attribute schema between the two model data sources.
