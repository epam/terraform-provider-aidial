# Spec Delta

## MODIFIED Requirements

### Requirement: Terraform dev overrides
On container creation the devcontainer SHALL ensure `~/.terraformrc` contains a `dev_overrides` entry mapping `alexey-ban/aidial` to the absolute path `<workspace>/dist/linux_amd64`, and SHALL NOT overwrite an existing `~/.terraformrc`.

#### Scenario: First creation
- **WHEN** the container is created and `~/.terraformrc` does not exist
- **THEN** the file is created with the `alexey-ban/aidial` override pointing at the workspace `dist/linux_amd64` directory

#### Scenario: Existing configuration preserved
- **WHEN** `~/.terraformrc` already exists
- **THEN** post-create leaves it unchanged

#### Scenario: Plan against local build
- **WHEN** the provider is built into `dist/linux_amd64` and `terraform plan` runs on a configuration using `alexey-ban/aidial` without `terraform init`
- **THEN** Terraform loads the local binary

### Requirement: Task runner targets
The repository SHALL provide a `Makefile` with `build`, `install-dev`, `test`, `vet`, `fmt`, `lint`, `lint-md` and `docs` targets that wrap the commands documented in `AGENTS.md`.

#### Scenario: Install local build
- **WHEN** `make install-dev` is run
- **THEN** the provider binary is built to `dist/linux_amd64/terraform-provider-aidial`

#### Scenario: Docs generation
- **WHEN** `make docs` is run
- **THEN** `tfplugindocs generate --provider-name aidial` completes without errors and updates `docs/`

#### Scenario: Lint
- **WHEN** `make lint` is run on a clean checkout
- **THEN** `golangci-lint run` exits with status 0
