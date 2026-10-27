# Spec Delta

## Purpose

Define how CI runs the `tests/integration` manifests against a throwaway DIAL Core so the check needs no secrets and leaves no shared state.

## ADDED Requirements

### Requirement: Integration runs against an ephemeral Core
The linux integration job SHALL start a DIAL Core and its Valkey as job services, with the `epam/ai-dial-core:latest` Core image, and SHALL apply `tests/integration` only against that Core.

#### Scenario: Merge request pipeline
- **WHEN** a merge request pipeline runs after `build:linux` and `terraform:validate:linux` succeed
- **THEN** one integration job runs automatically, without manual approval and without reading any `DIALX_*` variable

#### Scenario: Core not healthy
- **WHEN** the Core service does not answer `/health` within the wait limit
- **THEN** the job fails before running Terraform

### Requirement: Credentials are job-local
The job SHALL configure Core with a fixed CI key and SHALL pass the Core URL, key and test values to Terraform explicitly so project CI/CD or `TF_VAR_*` variables cannot redirect it to another Core.

#### Scenario: Conflicting project variable
- **WHEN** a project-level `TF_VAR_dial_base_url` points at a real Core
- **THEN** the integration job still applies only to the ephemeral Core

### Requirement: Full lifecycle is exercised
The job SHALL run `terraform validate`, `plan`, `apply`, the null-vs-empty drift check and `destroy`, and SHALL fail if any step fails.

#### Scenario: Drift check
- **WHEN** apply succeeds
- **THEN** `drift_check_null_vs_empty.sh` runs against the ephemeral Core and the job fails if it fails

#### Scenario: Destroy
- **WHEN** the drift check passes
- **THEN** `terraform destroy -auto-approve` succeeds and leaves no managed resources in state

### Requirement: No real-Core integration jobs remain
`.gitlab-ci.yml` SHALL NOT contain integration jobs that target a real DIAL Core, the `TF_INTEGRATION_MANUAL_APPROVAL` variable or `DIALX_*` references.

#### Scenario: Darwin
- **WHEN** `DARWIN_RUNNER_AVAILABLE` is `true`
- **THEN** only `terraform:validate:darwin` is offered; no darwin integration plan/apply/destroy job exists

### Requirement: Debug state is kept
The job SHALL upload the Terraform state files as an artifact even when the job fails.

#### Scenario: Failed apply
- **WHEN** apply fails
- **THEN** the state artifact is still attached to the job
