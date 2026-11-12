# Spec Delta

## MODIFIED Requirements

### Requirement: Credentials are job-local
The job SHALL configure Core with a fixed CI key and SHALL pass the Core URL, key and test values to Terraform explicitly so repository or organization variables and `TF_VAR_*` environment cannot redirect it to another Core. The JWT run SHALL likewise pass the Core URL and the token explicitly and SHALL ignore any ambient `AIDIAL_*` credential variable.

#### Scenario: Conflicting project variable
- **WHEN** a repository-level `TF_VAR_aidial_base_url` points at a real Core
- **THEN** the integration job still applies only to the ephemeral Core

#### Scenario: Ambient credential variable
- **WHEN** `AIDIAL_API_KEY` is set in the runner environment
- **THEN** the JWT run still authenticates with the ID token, not with that key

## ADDED Requirements

### Requirement: JWT authentication run
A separate CI job SHALL start the shared ephemeral Core and apply a small dedicated manifest authenticating only with a GitHub Actions ID token (provider `id_token`), using its own state, and SHALL destroy what it created. The existing API-key integration job SHALL stay unchanged.

#### Scenario: Token accepted
- **WHEN** the JWT run executes in a non-fork workflow run
- **THEN** create, read and destroy of a role succeed under `Authorization: Bearer`

#### Scenario: Run fails
- **WHEN** Core rejects the token or any Terraform step fails
- **THEN** the job fails

#### Scenario: API-key run unchanged
- **WHEN** the JWT job is skipped
- **THEN** the full API-key integration job still runs and decides its own result

### Requirement: Token permission is job-scoped
`id-token: write` SHALL be granted only to the job that mints the token, and the token SHALL be masked in the log before being exported. The JWT run SHALL be skipped for pull requests from forks, which receive no token.

#### Scenario: Fork pull request
- **WHEN** a pull request originates from a fork
- **THEN** the JWT run is skipped, the API-key run still executes, and the workflow does not fail because of the skip

#### Scenario: Same-repository pull request
- **WHEN** a pull request originates from this repository
- **THEN** the JWT run executes

#### Scenario: Other jobs
- **WHEN** a job other than the JWT-minting one runs
- **THEN** it does not have `id-token: write`

#### Scenario: Token in logs
- **WHEN** the workflow log is inspected
- **THEN** the ID token value appears only masked
