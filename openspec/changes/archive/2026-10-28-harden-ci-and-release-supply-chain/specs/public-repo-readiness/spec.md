# Spec Delta

## ADDED Requirements

### Requirement: Code owners
The repository SHALL contain `.github/CODEOWNERS` assigning an owner to the whole repository and explicitly to `.github/`, `.goreleaser.yml` and `SECURITY.md`.

#### Scenario: Workflow change review
- **WHEN** a pull request modifies a file under `.github/`
- **THEN** the code owner is requested as reviewer

### Requirement: Documented GitHub settings
`SECURITY.md` SHALL describe how to verify a release attestation and SHALL list the GitHub settings that cannot be configured from the repository: branch protection requiring the CI and Security checks, tag protection for `v*`, secret scanning with push protection, and a `release` environment with required reviewers that holds the signing secrets.

#### Scenario: Settings checklist
- **WHEN** a maintainer reads `SECURITY.md`
- **THEN** each of the four settings and the attestation verification command is described
