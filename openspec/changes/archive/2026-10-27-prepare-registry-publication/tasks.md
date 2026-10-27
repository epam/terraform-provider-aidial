# Tasks

## 1. Pre-publication audit

- [x] 1.1 Search the tree (excluding `.git`, `openapicore/`) for internal hosts, GitLab paths, `@epam.com` addresses, tokens and keys; record findings and resolutions in the change notes. Verify with `gitleaks detect --no-git` reporting no unresolved findings.
- [x] 1.2 Review `stub/.env.example`, `stub/core/config.json`, `stub/settings/` and `.devcontainer/` for anything unsuitable for a public repo; fix or justify. Verify by re-running the search from 1.1.

## 2. Rename provider and module

- [x] 2.1 Rewrite `scripts/rename-provider-name.sh` replacement list (module path, `registry.terraform.io/dial/dial` and `dial/dial`, `DialProvider`, `dial_` type prefixes and variables, `tf_dial_` file names via `git mv`; exclude `openapicore/` and `openspec/changes/archive/`) and run it. Verify with `git diff --stat` reviewed and `git status` showing renames, not deletions.
- [x] 2.2 Update `go.mod` module path, `main.go` provider address (`registry.terraform.io/alexey-ban/aidial`) and `provider.go` type name `aidial`. Verify with `go build -o terraform-provider-aidial` and `go vet ./...`.
- [x] 2.3 Update Go tests for the renamed types. Verify with `go test ./... -count=1 -race` passing without assertion changes beyond names.
- [x] 2.4 Update `Makefile`, `.devcontainer/*`, `.vscode/launch.json`, `scripts/install-provider.sh`, `.golangci.yml`, `.pre-commit-config.yaml`, `.gitignore` for binary name and address. Verify with `make build`, `make install-dev` producing `dist/linux_amd64/terraform-provider-aidial`, and `make lint`.
- [x] 2.5 Update `stub/models/**` and `tests/integration/**` HCL (resource types, `required_providers`, variable names) and `drift_check_null_vs_empty.sh` if it names resources. Verify with `terraform fmt -check -recursive` and, using `dev_overrides` plus `stub/docker-compose.yml`, `terraform validate` and `plan` succeeding.
- [x] 2.6 Update main OpenSpec specs under `openspec/*/spec.md` (not `changes/archive/`) and `AGENTS.md` to the new names. Verify with a search for `dial_`, `dial/dial`, `dialx` and `terraform-provider-dial` returning only legitimate hits (`ai-dial-core`, `DIAL Core`).

## 3. Docs, examples and community files

- [x] 3.1 Restructure `examples/` into `examples/provider/provider.tf`, `examples/resources/aidial_{role,key,model,route}/{resource.tf,import.sh}` and `examples/data-sources/aidial_{bucket,role,file_model}/data-source.tf`, removing the flat files and the stray `openspec-dial-key-fix-example.md`. Verify with `terraform fmt -check -recursive examples`.
- [x] 3.2 Generate and commit `docs/` with `tfplugindocs generate --provider-name aidial`; drop any docs ignore rule. Verify with a second generate run leaving `git diff --exit-code docs/` clean.
- [x] 3.3 Rewrite `README.md`: Registry install with `source = "alexey-ban/aidial"` and version constraint, resources table with new names, keep a development section (devcontainer, `dev_overrides`, make targets, OpenAPI refresh). Verify with `make lint-md` and a search showing no "not published" text.
- [x] 3.4 Add `LICENSE` (Apache-2.0), `CHANGELOG.md` (first entry `0.1.0`), `SECURITY.md`, `CONTRIBUTING.md` (semantic commits, tests, spec-first workflow). Verify with `make lint-md` and file presence.

## 4. Release pipeline

- [x] 4.1 Add `terraform-registry-manifest.json` with `{"version": 1, "metadata": {"protocol_versions": ["6.0"]}}`. Verify the version against the framework protocol (`terraform-plugin-framework` v1.9.0 serves protocol 6).
- [x] 4.2 Add `.goreleaser.yml` (builds, ignores, zip archives, SHA256SUMS, GPG signing, manifest extra file, GitHub release). Verify with `goreleaser check` and `goreleaser release --snapshot --clean` yielding correctly named zips, checksum file and manifest in `dist/`.
- [x] 4.3 Add `.github/workflows/release.yml` triggered on `v*.*.*` tags, importing the GPG key and running GoReleaser, failing when secrets are missing. Verify with `actionlint` and a read-through against the registry-release spec scenarios.

## 5. CI port

- [x] 5.1 Add `.github/workflows/ci.yml` with format, vet, test, golangci-lint, tfproviderlint, markdownlint, `terraform fmt`, docs drift, Linux and macOS build and validate jobs. Verify with `actionlint` and running each command locally.
- [x] 5.2 Port the ephemeral-Core integration job (Core and Valkey service containers, health wait, fixed CI key, explicit `-var` values, validate, plan, apply, drift check, destroy, state artifact on failure) and an opt-in stub-core workflow. Verify with `actionlint` and by running the same steps locally against `epam/ai-dial-core` with Docker.
- [x] 5.3 Add `.github/dependabot.yml` (gomod, github-actions) and PR template if desired; delete `.gitlab-ci.yml`; update `AGENTS.md` CI and release sections and the README. Verify with a search showing no `.gitlab-ci` or GitLab references outside the archive.

## 6. Final checks

- [x] 6.1 Run `go build ./... && go test ./... -count=1 -race`, `test -z "$(gofmt -l .)"`, `go vet ./...`, `make lint`, `make lint-md`, `terraform fmt -check -recursive`.
- [x] 6.2 Run the legacy-name search and the gitleaks scan one last time; run `openspec validate prepare-registry-publication --strict`.
- [x] 6.3 Write the manual publication checklist into `CONTRIBUTING.md` or `docs/RELEASING.md`: create the repo, GPG key, Registry namespace key upload, GitHub secrets, tag `v0.1.0`, Registry publish. Verify it matches the steps in design.md.

## Workflow follow-up

- Owner steps after this change is applied, not performed by the agent: wipe git history, set `origin` to `git@github.com:alexey-ban/terraform-provider-aidial.git`, push, add secrets, push tag `v0.1.0`, publish on the Registry.
- Archive the change after the review requirements are satisfied.
