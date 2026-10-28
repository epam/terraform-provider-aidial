# Proposal

## Why

The integration flow (`tests/integration` apply, drift check, destroy) runs only in the GitHub Actions `integration` job, which starts a throwaway DIAL Core with inline `docker run` commands. The devcontainer has no Docker daemon, so contributors cannot test against a real Core and must push and read CI output. The only Go tests are `httptest`-based, so none exercise a real Core.

## What Changes

- Add a docker compose definition of an ephemeral DIAL Core plus Valkey under `tests/core/`, with the Core version selectable via `CORE_VERSION` (default `latest`).
- Add scripts and `make` targets to start and stop that Core and to run the integration flow against it with one command.
- Add Go acceptance tests, gated by `TF_ACC` and a Core URL and key, that run against the ephemeral Core and are skipped otherwise.
- Support an external Core: with `CORE_URL`, the scripts only connect to a Core that the developer started on the host with `scripts/core.sh up`, so the devcontainer (and any agent in it) needs no Docker access. The devcontainer maps `host.docker.internal` to the host.
- The devcontainer has no Docker client and never runs privileged.
- Make the CI `integration` job use the same compose file and runner script instead of inline heredocs and `docker run`.
- Document the workflow in README, AGENTS.md and CONTRIBUTING.

## Capabilities

### New Capabilities
- `local-ephemeral-core`: starting, selecting the version of, and tearing down a throwaway DIAL Core, and running the integration flow and acceptance tests against it.

### Modified Capabilities
- `devcontainer-tooling`: the devcontainer can reach a host-run Core and has no Docker client.
- `ci-integration-core`: the integration job reuses the shared Core definition and runner.

## Impact

- New: `tests/core/` (compose, `config.json`, `settings.json`), `scripts/` runners, `internal/provider/*_acc_test.go`, `Makefile` targets.
- Changed: `.github/workflows/ci.yml`, `go.mod`/`go.sum` (`terraform-plugin-testing`), README, AGENTS.md, CONTRIBUTING.
- Unchanged: provider behavior, `stub/` (long-lived manual stack) and `tests/integration/` manifests.
- Requires Docker on the host, a devcontainer rebuild and network access to pull `epam/ai-dial-core` and `valkey/valkey` images.
