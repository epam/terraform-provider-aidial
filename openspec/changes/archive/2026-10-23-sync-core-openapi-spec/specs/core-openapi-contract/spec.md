# Spec Delta

## Purpose

Define how the checked-in DIAL Core OpenAPI contract is sourced, versioned and refreshed, so specs and client code are checked against a known Core release rather than an unlabelled copy.

## ADDED Requirements

### Requirement: Contract comes from an upstream release tag
`openapicore/open_api_core.yaml` SHALL be a byte-for-byte copy of `docs/open_api_core.yaml` from a tagged `epam/ai-dial-core` release, not from a branch head.

#### Scenario: Refresh by tag
- **WHEN** the refresh is run with an upstream tag
- **THEN** `openapicore/open_api_core.yaml` equals that tag's `docs/open_api_core.yaml`

#### Scenario: Unknown tag
- **WHEN** the refresh is run with a tag that does not exist upstream
- **THEN** it exits non-zero and leaves the existing contract and provenance file unchanged

### Requirement: Source version is recorded
The repository SHALL record the upstream tag and source URL of the checked-in contract in a file under `openapicore/`, updated by the same refresh that replaces the contract.

#### Scenario: Provenance after refresh
- **WHEN** a refresh to tag `X` succeeds
- **THEN** the provenance file names tag `X` and the URL it was fetched from

### Requirement: Refresh is a single command
The repository SHALL provide a `make` target that performs the refresh given a tag, and SHALL document it in the README.

#### Scenario: Missing tag
- **WHEN** the target is run without a tag
- **THEN** it exits non-zero with a usage message and changes nothing

### Requirement: Refresh is not part of CI
CI SHALL NOT fetch the contract from the network; it SHALL use the checked-in copy.

#### Scenario: Offline pipeline
- **WHEN** a pipeline runs without access to GitHub
- **THEN** no job fails because of the contract file
