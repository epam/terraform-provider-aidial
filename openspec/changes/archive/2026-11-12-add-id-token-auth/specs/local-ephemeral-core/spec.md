# Spec Delta

## MODIFIED Requirements

### Requirement: Ephemeral Core definition is shared
The repository SHALL define a throwaway DIAL Core and its Valkey once, as a docker compose project under `tests/core/` with a fixed throwaway API key and no secrets, and local runs and CI SHALL use that definition. The Core SHALL also trust GitHub Actions OIDC tokens issued for this repository, in addition to the fixed key.

#### Scenario: Single definition
- **WHEN** the CI integration job and a local run start a Core
- **THEN** both use `tests/core/docker-compose.yml` and its config files

#### Scenario: Local run keeps the API key
- **WHEN** a developer runs `make test-integration` without internet access
- **THEN** the flow authenticates with the fixed API key and the unreachable identity provider does not affect it

#### Scenario: GitHub token for this repository
- **WHEN** a request carries a valid GitHub Actions ID token for this repository with the configured audience
- **THEN** Core grants it the admin role

#### Scenario: GitHub token for another repository
- **WHEN** a request carries a valid GitHub Actions ID token issued for a different repository
- **THEN** Core does not grant it the admin role and admin API calls are rejected

#### Scenario: Token from another issuer
- **WHEN** a request carries a token whose issuer is not GitHub Actions
- **THEN** Core rejects it
