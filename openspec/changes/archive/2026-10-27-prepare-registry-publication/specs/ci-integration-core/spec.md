# Spec Delta

## MODIFIED Requirements

### Requirement: Integration runs against an ephemeral Core
The linux integration job in GitHub Actions SHALL start a DIAL Core and its Valkey as job services, with the `epam/ai-dial-core:latest` Core image, and SHALL apply `tests/integration` only against that Core.

#### Scenario: Merge request pipeline
- **WHEN** a pull request workflow runs and the build and validate jobs succeed
- **THEN** one integration job runs automatically, without manual approval and without reading any repository secret

#### Scenario: Core not healthy
- **WHEN** the Core service does not answer `/health` within the wait limit
- **THEN** the job fails before running Terraform

### Requirement: Credentials are job-local
The job SHALL configure Core with a fixed CI key and SHALL pass the Core URL, key and test values to Terraform explicitly so repository or organization variables and `TF_VAR_*` environment cannot redirect it to another Core.

#### Scenario: Conflicting project variable
- **WHEN** a repository-level `TF_VAR_aidial_base_url` points at a real Core
- **THEN** the integration job still applies only to the ephemeral Core

### Requirement: No real-Core integration jobs remain
The GitHub Actions workflows SHALL NOT contain integration jobs that target a real DIAL Core, a manual-approval integration variable or `DIALX_*` references, and `.gitlab-ci.yml` SHALL NOT exist.

#### Scenario: Darwin
- **WHEN** the workflows are inspected
- **THEN** macOS runs only provider build and `terraform validate`; no macOS integration plan, apply or destroy job exists

#### Scenario: GitLab pipeline removed
- **WHEN** the repository root is listed
- **THEN** `.gitlab-ci.yml` is absent

## ADDED Requirements

### Requirement: Standard checks on every change
The GitHub Actions CI workflow SHALL run gofmt, `go vet`, `go test -race`, golangci-lint, tfproviderlint, markdownlint, `terraform fmt -check` and the docs drift check on pull requests and on pushes to the default branch.

#### Scenario: Unformatted Go file
- **WHEN** a pull request contains a file that `gofmt -l` lists
- **THEN** the CI workflow fails

#### Scenario: Stale docs
- **WHEN** a pull request changes a schema without regenerating `docs/`
- **THEN** the docs drift check fails
