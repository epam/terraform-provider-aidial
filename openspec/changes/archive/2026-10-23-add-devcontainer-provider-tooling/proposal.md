# Proposal

## Why

The devcontainer already provides Go, Terraform and pre-commit, but a contributor still has to install the provider-specific tooling (docs generator, linters, debugger), wire up `dev_overrides` and write Make-less command lines by hand before they can iterate on the provider. A fresh container should be ready to build, lint, document, debug and run Terraform against a local build with no manual setup.

## What Changes

- Install `tfplugindocs`, `golangci-lint`, `tfproviderlint`, `markdownlint-cli2`, `dlv`, `goimports` and `gopls` in the devcontainer image at pinned versions, on `PATH`.
- Set `GOPATH`/`GOBIN` so installed tools and built binaries are on `PATH`.
- Generate `~/.terraformrc` with `dev_overrides` pointing at the local build directory (`dist/linux_amd64`) on first container creation, without overwriting an existing file.
- Add `.vscode/launch.json` (tracked, un-ignored) with dlv configurations for debugging tests and the provider build, plus gopls/golangci-lint settings in `devcontainer.json`.
- Add a `Makefile` (`build`, `install-dev`, `test`, `vet`, `fmt`, `lint`, `lint-md`, `docs`), a minimal `.golangci.yml` and a `.markdownlint-cli2.jsonc`.
- Extend `.pre-commit-config.yaml` with goimports, golangci-lint, tfproviderlint and markdownlint hooks alongside the existing gofmt and terraform fmt.
- Move the OpenSpec workflow skills and `/opsx:*` commands out of the repo (`.claude/`) into the user's `~/.claude/`; `AGENTS.md` says so.
- Document devcontainer usage in `README.md` and the new make targets in `AGENTS.md`.

Out of scope: provider code or schema changes (including adding a `-debug`/reattach flag to `main.go`), the active `.gitlab-ci.yml`, committing generated `docs/` output, linting the generated `openspec/` Markdown, registry publishing.

## Capabilities

### New Capabilities
- `devcontainer-tooling`: the development environment contract for the devcontainer: required tools, environment variables, Terraform dev overrides, debugging, task runner targets, pre-commit hooks and contributor documentation.

### Modified Capabilities

## Impact

- `.devcontainer/Dockerfile`, `.devcontainer/devcontainer.json`, `.devcontainer/post-create.sh`
- New: `Makefile`, `.golangci.yml`, `.vscode/launch.json`; `.gitignore` (un-ignore `.vscode/launch.json`)
- `.markdownlint-cli2.jsonc`; removal of the 12 tracked files under `.claude/`
- `.pre-commit-config.yaml`, `README.md`, `AGENTS.md`
- Formatting-only fixes: goimports in `internal/provider/bucket_data_source.go` and `role_data_source.go`, markdownlint `--fix` in `README.md` and `AGENTS.md`.
- Contributors and fresh clones no longer get the OpenSpec commands from the repo and must install them in `~/.claude/`.
- Container image grows and rebuild requires network access to the Go proxy/GitHub at build time.
