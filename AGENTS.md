# AGENTS.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

The binary is named `terraform-provider-aidial` (module `github.com/epam/terraform-provider-aidial`, Registry address `epam/aidial`). CI and releases build with Go 1.26 (latest patch, for the standard-library fixes `govulncheck` enforces), matching `go.mod`.

```bash
go build -o terraform-provider-aidial
go test ./... -count=1 -race                          # what CI runs
go test ./internal/provider -run TestName -count=1    # single test (use ./internal/client for the client package)
test -z "$(gofmt -l .)"                               # CI format check
go vet ./...
terraform fmt -check -recursive                       # for .tf files (examples/, stub/, tests/)
```

`Makefile` wraps these: `make build|install-dev|test|vet|fmt|lint|lint-md|docs|core-up|core-down|test-integration|testacc` (`install-dev` builds into `dist/linux_amd64` for `dev_overrides`; `lint` is `golangci-lint run` + `tfproviderlint ./...`; `docs` is `tfplugindocs generate --provider-name aidial`; `docs/` is committed and CI fails on drift, so rerun it after schema or `examples/` changes). The devcontainer installs all these tools.

Pre-commit (`.pre-commit-config.yaml`) runs `gofmt -w`, `goimports -w`, `golangci-lint run`, `tfproviderlint ./...`, `markdownlint-cli2` and `terraform fmt -recursive`.

Running Terraform against an unreleased local build: released versions come from the Registry, so use `dev_overrides` in `~/.terraformrc` (or `TF_CLI_CONFIG_FILE`) pointing at an **absolute directory** containing the binary (e.g. `go build -o dist/linux_amd64/terraform-provider-aidial`). Skip `terraform init` while the override is active. See README "Local builds".

Other test environments:

- `stub/` is a disposable local DIAL Core (`stub/docker-compose.yml`); `stub/models/*.tf*` are applied to it by the manually dispatched `Stub Core apply` workflow (`.github/workflows/stub-core.yml`).
- `tests/integration/` holds manifests applied in CI by the `integration` job of `.github/workflows/ci.yml` against an ephemeral DIAL Core (`epam/ai-dial-core:latest` + Valkey containers started by the job, no secrets). `drift_check_null_vs_empty.sh` is run from that directory after `terraform apply` and needs `DIAL_BASE_URL`, `DIAL_API_KEY`, `curl` and `jq`.
- Local ephemeral Core: `tests/core/docker-compose.yml` (version via `CORE_VERSION`, default `latest`) is shared by CI and `make test-integration` / `make testacc` (`scripts/core.sh`, `scripts/test-integration.sh`). The devcontainer has no Docker by default: start the Core on the host with `scripts/core.sh up`; with no Docker daemon the scripts default to `http://host.docker.internal:8080` (override with `CORE_URL`) and only connect to it; see README "Testing against a local Core". Acceptance tests are `TestAcc*` in `internal/provider/*_acc_test.go`, skipped without `TF_ACC`.
- Releases: pushing a `vX.Y.Z` tag runs `.github/workflows/release.yml` (GoReleaser, GPG-signed checksums, Registry manifest; see README "Releasing").

## Architecture

Terraform Plugin Framework provider with two packages:

- `internal/client`: DIAL Core HTTP client (`Api-Key` header on every request).
- `internal/provider`: resources `aidial_role`, `aidial_key`, `aidial_model`, `aidial_route` and data sources `aidial_bucket`, `aidial_role`, `aidial_model` (API-managed models by `bucket`/`path`), `aidial_file_model` (static-config models by name only), `aidial_catalog_schema`, plus shared helpers.

Conventions that span several files:

- **Shared identity model.** All four resources have optional `bucket` (default `platform`) and `path` (default: `name`; required for `aidial_key`, which has no name). Both force replacement, and `id` is `<bucket>/<path>`. The defaults are plan modifiers in `internal/provider/helpers.go`.
- **API semantics.** GET 404 means the resource is absent (not an error), and DELETE 404 is a successful no-op. PUT returns only `{"name": ...}`, so every create/update re-reads via GET to get the real state. ETag/`If-Match` headers are not used. `aidial_key`'s GET never returns the secret `key`, so state must keep the prior sensitive value, and PUT requires a non-empty `key`.
- **Null vs empty stays distinct** for `aidial_model` (and the other resources' collections). When the API omits nested objects/lists/sets, they are normalized to explicit nulls to avoid plan churn. The free-form `defaults` attribute (`model_defaults.go`) keeps the prior value when the API JSON is semantically equal, so config types and formatting survive read-back.
- **Specs first.** Behavior is specified once in `openspec/<capability>/spec.md` (per-resource `*-schema`, `*-client`, `*-add/read/update/delete/import`, `*-state-consistency`). Cross-resource rules live in `openspec/resource-identity-defaults`. Update the spec and the code together. `openapicore/open_api_core.yaml` is the DIAL Core API contract they are checked against. The OpenSpec workflow skills and `/opsx:*` commands are installed per user in `~/.claude/`, not in the repo.
- **Keep `stub/models/` and `tests/integration/` separate.** They run against different backends with different expectations. Don't copy configs into `examples/`, which holds only the snippets embedded in the generated `docs/`.
- **CI:** pipelines are defined only in `.github/workflows/` (`ci.yml`, `stub-core.yml`, `release.yml`); release packaging is `.goreleaser.yml`.

## Commits and merge requests

Use Semantic Commit Messages (`type: subject`, e.g. `feat:`, `fix:`, `build:`, `ci:`, `docs:`, `chore:`) for every commit and for merge request titles. Keep the subject imperative and lowercase after the type, and put detail in the body.
