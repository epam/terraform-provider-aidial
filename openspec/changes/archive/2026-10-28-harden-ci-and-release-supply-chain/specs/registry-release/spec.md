# Spec Delta

## ADDED Requirements

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
