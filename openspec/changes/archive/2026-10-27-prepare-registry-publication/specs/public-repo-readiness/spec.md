# Spec Delta

## Purpose

Define what the repository must contain and exclude to be published publicly and listed in the Terraform Registry: license, generated docs, user-facing documentation and a clean tree.

## ADDED Requirements

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
- **THEN** it includes an example from `examples/resources/<type>/` and, where import is supported, an import snippet

### Requirement: Registry-oriented README
`README.md` SHALL document installation from the Registry with `source = "alexey-ban/aidial"` and a version constraint, and SHALL keep a development section covering local builds with `dev_overrides`.

#### Scenario: Install instructions
- **WHEN** a new user follows the README
- **THEN** they can configure the provider from the Registry without building it

#### Scenario: No stale statements
- **WHEN** the README is searched for "not published" or the `dial/dial` address
- **THEN** neither appears

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
