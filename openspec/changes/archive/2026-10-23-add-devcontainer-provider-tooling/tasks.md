# Tasks

## 1. Image tooling

- [x] 1.1 In `.devcontainer/Dockerfile` add pinned ARGs and install `gopls`, `dlv`, `goimports`, `tfplugindocs`, `tfproviderlint` (built against a newer `golang.org/x/tools`, see design) (`go install` into `GOBIN=/usr/local/go-tools`) and `golangci-lint` (release tarball); set `GOPATH`, `GOBIN`, `PATH`; verify by building the image and running each tool's version/help command
- [x] 1.2 Update `devcontainer.json`: keep `golang.go`, `hashicorp.terraform`, `redhat.vscode-yaml`; add `go.lintTool=golangci-lint` and gopls settings; verify the JSON is valid and extensions load after rebuild

## 2. Terraform dev overrides

- [x] 2.1 Extend `.devcontainer/post-create.sh` to create `~/.terraformrc` with the `dial/dial` override to `<workspace>/dist/linux_amd64` only when absent (notice otherwise); verify by running the script twice and checking the file is unchanged on the second run
- [x] 2.2 Verify end to end: `go build -o dist/linux_amd64/terraform-provider-dial` then `terraform plan` in a minimal config using `dial/dial` (no `init`) loads the local binary

## 3. Task runner, lint and docs generation

- [x] 3.1 Add `Makefile` with `build`, `install-dev`, `test`, `vet`, `fmt`, `lint`, `docs`; verify `make build install-dev test vet` succeed
- [x] 3.2 Add `.golangci.yml`; run `golangci-lint run` and fix trivial findings or narrow the config so it exits 0; verify `make lint`
- [x] 3.3 Verify `make docs` (`tfplugindocs generate --provider-name dial`) runs without errors; do not commit the generated `docs/` output (remove or confirm it is not staged)

- [x] 3.4 Add `markdownlint-cli2` (Dockerfile npm install, `.markdownlint-cli2.jsonc`, `make lint-md`), fix findings in README/AGENTS; verify `make lint-md` exits 0
- [x] 3.5 Move the OpenSpec skills and `/opsx:*` commands from `.claude/` to `~/.claude/`, `git rm` them from the repo and update `AGENTS.md`; verify `.claude/` is gone and `AGENTS.md` names `~/.claude/`

## 4. Debugging

- [x] 4.1 Add `.vscode/launch.json` (test debug configs) and update `.gitignore` to track only that file; verify `git status` shows the file untracked-but-not-ignored and that a breakpoint in a test under `internal/provider` hits in VS Code

## 5. Pre-commit

- [x] 5.1 Add goimports, golangci-lint and tfproviderlint hooks to `.pre-commit-config.yaml`; verify `pre-commit run --all-files` passes

## 6. Documentation

- [x] 6.1 Add a "Devcontainer" section (and TOC entry) to `README.md` covering opening the container, installed tools, dev overrides, debugging and make targets; verify the documented commands run as written
- [x] 6.2 Add the make targets to the Commands section of `AGENTS.md`; verify they match the Makefile

## 7. Integration check

- [x] 7.1 Rebuild the container from scratch and verify each acceptance scenario in `specs/devcontainer-tooling/spec.md`, then `go test ./... -count=1 -race` and `openspec validate add-devcontainer-provider-tooling`
