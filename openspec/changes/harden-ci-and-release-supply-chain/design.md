# Design

## Context

CI (`ci.yml`), the manual stub apply (`stub-core.yml`) and the tag release (`release.yml`) use mutable action tags. `release.yml` holds `contents: write` workflow-wide and signs checksums with a GPG key from repository secrets, which is what the Terraform Registry verifies. `.golangci.yml` enables a minimal linter set, and `.gitleaks.toml` exists but no workflow runs it. See proposal.md for motivation and the delta specs for the requirements.

## Goals / Non-Goals

**Goals:**
- Make the CI and release path resistant to a compromised or retagged third-party action.
- Add supply-chain evidence (provenance, SBOM) without disturbing the Registry contract.
- Fail the build on known vulnerabilities, leaked secrets and unsafe workflow patterns.

**Non-Goals:**
- Any change to provider or client runtime behavior, including the `providerserver.Serve` error that `main.go` ignores.
- Hardening `stub/`, `tests/core` or the devcontainer.
- Changing GitHub repository settings; they are documented for maintainers.

## Decisions

- **SHA pins at exact, aged versions.** Each action is pinned to the commit of a release at least two weeks old, on its current major, with the version in a comment. Alternative: pin by major tag and rely on Dependabot. Rejected, because a retagged major runs unreviewed code with the release secrets.
- **Keep GPG, add attestation, no cosign.** The Registry only verifies the GPG-signed `SHA256SUMS`. `actions/attest-build-provenance` takes that checksum file as its subject set, so one attestation covers every archive and SBOM. Alternative: cosign keyless signing. Rejected as a second signature nobody verifies.
- **SBOM through GoReleaser `sboms`.** It reuses the archive pipeline; syft is installed with the pinned `anchore/sbom-action/download-syft` step because GoReleaser shells out to it. Alternative: a separate SBOM action that scans the repo. Rejected because it describes source, not the shipped archives.
- **gitleaks as a pinned, checksum-verified binary.** The gitleaks action requires a paid license for organization repositories. The workflow downloads a fixed release and checks its SHA-256, then runs against `.gitleaks.toml`.
- **One new workflow, `security.yml`.** Jobs run on pull requests, pushes to `main`, a weekly cron and manual dispatch. Keeping them out of `ci.yml` lets maintainers require the `Security` checks separately. All jobs block; the cron only notifies through the failed run.
- **zizmor lints the workflows.** It enforces SHA pins and flags injection patterns, so the pinning requirement is checked mechanically rather than by review.
- **gosec scope.** Findings in test code are excluded in `.golangci.yml` (G104 on `httptest` handlers, mirroring the existing errcheck exclusion; G704 in the acceptance-test cleanup that calls the Core named by `DIAL_BASE_URL`). The one production finding, the ignored `Serve` error in `main.go`, gets a `//nolint:gosec` with a reason, because handling it would change exit behavior and is out of scope.
- **Least privilege by default.** `permissions: {}` at workflow level where jobs declare their own, `persist-credentials: false` on every checkout, and release permissions moved to the job. The `release` environment is referenced in the workflow; its reviewers and secrets are maintainer configuration.
- **Drop state artifact uploads.** The Terraform state can hold sensitive values and the dummy CI keys set a bad precedent. Logs remain; local reproduction uses `scripts/test-integration.sh`.
- **Images stay tag-based.** `epam/ai-dial-core:latest` is an intentional upstream canary, and no Dependabot `docker` ecosystem is added.

## Risks / Trade-offs

- [The release path cannot be fully tested before a tag] → Run `goreleaser release --snapshot --clean` locally, and treat the first tag as the real test.
- [Blocking scanners may fail on unrelated advisories] → Fix or pin the dependency; narrow, commented suppressions only.
- [The 3-day cooldown delays non-security updates] → Security updates bypass the cooldown.
- [Newly required checks do nothing until branch protection names them] → Documented in `SECURITY.md`; the maintainer applies it.
- [The `release` environment auto-creates on first run without reviewers] → The checklist tells maintainers to add reviewers and move the secrets before the next tag.
