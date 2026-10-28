## 1. Devcontainer

- [x] 1.1 Keep the devcontainer free of a Docker client and unprivileged
- [x] 1.2 Map `host.docker.internal` to the host via `runArgs`; default `CORE_URL` to it (gateway fallback) when no daemon is reachable
- [x] 1.3 After a devcontainer rebuild, confirm `host.docker.internal` resolves and `scripts/core.sh up` finds a Core started on the host

## 2. Shared Core definition

- [x] 2.1 Create `tests/core/docker-compose.yml` with Core (`CORE_VERSION`, default `latest`) and Valkey, healthchecks, ports on `${CORE_BIND}:${CORE_PORT}`
- [x] 2.2 Move the Core config and settings from the CI heredocs into `tests/core/` as inline compose `configs`/env (no bind mounts)
- [x] 2.3 Add `scripts/core.sh up|down|url` honouring `CORE_URL` (external Core: wait for health only, `down` no-op), `CORE_HOST`/`CORE_PORT`/`CORE_BIND`, failing fast without a daemon, with bounded health wait, log dump on failure and `down -v --remove-orphans`

## 3. Integration runner

- [x] 3.1 Add `scripts/test-integration.sh` (build provider, temp `.terraformrc`, `-var-file`, validate/plan/apply/drift check/destroy, `trap` cleanup)
- [x] 3.2 Switch the CI `integration` job to `scripts/core.sh` and `scripts/test-integration.sh`, keeping the state artifact upload and credentials cleanup

## 4. Acceptance tests

- [x] 4.1 Add `terraform-plugin-testing` to `go.mod` and a shared acceptance test helper gated by `TF_ACC`, `DIAL_BASE_URL`, `DIAL_API_KEY`
- [x] 4.2 Add acceptance tests for `aidial_role`, `aidial_key`, `aidial_model`, `aidial_route` (create, import, update, destroy, absent resource)
- [x] 4.3 Confirm `go test ./... -count=1 -race` skips them without `TF_ACC`

## 5. Make targets and docs

- [x] 5.1 Add `core-up`, `core-down`, `test-integration`, `testacc` to the Makefile
- [x] 5.2 Document the local workflow and `CORE_VERSION` in README, AGENTS.md commands and CONTRIBUTING; run `markdownlint-cli2`

## 6. Verification

- [x] 6.1 Run `make test-integration` and `make testacc` in the devcontainer against a Core started on the host (external mode); the external Core survives
- [ ] 6.1b Run the self-managed mode (`CORE_VERSION=<tag> make test-integration` on a host with Docker) and confirm containers are removed
- [x] 6.2 Run `gofmt`, `go vet`, `terraform fmt -check -recursive` and `openspec validate add-local-ephemeral-core-testing`
- [ ] 6.3 Push and confirm the CI integration job passes with the shared scripts
