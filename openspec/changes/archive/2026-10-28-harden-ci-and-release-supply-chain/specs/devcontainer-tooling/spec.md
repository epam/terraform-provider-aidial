# Spec Delta

## MODIFIED Requirements

### Requirement: Provider development tools are installed
The devcontainer SHALL provide `go`, `terraform`, `gopls`, `dlv`, `goimports`, `golangci-lint`, `tfplugindocs`, `tfproviderlint`, `markdownlint-cli2`, `govulncheck`, `zizmor`, `goreleaser` and `syft` on `PATH`, each pinned to an explicit version, and SHALL verify the SHA-256 of the `zizmor`, `goreleaser` and `syft` downloads at build time.

#### Scenario: Tools available after build
- **WHEN** a new container is created and a terminal is opened
- **THEN** `terraform version`, `gopls version`, `dlv version`, `goimports -h`, `golangci-lint version`, `tfplugindocs --help` and `tfproviderlint -h` and `markdownlint-cli2 --help` all run successfully
- **THEN** `govulncheck -version`, `zizmor --version`, `goreleaser --version` and `syft version` also run successfully

#### Scenario: Build works without setup
- **WHEN** `go build ./...` is run in a new container
- **THEN** it succeeds without installing anything further

#### Scenario: Checksum mismatch
- **WHEN** a downloaded `zizmor`, `goreleaser` or `syft` archive does not match its pinned SHA-256
- **THEN** the image build fails
