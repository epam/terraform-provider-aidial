# Design

## Context

Today `terraform:integration:{plan,apply,destroy}:linux` (stages plan/apply/destroy) share the `.terraform_integration_before_script` anchor, which writes `.terraformrc` and `terraform.tfvars.json` from `DIALX_*` variables. Apply also runs `tests/integration/drift_check_null_vs_empty.sh` (needs `curl`, `jq`, `DIAL_BASE_URL`, `DIAL_API_KEY`); state is passed to destroy as an artifact. Separately `stub/docker-compose.yml` runs Core 0.48.1 with filesystem storage and a file-based config, and `terraform:stub-core:apply` uses docker-in-docker. The reference is the `apply-check` job in `ai-dial-platform-config/.gitlab-ci.yml`, which uses GitLab `services` with an entrypoint that writes the Core settings.

## Goals / Non-Goals

**Goals:** integration runs with no secrets against a fresh Core per job; same coverage as before (validate, plan, apply, drift check, destroy).
**Non-Goals:** changing the manifests' intent, replacing the stub job, supporting a real Core from CI.

## Decisions

- **GitLab services, not docker-in-docker.** Matches the reference job and the `kubernetes` runner tag already used by integration jobs; no privileged docker. Services: `valkey` (alias `valkey`) and `epam/ai-dial-core:latest` (alias `dial-core`).
- **Use the `latest` Core tag, unpinned.** Image `epam/ai-dial-core:latest` (the most recent release; the reference job uses the rolling `development` tag, this job deliberately does not), so the provider is checked against the current released Core without being exposed to every upstream commit. Trade-off: results are not reproducible when `latest` moves, and a new release can break unrelated MRs; upstream changes are only seen once released. The stub job stays on 0.48.1.
- **Core config via entrypoint.** The service cannot mount files, so the entrypoint writes a config JSON (key `ci-check-key`, role `admin`, roles `admin`/`default` with empty limits, mirroring `stub/core/config.json`) and a settings JSON, exports `AIDIAL_SETTINGS`, and `exec docker-entrypoint.sh`. Storage settings must make `PUT /v1/models|roles|keys|routes` work with no volume; if Core's default storage does not, set the filesystem storage in the settings file (as in the stub). Resolved during implementation by running the same image locally.
- **One job instead of three.** Core is destroyed with the job, so a separate plan/apply/destroy handoff via artifacts adds nothing. Steps in a single job (stage `apply`): wait for `/health`, `terraform validate`, `plan`, `apply -auto-approve`, drift check, `terraform destroy -auto-approve`. Running destroy keeps coverage of delete paths. The job needs `build:linux` artifacts and `terraform:validate:linux`.
- **Variables.** `-var`/`TF_VAR_*` set in the job's own `variables` or script, using constants (`http://dial-core:8080`, `ci-check-key`, bucket `platform`, role path `tf-dial-ci-${CI_PIPELINE_ID}-${CI_JOB_ID}`, a dummy service key). The reference passes `-var` flags because project CI/CD variables outrank job variables; do the same so stale `TF_VAR_*` project variables cannot redirect the run to a real Core. The tfvars JSON file is no longer written.
- **Keep the tfstate artifact** (`when: always`) for debugging; the job no longer needs to hand it to a destroy job.
- **Darwin integration jobs removed**: they only ever targeted the real Core and cannot run service containers on a macOS runner. `terraform:validate:darwin` stays.
- **Image for the job**: `hashicorp/terraform:1.12` with `apk add curl jq`, as before. The terraform image can reach services by alias; verify the health probe.

## Risks / Trade-offs

- [Ephemeral Core may not accept everything the real one does (e.g. unknown fields, admin-role requirements)] → run the full `tests/integration` set locally against the same image first and adjust config or manifests.
- [Core/Valkey image pulls from Docker Hub/ECR rate limits] → use the ECR mirror for Valkey like the reference; Core image is on Docker Hub as in the stub job.
- [Loses signal on the shared real environment] → accepted by request; the stub job and unit tests remain.
- [Obsolete GitLab CI/CD variables stay configured] → listed as follow-up, not done here.
- [Unpinned `latest` can break unrelated MRs or change behavior between reruns once a new release is published] → accepted by request; on a failure, re-run to rule out a flake and compare against the stub job (0.48.1) to confirm the Core change.
