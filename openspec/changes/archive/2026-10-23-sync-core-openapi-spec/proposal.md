# Proposal

## Why

`openapicore/open_api_core.yaml` is the contract that the client and every `openspec/*` "known gaps" note are checked against, but it is a hand copy with no recorded source. It has `info.version: 0.0.0`. Upstream `epam/ai-dial-core` publishes the same file at `docs/open_api_core.yaml`. At release `0.49.3` it differs from ours by roughly 930 diff lines: it declares `info.version: 0.49.2`, adds a `Projects` tag and adds `/openai/responses`. A stale contract makes the gap analysis unreliable, and nothing says which Core version it describes.

## What Changes

- Add a reproducible way to fetch `docs/open_api_core.yaml` from a given `epam/ai-dial-core` release tag into `openapicore/open_api_core.yaml` (script plus `make` target).
- Record the upstream tag and source URL beside the file in `openapicore/`.
- Refresh the checked-in spec from the latest release (`0.49.3` at time of writing).
- Re-check the contract-derived notes in the `openspec/*-schema` and `openspec/*-client` specs against the refreshed file, and update any that no longer hold.
- No provider code or Terraform schema changes; no CI Core image pinning (out of scope).

## Capabilities

### New Capabilities
- `core-openapi-contract`: How the checked-in DIAL Core OpenAPI contract is sourced, versioned and refreshed.

### Modified Capabilities
<!-- None: no resource behavior changes. Notes in existing specs are corrected in place, not via requirement deltas. -->

## Impact

- `openapicore/open_api_core.yaml` (replaced), new provenance file in `openapicore/`.
- `scripts/` (new fetch script), `Makefile` (new target), `README.md` (document the refresh).
- `openspec/{model,role,route}-{schema,client}/spec.md` notes may be edited if the new contract changes them.
- Needs network access to `raw.githubusercontent.com` only when running the refresh, not in CI.
