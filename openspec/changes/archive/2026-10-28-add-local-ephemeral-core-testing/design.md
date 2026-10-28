## Context

CI starts Core and Valkey with inline `docker run` and heredoc config (`.github/workflows/ci.yml`, `integration` job). `stub/docker-compose.yml` is a separate, long-lived manual stack (port 8081, pinned Core, persistent data) and must stay separate. The devcontainer (`.devcontainer/devcontainer.json`) has no Docker CLI or daemon, and it must stay unprivileged. `stub/.env.example` already assumes a Docker host named `docker`. Existing Go tests use `httptest`; `terraform-plugin-testing` is not in `go.mod`.

## Goals / Non-Goals

**Goals:** one command in the devcontainer to run the CI integration flow and Go acceptance tests against a fresh Core of a chosen version; one Core definition shared with CI.

**Non-Goals:** changing provider behavior; replacing `stub/`; a Core version matrix in CI; running against non-local Cores.

## Decisions

1. **docker compose in `tests/core/`** (`docker-compose.yml`, `config.json`, `settings.json`), moved from the CI heredocs. Image `epam/ai-dial-core:${CORE_VERSION:-latest}`, Valkey `valkey/valkey:9-alpine`, healthchecks on both, ports published on `${CORE_BIND:-0.0.0.0}:${CORE_PORT:-8080}`. Config is delivered as inline compose `configs` (`content:`) or `AIDIAL_*` environment, not bind mounts, because a remote daemon cannot see workspace files. Fixed key `ci-check-key`, as today. Alternative: reuse `stub/`; rejected (different port, config, persistence, purpose).
2. **Core on the host, devcontainer only connects (recommended).** Any Docker daemon the devcontainer controls is a way out of it (host socket, privileged DinD or a privileged `docker:dind` sidecar all give an agent root-equivalent reach). So the default flow keeps Docker out of the devcontainer: the developer runs `scripts/core.sh up` on the host (published on `0.0.0.0` by default). Without a reachable daemon the scripts default `CORE_URL` to `http://host.docker.internal:8080`, falling back to the default-gateway IP read from `/proc/net/route` when the name does not resolve (before the devcontainer is rebuilt); `CORE_URL` overrides. `devcontainer.json` adds `--add-host=host.docker.internal:host-gateway`. In external mode `scripts/core.sh` only waits for `/health`; `down` is a no-op, so an external Core is never deleted.
   **Self-managed mode.** Without `CORE_URL` and with a reachable daemon (a host with Docker) the scripts start and stop a Core via compose. The devcontainer has no Docker client on purpose: any daemon it controls is an escape path. Rejected: DinD feature (privileged), host socket (host root), privileged sidecar, Sysbox (host prerequisite).
   Without `CORE_URL` the Core URL comes from `CORE_HOST` (default `localhost`) and `CORE_PORT`.

3. **Scripts** `scripts/core.sh up|down` (project name `aidial-test`, `up -d --wait` with a timeout, logs on failure, `down -v --remove-orphans`) and `scripts/test-integration.sh`. The runner builds the provider into a temp dir, writes a temp `.terraformrc` with `dev_overrides`, writes `terraform.tfvars.json` with a per-run unique role path via `-var-file` (not `TF_VAR_*`), runs validate/plan/apply/drift check/destroy, and cleans up tfvars and the Core via `trap`. CI calls the same scripts and keeps its own state-artifact upload.
4. **Acceptance tests** use `terraform-plugin-testing` (`resource.Test`, provider factory, `TF_ACC=1`, `DIAL_BASE_URL`, `DIAL_API_KEY`), in `internal/provider/*_acc_test.go`, covering at least role, key, model and route create/import/update/delete and the absent-resource (404) path. This adds a dependency and requires the `terraform` binary, which the devcontainer and CI have. Fallback if the dependency conflicts with `go.mod`: client-level tests in `internal/client` against the real Core.
5. **Make targets**: `core-up`, `core-down`, `test-integration`, `testacc`; all honour `CORE_VERSION`.
6. CI keeps `latest` by default; no version matrix.

## Risks / Trade-offs

- Contributors need Docker on the host; without a reachable Core or daemon the scripts fail fast with a message pointing at the README.
- The `0.0.0.0` default publish address exposes the throwaway Core to the network; documented, with a firewall recommendation and `CORE_BIND=127.0.0.1` to opt out.
- `/health` answers `401` on some Core versions, so the wait treats any HTTP response as up.
- An external Core keeps state between runs; resources use per-run unique paths and the developer resets it with `down`/`up`.
- Compose `configs` with inline `content` against a remote daemon is assumed to work; verify with the sidecar before relying on it, else fall back to `AIDIAL_*` env for all settings.
- Port 8080 may clash on the daemon host; `CORE_PORT` overrides it, and scripts export the matching URL.
- Moving config out of the workflow touches the CI path; verify by running the job on the branch before merging.
- Acceptance tests share one Core and must use unique paths per test to run safely in parallel.
