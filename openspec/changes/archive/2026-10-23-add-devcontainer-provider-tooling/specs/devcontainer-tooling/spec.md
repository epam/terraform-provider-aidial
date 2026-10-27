# Spec Delta

## Purpose

Define the tooling and environment a freshly created devcontainer provides so a contributor can build, lint, document, debug and locally test the Terraform provider without manual setup.

## ADDED Requirements

### Requirement: Provider development tools are installed
The devcontainer SHALL provide `go`, `terraform`, `gopls`, `dlv`, `goimports`, `golangci-lint`, `tfplugindocs`, `tfproviderlint` and `markdownlint-cli2` on `PATH`, each pinned to an explicit version.

#### Scenario: Tools available after build
- **WHEN** a new container is created and a terminal is opened
- **THEN** `terraform version`, `gopls version`, `dlv version`, `goimports -h`, `golangci-lint version`, `tfplugindocs --help` and `tfproviderlint -h` and `markdownlint-cli2 --help` all run successfully

#### Scenario: Build works without setup
- **WHEN** `go build ./...` is run in a new container
- **THEN** it succeeds without installing anything further

### Requirement: Go environment variables
The devcontainer SHALL set `GOPATH` and `GOBIN` so that tools installed with `go install` are on `PATH` for the `node` user.

#### Scenario: GOBIN on PATH
- **WHEN** `go install` places a binary in `GOBIN`
- **THEN** it is runnable by name from a new terminal

### Requirement: Terraform dev overrides
On container creation the devcontainer SHALL ensure `~/.terraformrc` contains a `dev_overrides` entry mapping `dial/dial` to the absolute path `<workspace>/dist/linux_amd64`, and SHALL NOT overwrite an existing `~/.terraformrc`.

#### Scenario: First creation
- **WHEN** the container is created and `~/.terraformrc` does not exist
- **THEN** the file is created with the `dial/dial` override pointing at the workspace `dist/linux_amd64` directory

#### Scenario: Existing configuration preserved
- **WHEN** `~/.terraformrc` already exists
- **THEN** post-create leaves it unchanged

#### Scenario: Plan against local build
- **WHEN** the provider is built into `dist/linux_amd64` and `terraform plan` runs on a configuration using `dial/dial` without `terraform init`
- **THEN** Terraform loads the local binary

### Requirement: Dependencies pre-installed
Post-create SHALL run `go mod download` and install the pre-commit hooks.

#### Scenario: Offline build after create
- **WHEN** the container has been created
- **THEN** Go module dependencies are in the module cache

### Requirement: Editor integration
The devcontainer SHALL install the Go (`golang.go`), HashiCorp Terraform (`hashicorp.terraform`) and YAML (`redhat.vscode-yaml`) VS Code extensions and configure Go to use `gopls` and `golangci-lint`.

#### Scenario: Extensions installed
- **WHEN** the devcontainer is opened in VS Code
- **THEN** the three extensions are installed in the container

### Requirement: Debugging with dlv
The repository SHALL include a tracked VS Code launch configuration that debugs Go tests with dlv.

#### Scenario: Debug a test
- **WHEN** the user starts the test debug configuration with a breakpoint in `internal/provider`
- **THEN** execution stops at the breakpoint

### Requirement: Task runner targets
The repository SHALL provide a `Makefile` with `build`, `install-dev`, `test`, `vet`, `fmt`, `lint`, `lint-md` and `docs` targets that wrap the commands documented in `AGENTS.md`.

#### Scenario: Install local build
- **WHEN** `make install-dev` is run
- **THEN** the provider binary is built to `dist/linux_amd64/terraform-provider-dial`

#### Scenario: Docs generation
- **WHEN** `make docs` is run
- **THEN** `tfplugindocs generate` completes without errors

#### Scenario: Lint
- **WHEN** `make lint` is run on a clean checkout
- **THEN** `golangci-lint run` exits with status 0

### Requirement: Markdown linting
The repository SHALL provide a `markdownlint-cli2` configuration that lints repository Markdown (excluding generated `openspec/` content) and passes on a clean checkout.

#### Scenario: Lint Markdown
- **WHEN** `make lint-md` is run on a clean checkout
- **THEN** it exits with status 0

### Requirement: Pre-commit checks
`.pre-commit-config.yaml` SHALL run gofmt, goimports, golangci-lint, tfproviderlint, markdownlint and `terraform fmt` on commit.

#### Scenario: Unformatted Go file
- **WHEN** a commit includes a Go file with unsorted imports
- **THEN** the hook rewrites or rejects it

### Requirement: Contributor documentation
`README.md` SHALL describe how to open the devcontainer, which tools it provides, how dev overrides are configured, and the make targets.

#### Scenario: Documented usage
- **WHEN** a contributor reads the devcontainer section
- **THEN** they can build the provider and run Terraform against it using only the documented steps

### Requirement: OpenSpec tooling is per-user
The repository SHALL NOT ship the OpenSpec workflow skills or `/opsx:*` commands; `AGENTS.md` SHALL state that they are installed in the user's `~/.claude/`.

#### Scenario: Fresh clone
- **WHEN** the repository is freshly cloned
- **THEN** it contains no `.claude/skills/openspec-*` or `.claude/commands/opsx` files

#### Scenario: Documented location
- **WHEN** a contributor reads the OpenSpec note in `AGENTS.md`
- **THEN** it points to `~/.claude/` as the install location
