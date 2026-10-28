# Tasks

## 1. Client request paths

- [ ] 1.1 Add one unexported helper in `internal/client/client.go` that applies `url.PathEscape`. Route `roleURL`, `keyURL`, `modelURL`, `routeURL`, `catalogSchemaURL` and `fileModelURL` through it. Verify that `go build ./...` succeeds.
- [ ] 1.2 Add table-driven client tests with `httptest` and `r.URL.EscapedPath()`, covering the `core-client-requests` scenarios: `team/admin` → `team%2Fadmin`, `a?b` → `a%3Fb` with an empty query, file model `a/b` → `a%2Fb`, and plain names with `-_.:` unchanged. Fix `TestGetFileModelEscapesDotsInName` so it asserts `/v1/admin/config/file/models/a%2Fb`, as its comment describes. Verify with `go test ./internal/client -count=1`.
- [ ] 1.3 Remove the unused `bucket` parameter from `client.GetBucket` and update `bucket_data_source.go` and any tests that call it. Verify with `go vet ./...` and `go test ./... -count=1`.
- [ ] 1.4 Add a `### Fixed` entry under `## Unreleased` in `CHANGELOG.md` for path escaping. Verify the entry exists.

## 2. Data source secrets

- [ ] 2.1 Set `Sensitive: true` on upstream `key` and `secret_extra_data` in `model_data_source.go` and `file_model_data_source.go`. Verify with `go build ./...`.
- [ ] 2.2 Add a unit test that walks the `aidial_model` and `aidial_file_model` data source schemas and asserts that the two attributes are sensitive while `endpoint`, `responses_endpoint`, `id`, `extra_data`, `tier` and `weight` are not. Verify with `go test ./internal/provider -run Sensitive -count=1`.
- [ ] 2.3 Add a `### Security` entry under `## Unreleased` in `CHANGELOG.md`. It should say the values are now redacted, and that root module outputs exposing them need `sensitive = true`. Verify the entry exists.

## 3. `aidial_key` state

- [ ] 3.1 In `key_resource.go`, map `project` and `role` with a helper following design.md:
  - a non-empty API value is stored as is
  - otherwise `""` is kept only when the planned value (Create/Update) or prior state (Read) is exactly `""`
  - otherwise null

  Thread the planned or prior value into `setKeyState`. Verify with `go build ./...`.
- [ ] 3.2 Remove the dead `saved.Key` / `key.Key` assignments in Create, Read and Update, keeping the comment that explains why `state.Key` comes from plan or state. Verify with `go vet ./...`.
- [ ] 3.3 Add unit tests for every `key-state-consistency` scenario: omitted → null, explicit `""` kept, prior `""` kept, removed outside Terraform → null, returned value stored. Verify with `go test ./internal/provider -run Key -count=1`.
- [ ] 3.4 Add a `### Fixed` entry under `## Unreleased` in `CHANGELOG.md`. Verify the entry exists.

## 4. Docs and metadata drift

- [ ] 4.1 Update the Architecture section of `AGENTS.md`:
  - list the `aidial_catalog_schema` resource
  - change "All four resources" to five
  - say that `path` is required for `aidial_key` and `aidial_catalog_schema`
  - say that `bucket` defaults come from plan modifiers in `helpers.go`, or from `stringdefault` for key and catalog schema

  Verify with `markdownlint-cli2 AGENTS.md`.
- [ ] 4.2 Update `README.md`:
  - the description line lists catalog schemas
  - the Installation snippet uses `version = "~> 0.2"` and states "Terraform 1.5 or later is required"
  - the resources and data sources table adds the `aidial_catalog_schema` resource and data source and the `aidial_model` data source
  - the identity paragraph describes five resources

  Verify by checking every type registered in `provider.go` against the table, and with `markdownlint-cli2 README.md`.
- [ ] 4.3 Update `CHANGELOG.md`: add the `aidial_model` data source to the 0.2.0 `### Added` list, and keep the `## Unreleased` heading above 0.2.0. Verify that the file matches the "CHANGELOG tracks every shipped change" scenarios.
- [ ] 4.4 Add `examples/resources/<type>/import-by-string-id.tf` for `aidial_role`, `aidial_key`, `aidial_model`, `aidial_route` and `aidial_catalog_schema`. Each one uses the same `<bucket>/<path>` as that type's `import.sh`. Add `version = "~> 0.2"` to `examples/provider/provider.tf`. Verify with `terraform fmt -check -recursive examples`.
- [ ] 4.5 Regenerate the docs with `go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs@v0.22.0 generate --provider-name aidial`, and commit the `docs/` changes. Verify that each resource page shows both import forms, that the two data source pages mark the upstream secrets sensitive, and that a second run leaves `git status --porcelain -- docs/` empty.

## 5. Release pipeline

- [ ] 5.1 In `.github/workflows/release.yml`, before GoReleaser:
  - add a step that extracts the `## <version>` section of `CHANGELOG.md` (the tag minus `v`) into `$RUNNER_TEMP/release-notes.md`, and fails when the section is missing or empty
  - pass the file through `--release-notes`

  Verify locally by running the step's script against the 0.2.0 tag (non-empty output) and against an unknown version (exit code ≠ 0).
- [ ] 5.2 Pin `goreleaser-action` to `version: v2.18.3` and `download-syft` to `syft-version: v1.54.1`. Add a comment pointing to the matching pins in `.devcontainer/Dockerfile` and `ci.yml`. Verify that `grep -n '~>' .github/workflows/release.yml` returns nothing.
- [ ] 5.3 Add `timeout-minutes: 30` to the job, and a workflow-level `concurrency` with `group: release-${{ github.ref }}` and `cancel-in-progress: false`. Verify by inspection, and with `zizmor .github/workflows/release.yml`.
- [ ] 5.4 After the attestation step, add the verification steps:
  - `gh release download "$GITHUB_REF_NAME"` into a fresh directory, with `GH_TOKEN` set to `github.token`
  - `gpg --verify` the `_SHA256SUMS.sig`
  - `sha256sum -c` the `_SHA256SUMS`, without `--ignore-missing`
  - `gh attestation verify` the `linux_amd64` zip with `--repo "$GITHUB_REPOSITORY"`

  Verify with `zizmor` and `actionlint` (if available), and by dry-running the shell against a downloaded v0.2.0 release for the gpg and sha256 parts.
- [ ] 5.5 In `.github/workflows/ci.yml`, add a `goreleaser check` step to the `lint` job using the same SHA-pinned `goreleaser-action` and `version: v2.18.3`. Verify locally with `goreleaser check`, and that `zizmor .github/workflows/ci.yml` reports no new findings.
- [ ] 5.6 Update the Releasing section of `README.md`:
  - step 1 moves the `Unreleased` entries into `## X.Y.Z - YYYY-MM-DD`
  - step 2 uses `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`
  - step 3 mentions the verification steps and the environment approval
  - the example version becomes a placeholder, not `v0.1.0`

  Verify with `markdownlint-cli2 README.md`.
- [ ] 5.7 Add a `### Changed` entry under `## Unreleased` in `CHANGELOG.md` for the release-notes and verification changes. Verify the entry exists.

## 6. SECURITY.md

- [ ] 6.1 Fix the "Repository safeguards" sentence so it names the jobs that run on the weekly schedule in `security.yml` (govulncheck, CodeQL, gitleaks and zizmor; dependency review runs on pull requests only). Verify against the `schedule` trigger and the job `if:` conditions.
- [ ] 6.2 Rewrite "Settings maintainers apply in GitHub" as the seven settings from the modified "Documented GitHub settings" requirement:
  - branch protection with pull requests and the `CI` and `Security` checks required, plus a note to add code owner review once there is a second maintainer
  - tag protection for `v*`
  - secret scanning with push protection
  - private vulnerability reporting
  - a read-only workflow token that cannot approve pull requests
  - required SHA pinning
  - the `release` environment with required reviewers and the signing secrets

  Verify with `markdownlint-cli2 SECURITY.md` and by checking each item against the requirement.

## 7. Integration checks

- [ ] 7.1 Run the CI-equivalent checks and confirm they all pass:
  - `go build ./...`
  - `go vet ./...`
  - `test -z "$(gofmt -l .)"`
  - `go test ./... -count=1 -race`
  - `golangci-lint run`
  - `tfproviderlint ./...`
  - `terraform fmt -check -recursive`
  - `markdownlint-cli2`
  - the docs drift check
- [ ] 7.2 Run `make test-integration` against a local or host Core (see the README "Testing against a local Core"), and confirm that apply, the drift check and destroy pass with the new escaping.
- [ ] 7.3 Run `openspec validate fix-data-source-secrets-and-drift --strict` and confirm it passes.

## Workflow follow-up

- The maintainer applies the GitHub settings listed in `SECURITY.md`. At minimum, the `release` environment must hold `GPG_PRIVATE_KEY` and `PASSPHRASE` before the tag is pushed.
- Move the `Unreleased` entries into `## 0.2.1 - <date>`, push the annotated tag `v0.2.1`, approve the environment deployment, and confirm that the verification steps pass and the Registry lists 0.2.1.
- Archive this change after the release is confirmed.
