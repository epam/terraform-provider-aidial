# Proposal

## Why

The `terraform:integration:*` CI jobs run against a shared, real DIAL Core. That requires secrets (`DIALX_TEST_BASE_URL`, `DIALX_TEST_API_KEY`, `DIALX_TEST_ROLE_BUCKET`, `DIALX_SERVICE_KEY`, optional `DIALX_ADMIN_URL`), leaves state in a shared environment, can collide between pipelines and fails when that environment is down. The apply-check job in `ai-dial-platform-config` already shows a better pattern: start a throwaway DIAL Core (plus Valkey) as GitLab CI services, apply against it and discard everything with the job.

## What Changes

- Replace the three linux integration jobs (`terraform:integration:plan:linux`, `apply:linux`, `destroy:linux`) with one job that starts an ephemeral DIAL Core and Valkey as services and, against it, runs `terraform validate`, `plan`, `apply`, the null-vs-empty drift check and `destroy`.
- Core is configured by a settings file the service entrypoint writes at start, with a fixed CI admin key. Credentials are job-local constants, not CI/CD variables.
- Remove the darwin integration plan/apply/destroy jobs (they targeted the real Core and were already disabled by default). The darwin validate job is kept.
- Remove `TF_INTEGRATION_MANUAL_APPROVAL` and every `DIALX_*` secret dependency from `.gitlab-ci.yml`, including the `.terraform_integration_before_script` anchor.
- Update `AGENTS.md` and `README.md` where they say integration manifests run against a real DIAL Core.

Out of scope: `stub/` and `terraform:stub-core:apply`, the `tests/integration/*.tf` manifests (only changed if the ephemeral Core requires it), provider code, the unused `ci/*.gitlab-ci.yml`, deleting the now-unused CI/CD variables from GitLab project settings.

## Capabilities

### New Capabilities
- `ci-integration-core`: the contract for how CI runs the integration manifests against a throwaway DIAL Core.

### Modified Capabilities

## Impact

- `.gitlab-ci.yml` (integration section, variables), `AGENTS.md`, `README.md`.
- Integration now runs on every MR without secrets, so forks and unprivileged pipelines can run it. The test no longer validates behavior of the long-lived real deployment (version skew against it is no longer detected); it validates against the latest released Core image (`epam/ai-dial-core:latest`), so a new upstream Core release can fail an MR unrelated to it.
- Needs a runner that can pull `epam/ai-dial-core` and a Valkey image (the example uses `public.ecr.aws/valkey/valkey:9-alpine`).
