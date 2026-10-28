## MODIFIED Requirements

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
