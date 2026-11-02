# Tasks

## 1. Data source implementation

- [x] 1.1 Add `internal/provider/model_data_source.go` with the schema (copy of `aidial_file_model` plus `bucket`, `path`, `id`), `Configure` and `Read` (defaults, `GetModel`, `setModelState`, not-found error); verify `go build ./...` and `go vet ./...` pass
- [x] 1.2 Register `modelDataSource` in `DataSources` in `internal/provider/provider.go`; verify a unit test asserts `aidial_model` is in the registered data sources
- [x] 1.3 Add unit tests against a fake client server: defaults applied, explicit bucket/path, existing model populates `id`/`api_name`/catalog attributes, omitted collections follow the resource (null objects/lists/sets, empty maps), 404 yields a not-found diagnostic, non-2xx carries status and body; verify `go test ./internal/provider -run ModelDataSource -count=1 -race` passes
- [x] 1.4 Add a unit test that every computed attribute of `aidial_file_model` also exists in `aidial_model`; verify it passes and fails when an attribute is removed

## 2. Acceptance, docs and examples

- [x] 2.1 Add a `TestAcc*` in `internal/provider/model_data_source_acc_test.go` that creates an `aidial_model` resource and checks the data source values equal the resource's; verify it is skipped without `TF_ACC` and passes with `make testacc` against a local Core
- [x] 2.2 Add `examples/data-sources/aidial_model/data-source.tf`; verify `terraform fmt -check -recursive` passes
- [x] 2.3 Run `make docs`; verify `docs/data-sources/model.md` is generated and `git diff --exit-code docs` is clean afterwards
- [x] 2.4 Update `AGENTS.md` data source list to include `aidial_catalog_schema` and `aidial_model`; verify `markdownlint-cli2` passes

## 3. Integration checks

- [x] 3.1 Run `go test ./... -count=1 -race`, `test -z "$(gofmt -l .)"`, `make lint`; verify all pass
- [x] 3.2 Run `openspec validate add-model-data-source --strict`; verify it reports no errors

## Workflow follow-up

- Archive the change after review; the `model-data-source` spec is then created from the delta.
