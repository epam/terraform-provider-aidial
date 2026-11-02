# Tasks

## 1. Contract and client

- [x] 1.1 Keep the pending `openapicore/SOURCE.md` and `open_api_core.yaml` refresh to tag 0.49.4 in the working tree for the final commit; verify `git diff openapicore` shows only the `info.version` and provenance changes
- [x] 1.2 Add `GetApplicationTypeSchema`, `SaveApplicationTypeSchema`, `DeleteApplicationTypeSchema` and `GetFileApplicationTypeSchema` to `internal/client/client.go` (URLs via `escapeSegment`, `name`/`status` stripped); verify `go build ./...` passes
- [x] 1.3 Add client unit tests in the style of `catalog_schema_test.go` and `file_model_test.go` (headers, 404 nil, delete 404 success, metadata stripped, `status` returned, escaped paths `/v1/schemas/platform/apps%2Fquick` and `/v1/admin/config/file/schemas/a%2Fb`); verify `go test ./internal/client -count=1 -race` passes

## 2. Provider

- [x] 2.1 Extract the pure schema helpers (parse with `$id`, validator, keep-prior state mapping) from the catalog schema files only if the move is mechanical; verify the existing catalog schema tests pass unchanged, otherwise copy them into the new files
- [x] 2.2 Add `internal/provider/application_type_schema_resource.go` (schema, validators, CRUD, import `<bucket>/<path>`, `schema_id` plan modifier) and `application_type_schema_data_source.go`; verify `go vet ./...` passes
- [x] 2.3 Add `internal/provider/file_application_type_schema_data_source.go` (`name` required, computed `schema`, `schema_id`, `status`, not-found error)
- [x] 2.4 Register the resource and both data sources in `internal/provider/provider.go`; verify a unit test asserts all three type names are registered
- [x] 2.5 Add provider unit tests against a fake server mirroring `catalog_schema_resource_test.go`: invalid schema, immutable identity, create re-read, key-order no-op, out-of-band deletion, invalid import ID, data source found/missing, file data source found/missing/status; verify `go test ./internal/provider -count=1 -race` passes

## 3. Acceptance, integration, docs

- [x] 3.1 Add `TestAcc*` tests in `internal/provider/*_acc_test.go` for the resource (create, update, import) and both data sources; verify they skip without `TF_ACC` and pass with `make testacc` against a local Core
- [x] 3.2 Add `tests/integration/tf_aidial_integration_application_type_schema.tf`; leave `drift_check_null_vs_empty.sh` untouched; verify `make test-integration` passes
- [x] 3.3 Add `examples/resources/aidial_application_type_schema/` (`resource.tf`, `import.sh`, import-by-string-id), `examples/data-sources/aidial_application_type_schema/data-source.tf` and `examples/data-sources/aidial_file_application_type_schema/data-source.tf`; verify `terraform fmt -check -recursive` passes
- [x] 3.4 Run `make docs`; verify `git diff --exit-code docs` is clean afterwards
- [x] 3.5 Update the README resource and data source tables and identity paragraph, the `AGENTS.md` architecture and identity bullets (resources and data sources list), and add a `CHANGELOG.md` entry under `Unreleased`; verify `make lint-md` passes

## 4. Final checks

- [x] 4.1 Run `go test ./... -count=1 -race`, `test -z "$(gofmt -l .)"`, `go vet ./...`, `make lint`; verify all pass
- [x] 4.2 Run `openspec validate add-application-type-schema --strict`; verify it reports no errors
- [x] 4.3 Commit the openapicore refresh and the feature using `feat:` semantic commit(s); verify `git status` is clean

## Workflow follow-up

- Archive the change after review so the two new capabilities are created and the two modified ones updated.
- Candidate follow-up change: `aidial_file_catalog_schema` data source for `/v1/admin/config/file/catalog_schemas/{name}`.
