# Spec Delta

## ADDED Requirements

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
