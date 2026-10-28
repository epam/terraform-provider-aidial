# Security Policy

## Supported versions

Security fixes are released for the latest minor version.

## Reporting a vulnerability

Please do not open a public issue for security problems. Report them privately through GitHub's [private vulnerability reporting](https://github.com/epam/terraform-provider-aidial/security/advisories/new) for this repository, with a description, reproduction steps and the affected version. You should receive an acknowledgement within a few working days.

## Handling secrets

The provider sends the configured `api_key` in the `Api-Key` header and stores the `aidial_key` secret in Terraform state. Mark variables that hold them `sensitive = true` and protect your state backend accordingly.

## Release integrity

Each release publishes the provider archives, a SHA256SUMS file and its detached GPG signature (the contract the Terraform Registry verifies), plus an SBOM per archive and a GitHub build provenance attestation.

Verify a downloaded archive against the attestation with the GitHub CLI:

```bash
gh attestation verify terraform-provider-aidial_<version>_<os>_<arch>.zip --repo epam/terraform-provider-aidial
```

## Repository safeguards

CI runs `govulncheck`, CodeQL, dependency review, gitleaks and zizmor (`.github/workflows/security.yml`), and `gosec` through golangci-lint. `govulncheck`, CodeQL and gitleaks also run weekly on `main`. GitHub Actions are pinned to commit SHAs, and Dependabot proposes updates after a seven-day cooldown.

### Settings maintainers apply in GitHub

These cannot be configured from the repository, so apply them once:

- Branch protection on `main`: require pull requests, code owner review, and the `CI` and `Security` checks.
- Tag protection (a ruleset) for `v*` so only maintainers can create release tags.
- Secret scanning with push protection enabled.
- A `release` environment with required reviewers. Move `GPG_PRIVATE_KEY` and `PASSPHRASE` from repository secrets into it, so only the release job can read them.
