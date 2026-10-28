# Proposal

## Why

The `aidial_model` and `aidial_file_model` data sources expose upstream `key` and `secret_extra_data` without marking them sensitive, so secrets appear in plan and console output. Docs, specs and the CHANGELOG have also drifted from what v0.2.0 actually shipped, and the release pipeline published v0.2.0 with empty release notes and a floating GoReleaser version. This patch release (v0.2.1) fixes all of that before the breaking v0.3.0 work starts, and it is the first release to run the pipeline hardened in #10.

## What Changes

- **Data source secrets**: upstream `key` and `secret_extra_data` in the `aidial_model` and `aidial_file_model` data sources become `Sensitive`, matching the resources.
- **URL path escaping**: every path parameter (`bucket`, `path`, file model `name`) is percent-encoded as one segment with `url.PathEscape`. Today only `/` is replaced, and the file model URL is not escaped at all. URLs for names without reserved characters are unchanged.
- **`aidial_key` null vs empty**: when DIAL Core omits `project` or `role`, state holds null rather than `""`. An explicitly configured `""` is still kept.
- **Cleanups with no behavior change**: drop the ignored `bucket` argument of `client.GetBucket`, remove the dead `saved.Key` / `key.Key` assignments in `aidial_key`, and fix `TestGetFileModelEscapesDotsInName` so its assertion matches its comment.
- **Docs and metadata drift**:
  - AGENTS.md, the README description and table, and the `provider-identity` spec list the `aidial_catalog_schema` resource and the `aidial_model` data source, and describe five resources instead of four.
  - `CHANGELOG.md` gets an `Unreleased` section, and its 0.2.0 entry gains the `aidial_model` data source.
  - The README states that Terraform ≥ 1.5 is required and installs with `~> 0.2`.
  - Each resource gets an `import-by-string-id.tf` import-block example.
  - `examples/provider/provider.tf` gets a version constraint.
  - `docs/` is regenerated.
- **Release pipeline**:
  - The tag's CHANGELOG section becomes the GitHub release notes.
  - The GoReleaser and syft versions are pinned.
  - CI runs `goreleaser check`.
  - The release job gets a timeout and a non-cancelling concurrency group.
  - After publishing, the job checks the GPG signature, the checksums and the provenance attestation.
  - The README's Releasing section documents annotated tags. The stable-only `vX.Y.Z` tag pattern stays.
- **SECURITY.md**:
  - The list of scheduled security jobs matches `security.yml`.
  - The maintainer settings list adds private vulnerability reporting (which the policy already tells reporters to use) and the GitHub Actions token and SHA-pinning settings.

No schema changes. One configuration-visible effect: a root module `output` that exposes a data source's upstream `key` or `secret_extra_data` must now set `sensitive = true`, or Terraform rejects the plan. The CHANGELOG calls this out.

## Capabilities

### New Capabilities

- `sensitive-attributes`: which attributes hold secrets and must be `Sensitive` in every resource and data source that exposes them.
- `core-client-requests`: how the DIAL Core client builds request URLs from bucket, path and name parameters.
- `key-state-consistency`: how `aidial_key` maps DIAL Core responses into state, starting with null vs empty for `project` and `role`.

### Modified Capabilities

- `provider-identity`: the registered resource and data source types now include `aidial_catalog_schema` (resource and data source) and the `aidial_model` data source.
- `public-repo-readiness`:
  - Docs include an import-block example.
  - The README states the minimum Terraform version.
  - The CHANGELOG keeps an `Unreleased` section and lists every shipped type.
  - `SECURITY.md`'s settings list and safeguards are accurate.
- `registry-release`:
  - Release notes come from the CHANGELOG.
  - Release tooling versions are pinned.
  - The GoReleaser config is checked in CI.
  - Releases verify themselves after publishing.
  - The release job is bounded.

## Impact

- **Code**:
  - `internal/client/client.go`: URL builders and `GetBucket`.
  - `internal/client/file_model_test.go`.
  - `internal/provider/model_data_source.go` and `internal/provider/file_model_data_source.go`: upstream schema.
  - `internal/provider/key_resource.go`.
  - `internal/provider/bucket_data_source.go`.
- **Docs**: `README.md`, `AGENTS.md`, `CHANGELOG.md`, `SECURITY.md`, `examples/`, and the regenerated `docs/`.
- **CI and release**: `.github/workflows/release.yml`, `.github/workflows/ci.yml` (the `goreleaser check` step), and `.goreleaser.yml` only if `check` flags something.
- **Users**:
  - Plan output for the two data sources now shows `(sensitive value)` for the upstream secrets.
  - New `aidial_key` resources that omit `project` or `role` store null. Existing state keeps its current values, with no diff.
- **Out of scope**:
  - OpenSpec housekeeping: archiving `add-model-data-source` or moving spec directories.
  - Dependabot and devcontainer changes.
  - GitHub repository settings. The maintainer applies them manually, and v0.2.1 is tagged only after the `release` environment holds the signing secrets.
