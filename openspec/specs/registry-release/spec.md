# registry-release Specification

## Purpose

Define the tag-triggered release that produces the signed, multi-platform artifacts and manifest the Terraform Registry requires to ingest a provider version.

## Requirements

### Requirement: Release triggered by version tag
A GitHub Actions workflow SHALL build and publish a GitHub release when a tag matching `vX.Y.Z` is pushed, and SHALL NOT run a release for other tags or branches.

#### Scenario: Version tag
- **WHEN** tag `v0.1.0` is pushed
- **THEN** a GitHub release `v0.1.0` is created with the provider artifacts

#### Scenario: Other tag
- **WHEN** a tag such as `test` is pushed
- **THEN** the release workflow does not run

### Requirement: Registry-conformant artifacts
The release SHALL contain one zip per supported OS and architecture named `terraform-provider-aidial_<version>_<os>_<arch>.zip`, each holding the binary `terraform-provider-aidial_v<version>`, built with `CGO_ENABLED=0`.

#### Scenario: Platform coverage
- **WHEN** a release is produced
- **THEN** it includes linux, darwin, windows and freebsd builds for the architectures the Registry accepts, and no unsupported combination

### Requirement: Signed checksums
The release SHALL include `terraform-provider-aidial_<version>_SHA256SUMS` covering every zip and a detached GPG signature `..._SHA256SUMS.sig` made with the project signing key supplied through repository secrets.

#### Scenario: Verification
- **WHEN** the signature is verified with the published public key
- **THEN** verification succeeds and every listed checksum matches its zip

#### Scenario: Missing key
- **WHEN** the signing secrets are not configured
- **THEN** the release workflow fails rather than publishing unsigned artifacts

### Requirement: Registry manifest
The release SHALL include `terraform-provider-aidial_<version>_manifest.json` copied from `terraform-registry-manifest.json`, declaring metadata version 1 and protocol version 6.0.

#### Scenario: Manifest content
- **WHEN** the release assets are listed
- **THEN** the manifest asset exists and its protocol versions equal the protocol the provider serves

### Requirement: Local release dry run
The repository SHALL allow a snapshot release to be built locally without publishing or signing.

#### Scenario: Snapshot
- **WHEN** `goreleaser release --snapshot --clean` is run
- **THEN** it produces the zips, checksums and manifest in `dist/` and publishes nothing

### Requirement: Build provenance attestation
The release SHALL publish a GitHub build provenance attestation whose subjects are the artifacts listed in the signed `SHA256SUMS` file, and the Registry-facing signed checksums and manifest SHALL remain unchanged.

#### Scenario: Verify an archive
- **WHEN** `gh attestation verify` is run on a released zip against this repository
- **THEN** verification succeeds

#### Scenario: Registry contract intact
- **WHEN** a release is produced
- **THEN** `SHA256SUMS`, its detached GPG signature and the manifest are still present and unchanged in format

### Requirement: SBOM per archive
The release SHALL include a software bill of materials for each platform archive.

#### Scenario: SBOM assets
- **WHEN** the release assets are listed
- **THEN** each zip has a corresponding SBOM asset, and each SBOM is covered by `SHA256SUMS`

### Requirement: Protected release environment
The release job SHALL run in a GitHub environment named `release`, SHALL hold `contents: write`, `id-token: write` and `attestations: write` only at job level, and the workflow SHALL grant no permissions by default.

#### Scenario: Permission scope
- **WHEN** the release workflow is inspected
- **THEN** the workflow-level `permissions` grant nothing and the three write permissions appear only on the release job

### Requirement: Release notes from the CHANGELOG
The GitHub release for tag `vX.Y.Z` SHALL use the `X.Y.Z` section of `CHANGELOG.md` as its description, and the release workflow SHALL fail before publishing when that section is missing or empty.

#### Scenario: Notes published
- **WHEN** tag `v0.2.1` is pushed and `CHANGELOG.md` has a non-empty `## 0.2.1` section
- **THEN** the GitHub release `v0.2.1` shows that section's content

#### Scenario: Missing section
- **WHEN** a tag is pushed and `CHANGELOG.md` has no section for its version
- **THEN** the workflow fails and no release is published

### Requirement: Pinned release tooling
The release workflow SHALL run exact, pinned versions of GoReleaser and syft, not version ranges, so two runs of the same tag use the same tools.

#### Scenario: Workflow inspection
- **WHEN** `.github/workflows/release.yml` is inspected
- **THEN** GoReleaser and syft each have an exact version, with no `~>` or `latest`

### Requirement: Release configuration checked in CI
CI SHALL validate `.goreleaser.yml` with the same pinned GoReleaser version the release uses, on every pull request.

#### Scenario: Invalid configuration
- **WHEN** a pull request makes `.goreleaser.yml` invalid
- **THEN** a CI check fails

### Requirement: Release verifies its own artifacts
After publishing, the release workflow SHALL verify the GPG signature of the `SHA256SUMS` file, every checksum it lists, and the build provenance attestation of at least one archive, and SHALL fail when any check fails.

#### Scenario: Successful release
- **WHEN** a release completes
- **THEN** the workflow log shows a good signature, every checksum matching, and a verified attestation

#### Scenario: Bad signature
- **WHEN** the signature does not verify
- **THEN** the workflow fails

### Requirement: Bounded release job
The release job SHALL have a time limit, and a release run SHALL NOT be cancelled by a later run for the same tag.

#### Scenario: Workflow inspection
- **WHEN** `.github/workflows/release.yml` is inspected
- **THEN** the job sets `timeout-minutes`, and its concurrency group has `cancel-in-progress: false`

### Requirement: Annotated release tags
The documented release procedure SHALL create release tags as annotated tags matching `vX.Y.Z`.

#### Scenario: Release procedure
- **WHEN** a maintainer follows the README's Releasing section
- **THEN** the documented command creates an annotated tag
