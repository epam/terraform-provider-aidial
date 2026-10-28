# Tasks

## 1. Pinned, least-privilege workflows

- [x] 1.1 Pin every action in `ci.yml`, `stub-core.yml` and `release.yml` to a commit SHA with a version comment, and verify `grep -rn "uses:" .github/workflows` shows no tag references
- [ ] 1.2 Add `persist-credentials: false` to every checkout and move release permissions to job level with `permissions: {}` at workflow level; verify with `zizmor .github/workflows` (or the CI zizmor job)
- [x] 1.3 Remove the state artifact uploads from `ci.yml` and `stub-core.yml`; verify `grep -rn upload-artifact .github/workflows` is empty
- [x] 1.4 Group Actions updates and add a 3-day cooldown for `gomod` and `github-actions` in `.github/dependabot.yml`; verify the YAML parses

## 2. Security checks

- [ ] 2.1 Add `.github/workflows/security.yml` with `govulncheck`, CodeQL, dependency review, gitleaks (pinned binary, SHA-256 verified) and zizmor, with pull request, push, weekly cron and manual triggers; verify the YAML parses and the jobs pass on the pull request
- [x] 2.2 Enable `gosec` in `.golangci.yml` with the test-only exclusions, and add the reasoned `//nolint:gosec` in `main.go`; verify `golangci-lint run` reports 0 issues and `go test ./... -count=1 -race` passes

## 3. Release provenance and SBOM

- [ ] 3.1 Add `sboms` to `.goreleaser.yml` and a pinned syft install step to `release.yml`; verify `goreleaser release --snapshot --clean` produces an SBOM per archive
- [x] 3.2 Add the `release` environment and the provenance attestation step to `release.yml` with subject checksums from the `SHA256SUMS` file; verify the workflow lists the three write permissions only on the release job

## 4. Governance and docs

- [ ] 4.1 Add `.github/CODEOWNERS` covering the repository, `.github/`, `.goreleaser.yml` and `SECURITY.md`; verify GitHub reports the file valid on the pull request
- [x] 4.2 Extend `SECURITY.md` with the attestation verification command and the GitHub settings checklist; verify `markdownlint-cli2 SECURITY.md` passes and the documented `gh attestation verify` command matches the release asset names

## 5. Integration

- [x] 5.1 Run the full local check set (`gofmt -l .`, `go vet ./...`, `go test ./... -count=1 -race`, `golangci-lint run`, `gitleaks detect`, `markdownlint-cli2`) and verify all pass
- [ ] 5.2 Open the pull request and verify every `CI` and `Security` job passes

## Workflow follow-up

- Maintainers apply the `SECURITY.md` settings checklist in GitHub (branch protection, tag protection, secret scanning, `release` environment).
- Confirm the release path on the next `vX.Y.Z` tag, then archive the change.
