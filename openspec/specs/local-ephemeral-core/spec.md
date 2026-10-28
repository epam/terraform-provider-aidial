# local-ephemeral-core Specification

## Purpose

Define the throwaway DIAL Core used by CI and by developers to run the integration flow and Go acceptance tests, including selecting its version, running it on the host or locally, and connecting to it from the devcontainer.

## Requirements

### Requirement: Ephemeral Core definition is shared
The repository SHALL define a throwaway DIAL Core and its Valkey once, as a docker compose project under `tests/core/` with a fixed throwaway API key and no secrets, and local runs and CI SHALL use that definition.

#### Scenario: Single definition
- **WHEN** the CI integration job and a local run start a Core
- **THEN** both use `tests/core/docker-compose.yml` and its config files

### Requirement: Core version is selectable
The Core image tag SHALL be taken from the `CORE_VERSION` environment variable and SHALL default to `latest`.

#### Scenario: Pinned version
- **WHEN** a contributor runs the start command with `CORE_VERSION=0.48.1`
- **THEN** the `epam/ai-dial-core:0.48.1` image is started

#### Scenario: Unknown version
- **WHEN** `CORE_VERSION` names a tag that cannot be pulled
- **THEN** the start command fails with the pull error and leaves no containers running

### Requirement: Core is ephemeral and local-only
Each start SHALL create a fresh Core with no data from earlier runs, the stop command SHALL remove its containers, networks and volumes, and the published ports SHALL bind to `0.0.0.0` by default unless `CORE_BIND` overrides it. The default publish address SHALL be `0.0.0.0` so containers can reach the Core.

#### Scenario: Restart
- **WHEN** the Core is stopped and started again
- **THEN** resources created in the earlier run are absent

### Requirement: External Core mode
When `CORE_URL` is set, or when no Docker daemon is reachable, the scripts SHALL use an external Core as is (default `http://host.docker.internal:<CORE_PORT>`, falling back to the container's default gateway when that name does not resolve): starting only waits for its `/health`, stopping does nothing, no Docker is required, and the Core is never removed or reset. This lets a Core started on the host be used from the devcontainer.

#### Scenario: External Core healthy
- **WHEN** `CORE_URL` points at a healthy Core and `make test-integration` runs without Docker
- **THEN** the flow runs against that Core and the Core is still running afterwards

#### Scenario: Default inside the devcontainer
- **WHEN** no Docker daemon is reachable and `CORE_URL` is unset
- **THEN** the scripts use the Core on the host and never start or stop one

#### Scenario: Unauthenticated health answer
- **WHEN** `/health` answers `401`
- **THEN** the Core counts as up

#### Scenario: External Core unreachable
- **WHEN** `CORE_URL` does not answer `/health` within the wait limit
- **THEN** the command fails with a message explaining how to start the Core on the host

### Requirement: Core address is configurable
When `CORE_URL` is not set, the scripts SHALL derive the Core URL from `CORE_HOST` (default `localhost`) and `CORE_PORT` (default `8080`), use a Docker daemon given by `DOCKER_HOST`, and pass that URL to Terraform and the acceptance tests. Core configuration SHALL NOT depend on bind-mounting workspace files, so a remote daemon works.

#### Scenario: Sidecar daemon
- **WHEN** `DOCKER_HOST` points at a sidecar and `CORE_HOST=docker CORE_BIND=0.0.0.0` are set
- **THEN** the Core starts on the sidecar and the integration flow reaches it at `http://docker:8080`

### Requirement: Start waits for health
The start command SHALL return only after Core answers `/health`, and SHALL fail within a bounded wait, printing recent Core logs, when it does not.

#### Scenario: Core not healthy
- **WHEN** Core does not become healthy within the wait limit
- **THEN** the command exits non-zero and prints the Core logs

### Requirement: One command runs the integration flow
A single script and `make` target SHALL build the provider, start the Core, run `terraform validate`, `plan`, `apply`, the null-vs-empty drift check and `destroy` from `tests/integration` with explicitly passed URL, key and per-run unique values, and stop the Core on exit, whether it succeeds or fails.

#### Scenario: Passing run
- **WHEN** the integration target runs against a healthy Core
- **THEN** all steps succeed and no Core containers or `terraform.tfvars.json` remain

#### Scenario: Conflicting environment
- **WHEN** `TF_VAR_aidial_base_url` points at another Core
- **THEN** the run still applies only to the ephemeral Core

### Requirement: Acceptance tests run against the ephemeral Core
Go acceptance tests SHALL run only when `TF_ACC` is set and the Core URL and key are provided, and a `make` target SHALL start the Core, run them and stop the Core. Without those variables they SHALL be skipped, so `go test ./...` needs no Docker.

#### Scenario: Plain test run
- **WHEN** `go test ./...` runs without `TF_ACC`
- **THEN** acceptance tests are skipped and the run passes without a Core

#### Scenario: Acceptance run
- **WHEN** the acceptance target runs
- **THEN** the tests create, read back, update and delete resources on the ephemeral Core and the Core is stopped afterwards
