# Tasks

## 1. Validate the ephemeral Core locally

- [x] 1.1 Run `epam/ai-dial-core:development` (done before the switch to `:latest`, see 1.3) + Valkey with the entrypoint-written config/settings from design.md (docker compose or `docker run`); confirm `/health` and that `GET /v1/bucket` with the CI key returns 200, adding filesystem storage settings if PUTs fail
- [x] 1.2 Apply `tests/integration` against it with a local build (`dev_overrides`) and run `drift_check_null_vs_empty.sh`, then `terraform destroy`; fix Core config or manifests for any failure, recording what changed
- [x] 1.3 Re-run 1.1 and 1.2 against `epam/ai-dial-core:latest` (it may be older than `development`); record any Core config or manifest differences. Verified by CI job 3359745: Core `:latest` accepted the entrypoint settings and the full lifecycle plus drift check passed with no config or manifest changes

## 2. CI job

- [x] 2.1 In `.gitlab-ci.yml` replace `terraform:integration:{plan,apply,destroy}:linux` and the `.terraform_integration_before_script` anchor with one `terraform:integration:linux` job (stage `apply`) using the `valkey` and `dial-core` services, health wait, explicit `-var` flags, validate/plan/apply/drift check/destroy, state artifact `when: always`, MR rule; verify with `glab ci lint` or the GitLab CI lint API
- [x] 2.2 Delete the three darwin integration jobs and the `TF_INTEGRATION_MANUAL_APPROVAL` variable; update the `DARWIN_RUNNER_AVAILABLE` description; verify `grep -n 'DIALX_\|TF_INTEGRATION_MANUAL\|integration:.*darwin' .gitlab-ci.yml` is empty
- [x] 2.3 Push to an MR and verify the job passes, and that a deliberately broken manifest makes it fail. Verified by CI job 3359878: a deliberately failing postcondition failed the job during apply, after resources were created, and the state artifact was still uploaded (jobs 3359775 and 3359850 had already failed at validate and plan)

## 3. Docs

- [x] 3.1 Update `AGENTS.md` (Commands section, conventions) and `README.md` (layout, separation notes, CI comment) to say integration runs against an ephemeral Core in CI; verify `make lint-md` passes

## 4. Final check

- [x] 4.1 Verify each scenario in `specs/ci-integration-core/spec.md`, `go test ./... -count=1 -race` and `test -z "$(gofmt -l .)"`. Verified: Go tests and gofmt pass; all scenarios are backed by CI jobs 3359745 and 3359878 except "Core not healthy" and "Conflicting project variable", which were verified by reading the job script only (health loop exits 1 after 60 tries; -var-file outranks TF_VAR_*)
