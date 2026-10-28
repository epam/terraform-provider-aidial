# ci-integration-core Specification

## Purpose

Define how CI runs the `tests/integration` manifests against a throwaway DIAL Core so the check needs no secrets and leaves no shared state.

## Requirements

### Requirement: Integration runs against an ephemeral Core
The linux integration job in GitHub Actions SHALL start a DIAL Core and its Valkey from the shared `tests/core/docker-compose.yml`, with the `epam/ai-dial-core:latest` image unless `CORE_VERSION` says otherwise, and SHALL apply `tests/integration` only against that Core through the shared runner script.

#### Scenario: Merge request pipeline
- **WHEN** a pull request workflow runs and the build and validate jobs succeed
- **THEN** one integration job runs automatically, without manual approval and without reading any repository secret

#### Scenario: Core not healthy
- **WHEN** the Core service does not answer `/health` within the wait limit
- **THEN** the job fails before running Terraform

#### Scenario: No inline Core definition
- **WHEN** `.github/workflows/ci.yml` is inspected
- **THEN** it contains no Core config heredocs or `docker run` commands for Core or Valkey

### Requirement: Credentials are job-local
The job SHALL configure Core with a fixed CI key and SHALL pass the Core URL, key and test values to Terraform explicitly so repository or organization variables and `TF_VAR_*` environment cannot redirect it to another Core.

#### Scenario: Conflicting project variable
- **WHEN** a repository-level `TF_VAR_aidial_base_url` points at a real Core
- **THEN** the integration job still applies only to the ephemeral Core

### Requirement: Full lifecycle is exercised
The job SHALL run `terraform validate`, `plan`, `apply`, the null-vs-empty drift check and `destroy`, and SHALL fail if any step fails.

#### Scenario: Drift check
- **WHEN** apply succeeds
- **THEN** `drift_check_null_vs_empty.sh` runs against the ephemeral Core and the job fails if it fails

#### Scenario: Destroy
- **WHEN** the drift check passes
- **THEN** `terraform destroy -auto-approve` succeeds and leaves no managed resources in state

### Requirement: No real-Core integration jobs remain
The GitHub Actions workflows SHALL NOT contain integration jobs that target a real DIAL Core, a manual-approval integration variable or `DIALX_*` references, and `.gitlab-ci.yml` SHALL NOT exist.

#### Scenario: Darwin
- **WHEN** the workflows are inspected
- **THEN** macOS runs only provider build and `terraform validate`; no macOS integration plan, apply or destroy job exists

#### Scenario: GitLab pipeline removed
- **WHEN** the repository root is listed
- **THEN** `.gitlab-ci.yml` is absent

### Requirement: Standard checks on every change
The GitHub Actions CI workflow SHALL run gofmt, `go vet`, `go test -race`, golangci-lint, tfproviderlint, markdownlint, `terraform fmt -check` and the docs drift check on pull requests and on pushes to the default branch.

#### Scenario: Unformatted Go file
- **WHEN** a pull request contains a file that `gofmt -l` lists
- **THEN** the CI workflow fails

#### Scenario: Stale docs
- **WHEN** a pull request changes a schema without regenerating `docs/`
- **THEN** the docs drift check fails

### Requirement: Blocking security checks
The GitHub Actions workflows SHALL run `govulncheck`, CodeQL for Go, dependency review on pull requests, gitleaks and zizmor, and golangci-lint SHALL enable `gosec`. A failure in any of them SHALL fail the workflow.

#### Scenario: Vulnerable dependency
- **WHEN** a pull request adds a dependency with a known vulnerability
- **THEN** dependency review or `govulncheck` fails the workflow

#### Scenario: Committed secret
- **WHEN** a pull request adds a string that matches a gitleaks rule outside the allowlisted paths
- **THEN** the gitleaks job fails

#### Scenario: Unsafe workflow change
- **WHEN** a pull request introduces a workflow pattern that zizmor reports, such as script injection from an untrusted context
- **THEN** the zizmor job fails

### Requirement: Scheduled security scans
`govulncheck`, CodeQL and gitleaks SHALL also run on a weekly schedule against the default branch, so new advisories and rules are applied to unchanged code.

#### Scenario: New advisory
- **WHEN** an advisory is published for a dependency already on the default branch
- **THEN** the next scheduled `govulncheck` run reports it

### Requirement: Actions pinned to commit SHAs
Every `uses:` reference to a third-party action in `.github/workflows/` SHALL be a full 40-character commit SHA, with the released version in a trailing comment.

#### Scenario: Tag reference
- **WHEN** a workflow references an action by a tag or branch name
- **THEN** the zizmor check fails

### Requirement: Least-privilege workflows
Workflows SHALL declare no write permission at workflow level, SHALL grant additional permissions only on the job that needs them, and every checkout SHALL disable persisted credentials.

#### Scenario: Default token scope
- **WHEN** a workflow other than the release workflow runs
- **THEN** its jobs receive at most read access to repository contents, plus `security-events: write` only on jobs that upload code scanning results

### Requirement: Dependency update cooldown
Dependabot SHALL group GitHub Actions updates into a single weekly pull request and SHALL wait seven days after a release before proposing it for Go modules and GitHub Actions.

#### Scenario: Fresh release
- **WHEN** an action publishes a new version
- **THEN** Dependabot proposes it no earlier than seven days later
