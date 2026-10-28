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

### Requirement: Debug state is kept
The job SHALL upload the Terraform state files as an artifact even when the job fails.

#### Scenario: Failed apply
- **WHEN** apply fails
- **THEN** the state artifact is still attached to the job

### Requirement: Standard checks on every change
The GitHub Actions CI workflow SHALL run gofmt, `go vet`, `go test -race`, golangci-lint, tfproviderlint, markdownlint, `terraform fmt -check` and the docs drift check on pull requests and on pushes to the default branch.

#### Scenario: Unformatted Go file
- **WHEN** a pull request contains a file that `gofmt -l` lists
- **THEN** the CI workflow fails

#### Scenario: Stale docs
- **WHEN** a pull request changes a schema without regenerating `docs/`
- **THEN** the docs drift check fails
