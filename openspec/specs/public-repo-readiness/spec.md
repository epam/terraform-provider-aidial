# public-repo-readiness Specification

## Purpose

Define what the repository must contain and exclude to be published publicly and listed in the Terraform Registry: license, generated docs, user-facing documentation and a clean tree.

## Requirements

### Requirement: License
The repository SHALL contain the Apache-2.0 license text in `LICENSE` at its root.

#### Scenario: License present
- **WHEN** the repository root is listed
- **THEN** `LICENSE` exists and contains the unmodified Apache License 2.0 text

### Requirement: Generated docs are committed
`docs/` SHALL contain tfplugindocs output for the provider, every resource and every data source, and SHALL be reproducible from the schema and `examples/`.

#### Scenario: No drift
- **WHEN** `tfplugindocs generate --provider-name aidial` runs on a clean checkout
- **THEN** `git diff --exit-code docs/` reports no changes

#### Scenario: Per-object examples
- **WHEN** a resource doc is generated
- **THEN** it includes an example from `examples/resources/<type>/` and, where import is supported, both a `terraform import` command and an `import` block example

### Requirement: Registry-oriented README
`README.md` SHALL document installation from the Registry with `source = "epam/aidial"` and a version constraint that matches the current minor release, SHALL state the minimum supported Terraform version, SHALL list every resource and data source the provider registers, and SHALL keep a development section covering local builds with `dev_overrides`.

#### Scenario: Install instructions
- **WHEN** a new user follows the README
- **THEN** they can configure the provider from the Registry without building it

#### Scenario: No stale statements
- **WHEN** the README is searched for "not published" or the `dial/dial` address
- **THEN** neither appears

#### Scenario: Terraform version stated
- **WHEN** a user reads the Installation section
- **THEN** it states that Terraform 1.5 or later is required

#### Scenario: Complete type table
- **WHEN** the resources and data sources table is compared with the types the provider registers
- **THEN** every registered resource and data source appears in the table

### Requirement: Community files
The repository SHALL contain `CHANGELOG.md`, `SECURITY.md` and `CONTRIBUTING.md`.

#### Scenario: Files present
- **WHEN** the repository root is listed
- **THEN** all three files exist and `CHANGELOG.md` has an entry for the first release

### Requirement: No internal-only content
The published tree SHALL NOT contain secrets, internal hostnames or URLs, GitLab project references or personal data other than intended authorship.

#### Scenario: Audit
- **WHEN** the audit searches the tree with gitleaks and a documented pattern list
- **THEN** no finding remains unresolved or unjustified

### Requirement: No legacy names
Outside `openspec/changes/archive/`, no file SHALL reference the old names `dial_*` resource types, `dial/dial`, `dialx` or `github.com/dial/terraform-provider-dial`.

#### Scenario: Search
- **WHEN** the repository is searched for the old names, excluding the archive and `openapicore/`
- **THEN** there are no matches

### Requirement: Code owners
The repository SHALL contain `.github/CODEOWNERS` assigning an owner to the whole repository and explicitly to `.github/`, `.goreleaser.yml` and `SECURITY.md`.

#### Scenario: Workflow change review
- **WHEN** a pull request modifies a file under `.github/`
- **THEN** the code owner is requested as reviewer

### Requirement: Documented GitHub settings
`SECURITY.md` SHALL describe how to verify a release attestation and SHALL list the GitHub settings that cannot be configured from the repository: branch protection requiring the CI and Security checks, tag protection for `v*`, secret scanning with push protection, private vulnerability reporting, a read-only default workflow token that cannot approve pull requests, required SHA pinning of actions, and a `release` environment with required reviewers that holds the signing secrets.

#### Scenario: Settings checklist
- **WHEN** a maintainer reads `SECURITY.md`
- **THEN** each of the seven settings and the attestation verification command is described

### Requirement: CHANGELOG tracks every shipped change
`CHANGELOG.md` SHALL keep an `Unreleased` section at the top for changes merged since the last tag, and each released version's entry SHALL name every resource and data source type added in that version.

#### Scenario: Unreleased section
- **WHEN** a change is merged after a release
- **THEN** it is recorded under `Unreleased` until the next tag moves it into a version heading

#### Scenario: Shipped types listed
- **WHEN** the 0.2.0 entry is read
- **THEN** it names the `aidial_catalog_schema` resource and data source and the `aidial_model` data source

### Requirement: Security policy matches the workflows
The safeguards section of `SECURITY.md` SHALL describe the security jobs and their triggers as defined in `.github/workflows/security.yml`.

#### Scenario: Scheduled jobs
- **WHEN** `SECURITY.md` is compared with the `schedule` trigger and job conditions in `security.yml`
- **THEN** it names exactly the jobs that run on the weekly schedule
