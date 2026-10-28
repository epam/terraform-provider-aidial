# Spec Delta

## ADDED Requirements

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
Dependabot SHALL group GitHub Actions updates into a single weekly pull request and SHALL wait three days after a release before proposing it for Go modules and GitHub Actions.

#### Scenario: Fresh release
- **WHEN** an action publishes a new version
- **THEN** Dependabot proposes it no earlier than three days later

## REMOVED Requirements

### Requirement: Debug state is kept
**Reason**: The state files can hold sensitive values, and uploading them as workflow artifacts exposes them to anyone with read access to the repository's runs.
**Migration**: Diagnose failures from the job log; reproduce locally with `scripts/test-integration.sh`.
