# Proposal

## Why

CI and release automation run on mutable action tags, the release workflow grants `contents: write` to every step, and nothing scans dependencies, secrets or the workflows themselves. Releases carry only a GPG-signed checksum file, with no provenance or SBOM, and CI uploads Terraform state files as artifacts. This is the supply-chain and CI/CD slice of a wider security review; provider and client behavior are deliberately untouched.

## What Changes

- Pin every GitHub Action to a full commit SHA, and have Dependabot group action updates and wait three days before proposing a new version.
- Run workflows with least privilege: no persisted checkout credentials, job-level release permissions, and a protected `release` environment for the signing job.
- Add a build provenance attestation and a per-archive SBOM to releases, keeping the GPG-signed checksums the Terraform Registry verifies.
- Add blocking CI security checks: `govulncheck`, CodeQL, dependency review, gitleaks and zizmor, plus `gosec` through golangci-lint. `govulncheck`, CodeQL and gitleaks also run weekly on the default branch.
- **BREAKING (CI only):** stop uploading Terraform state as CI artifacts; this removes the "Debug state is kept" requirement.
- Add `govulncheck`, `zizmor`, `goreleaser` and `syft` to the devcontainer so the new checks and the release dry run can run locally.
- Add `CODEOWNERS` and a `SECURITY.md` checklist of GitHub settings that maintainers apply by hand.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `ci-integration-core`: remove the state-artifact upload requirement; add requirements for blocking security checks and for least-privilege, SHA-pinned workflows.
- `registry-release`: add requirements for build provenance, an SBOM per archive, and a protected release environment.
- `devcontainer-tooling`: add the four security and release tools, with checksum-verified downloads.
- `public-repo-readiness`: add requirements for `CODEOWNERS` and for documented GitHub settings in `SECURITY.md`.

## Impact

- Changed: `.github/workflows/{ci,stub-core,release}.yml`, `.github/dependabot.yml`, `.goreleaser.yml`, `.golangci.yml`, `SECURITY.md`, `.devcontainer/Dockerfile`, README tool table; `main.go` gains one reasoned `//nolint:gosec`.
- New: `.github/workflows/security.yml`, `.github/CODEOWNERS`.
- Unchanged: provider and client behavior, `stub/` and `tests/core` (environment hardening is out of scope), and the rest of the devcontainer (only the four tools are added).
- Maintainers must enable branch protection with the new checks, tag protection for `v*`, secret scanning with push protection, and a `release` environment (documented, not applied by this change).
