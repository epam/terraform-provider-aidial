# Tasks

## 1. Schemas and helpers

- [x] 1.1 In `model_resource.go`, `role_resource.go` and `route_resource.go`, make `bucket` and `path` `Required` (no `Optional`/`Computed`), keep `RequiresReplace`, add `stringvalidator.OneOf("platform")` on `bucket` and `stringvalidator.LengthAtLeast(1)` on `path`, and update the attribute descriptions; verify `go build ./...` passes
- [x] 1.2 Make the same change in `key_resource.go` and `catalog_schema_resource.go` (drop the `stringdefault` on `bucket`; `path` stays required, add `LengthAtLeast(1)`); verify `go build ./...` passes
- [x] 1.3 In `model_data_source.go` and `catalog_schema_data_source.go`, make `bucket` and `path` `Required` with the same validators and remove the `defaultBucket` and `name` fallbacks in `Read`; in `role_data_source.go` add the validators to the already required attributes; verify `go build ./...` passes
- [x] 1.4 Delete `bucketDefaultsToPlatform`, `pathDefaultsToName` and `defaultBucket` from `helpers.go` (keeping a constant for `"platform"` only if the validators use it); verify `grep -rn "bucketDefaultsToPlatform\|pathDefaultsToName\|defaultBucket" internal` finds nothing

## 2. Unit and acceptance tests

- [x] 2.1 Delete `TestPathDefaultsToName` and the `bucketDefaultsToPlatform` assertions in `model_resource_test.go`; add schema tests asserting `bucket` and `path` are required, not computed, and have the validators, for all five resources and the three data sources; verify `go test ./internal/provider -count=1 -race` passes
- [x] 2.2 Add validator tests: omitted `bucket`/`path`, empty `path` and `bucket = "other"` each fail validation; verify the tests pass
- [x] 2.3 Add a plan test that changing only `name` (same `bucket` and `path`) is an in-place update and changing `path` plans replacement; verify the test passes
- [x] 2.4 Update every unit and `TestAcc*` config in `internal/provider/*_test.go` that omits `bucket` or `path` to set both explicitly; verify `go test ./... -count=1 -race` passes and `make testacc` passes against a local Core

## 3. Fixtures, examples and docs

- [x] 3.1 Delete `tests/integration/tf_aidial_integration_model_bucket_path_omitted.tf`, `..._model_path_omitted.tf` and `..._role_bucket_path_omitted.tf`; verify no remaining file or script references them
- [x] 3.2 Add explicit `bucket` and `path` to every `aidial_*` resource and data source block under `tests/integration/`, `stub/models/` (including `.tf.json`) and `examples/` that omits them (known: `stub/models/tf_aidial_integration_role_with_limits.tf` and `.tf.json`, `examples/data-sources/aidial_model/data-source.tf`, `examples/resources/aidial_catalog_schema/resource.tf`, `examples/data-sources/aidial_catalog_schema/data-source.tf`; the `stub/models/*.tf.json` models and the other examples need a scan); verify `terraform fmt -check -recursive` passes and `make test-integration` passes against a local Core
- [x] 3.3 Run `make docs`; verify the generated `docs/` describe `bucket` and `path` as required, `bucket` as accepting only `platform`, and `git diff --exit-code docs` is clean afterwards
- [x] 3.4 Update the identity convention paragraph in `AGENTS.md` (optional `bucket`/`path`, defaults, `resource-identity-defaults` reference) to the required-identity rules; verify `markdownlint-cli2` passes

## 4. Specs

- [x] 4.1 `git mv openspec/resource-identity-defaults openspec/resource-identity` and rewrite its `spec.md` from the delta in this change; verify no file still references `resource-identity-defaults` (`grep -rn resource-identity-defaults . --exclude-dir=archive --exclude-dir=.git`)
- [x] 4.2 Apply the `bucket`/`path` edits and the new scenarios to `openspec/model-schema/spec.md`, `role-schema/spec.md`, `route-schema/spec.md` and `openspec/specs/catalog-schema-resource/spec.md`, and to the `model-data-source` spec once it exists; verify no spec still says `bucket` or `path` is optional, computed or defaulted
- [x] 4.3 Run `openspec validate require-bucket-path-identity --strict`; verify it reports no errors

## 5. Integration checks

- [x] 5.1 Run `go test ./... -count=1 -race`, `test -z "$(gofmt -l .)"`, `go vet ./...` and `make lint`; verify all pass

## Workflow follow-up

- Archive `add-model-data-source` before applying this change, so the `model-data-source` spec exists to modify.
- Archive this change after review with `--skip-specs`: the CLI cannot merge the `model-schema`, `role-schema`, `route-schema` and `model-data-source` deltas (their main specs are not under `openspec/specs/`), so tasks 4.1 and 4.2 apply them by hand.
