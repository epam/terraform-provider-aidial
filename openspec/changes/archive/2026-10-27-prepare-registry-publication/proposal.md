# Proposal

## Why

The provider is moving from an internal GitLab project to a public GitHub repository, `github.com/alexey-ban/terraform-provider-aidial`, and to the official Terraform Registry. Its current identity (`dial` provider type, `dial/dial` address, `dial_*` resource names, GitLab-only CI, no license, no signed releases, no committed docs) cannot be published: the Registry requires a `terraform-provider-<name>` repository, signed GoReleaser artifacts, a manifest and generated docs. The `epam` HashiCorp Cloud organization is not accessible, so the first publication uses the personal namespace `alexey-ban`.

## What Changes

- **BREAKING** Rename the provider type from `dial` to `aidial`: resources `aidial_role`, `aidial_key`, `aidial_model`, `aidial_route` and data sources `aidial_bucket`, `aidial_role`, `aidial_file_model`. No `dial_*` aliases (the provider has never been published).
- **BREAKING** Change the provider address from `dial/dial` to `alexey-ban/aidial` (`registry.terraform.io/alexey-ban/aidial`), the Go module path to `github.com/alexey-ban/terraform-provider-aidial` and the binary to `terraform-provider-aidial`.
- Add the release pipeline: `.goreleaser.yml`, `terraform-registry-manifest.json` (protocol 6.0), and a GitHub Actions release workflow that builds multi-platform zips and a GPG-signed `SHA256SUMS` on `vX.Y.Z` tags.
- Replace `.gitlab-ci.yml` with GitHub Actions workflows (lint, test, build, stub-core and ephemeral-Core integration), plus Dependabot.
- Commit tfplugindocs-generated `docs/` and reorganize `examples/` into the tfplugindocs layout; rewrite the README for Registry installation.
- Add `LICENSE` (Apache-2.0), `CHANGELOG.md`, `SECURITY.md`, `CONTRIBUTING.md`; remove the stray `openspec-dial-key-fix-example.md`.
- Audit the tree for internal-only content (hostnames, GitLab paths, secrets, personal data) before publication.
- Update tooling (Makefile, devcontainer, VS Code launch, scripts, AGENTS.md, pre-commit) and the main OpenSpec specs to the new names.
- Document the manual publication steps (GPG key, Registry signing-key upload, GitHub secrets, tag). Out of scope: wiping git history and changing `origin`, which the owner does afterwards.

## Capabilities

### New Capabilities
- `provider-identity`: the provider type name, resource and data source type names, Registry source address, module path and binary name.
- `registry-release`: the tag-triggered signed release that satisfies the Terraform Registry's artifact requirements.
- `public-repo-readiness`: license, committed generated docs, Registry-oriented README and examples, community files, and the pre-publication content audit.

### Modified Capabilities
- `ci-integration-core`: the pipeline moves from GitLab CI to GitHub Actions (service containers, job-local credentials, `DIALX_*` rule restated for the new platform).
- `devcontainer-tooling`: the dev override address and built binary name change to `alexey-ban/aidial` and `terraform-provider-aidial`.

## Impact

- Code: `go.mod` and every import, `main.go`, `internal/provider/*` type names, tests.
- Config and tooling: `Makefile`, `.devcontainer/`, `.vscode/launch.json`, `scripts/`, `.golangci.yml`, `.pre-commit-config.yaml`, `AGENTS.md`; `.gitlab-ci.yml` removed; `.github/` added.
- Terraform configs: `examples/`, `stub/models/`, `tests/integration/` (resource types and `required_providers`).
- Docs: `README.md`, new `docs/`, `LICENSE` and community files; `openspec/` main specs updated to the new names (per-resource specs live directly under `openspec/<capability>/`, outside the CLI's `openspec/specs/` tree, so they are renamed by task, not by delta; archived changes stay untouched).
- Users: anyone with the old `dial` configuration must rename types and source (none are known to exist externally). A later move to the `epam` namespace would change the address again.
