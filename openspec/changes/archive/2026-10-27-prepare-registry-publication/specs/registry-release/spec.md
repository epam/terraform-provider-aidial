# Spec Delta

## Purpose

Define the tag-triggered release that produces the signed, multi-platform artifacts and manifest the Terraform Registry requires to ingest a provider version.

## ADDED Requirements

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
