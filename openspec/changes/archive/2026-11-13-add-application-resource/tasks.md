# Tasks

## 1. Verify against a real Core

- [x] 1.1 Start the ephemeral Core (`scripts/core.sh up`) and probe applications with throwaway objects: which of `/v1/applications/{bucket}/{path}` and `/v1/applications/platform/{path}` serve GET, PUT and DELETE; per-field JSON key casing (the `Application` schema mixes camelCase and snake_case); minimum payload (`{}` and `endpoint` only); `bucket` values (`platform`, `public`); slashes in `path`; values Core fills in on a minimal PUT; whether `author`/`createdAt`/`updatedAt` come back; whether Core ships a static application; delete the probes; record findings in `design.md` and update the `application-resource`, `resource-identity` and `core-client-requests` deltas where they differ

## 2. Client

- [x] 2.1 Add `Application` type and `GetApplication`, `SaveApplication`, `DeleteApplication`, `GetFileApplication` to `internal/client/application.go` (reusing `ModelFeatures`, `ModelInterface` and the generic helpers, as in `interceptor.go`); verify `go build ./...` passes
- [x] 2.2 Add client unit tests in the style of `interceptor_test.go` (headers, no ETag, 404 nil, delete 404, PUT `{"name"}`, error with status and body, encoded paths, key casing, omitted fields); verify `go test ./internal/client -count=1 -race` passes

## 3. Resource

- [x] 3.1 Add `internal/provider/application_resource.go`: schema per the `application-resource` spec (`bucket` via `platformBucket`, non-empty `path`, required `endpoint`), plan/state conversion with null-vs-empty rules, shared helpers for `features`/`interfaces`, `model_defaults.go` for free-form objects, Core-defaulted attributes optional and computed per task 1.1; verify `go vet ./...` passes
- [x] 3.2 Implement Create, Read, Update, Delete, ImportState (`<bucket>/<path>`) with PUT then GET and 404 removal; register in `internal/provider/provider.go`
- [x] 3.3 Add `application_resource_test.go` against a fake server mirroring `interceptor_resource_test.go`, plus `interceptors` set and `application_type_schema_id` cases; verify `go test ./internal/provider -count=1 -race` passes

## 4. Data sources

- [x] 4.1 Add `application_data_source.go` and register it; verify unit tests for found, missing, API error and resource parity pass
- [x] 4.2 Add `file_application_data_source.go` and register it; verify unit tests for found, missing, empty `name` and a registration test for the three new type names pass
- [x] 4.3 Extend `identity_test.go` to cover the application resource and data source; verify tests pass

## 5. Integration and acceptance

- [x] 5.1 Add `TestAcc*` in `internal/provider/application_resource_acc_test.go` (create, update, import, data source; file data source only if Core ships a static application) and `tests/integration/tf_aidial_integration_application.tf`; verify `make testacc` and `make test-integration` pass and acceptance tests skip without `TF_ACC`

## 6. Docs and final checks

- [x] 6.1 Add `examples/resources/aidial_application/` (`resource.tf`, `import.sh`, `import-by-string-id.tf`) and `examples/data-sources/aidial_application/`, `aidial_file_application/`; verify `terraform fmt -check -recursive` passes
- [x] 6.2 Run `make docs`; verify `git diff --exit-code docs` is clean afterwards and new pages exist
- [x] 6.3 Update README (tables, Application section noting unexposed fields, import, spec pointers), `CHANGELOG.md` under `Unreleased`, and `AGENTS.md` bullets; verify `make lint-md` passes
- [x] 6.4 Run `go test ./... -count=1 -race`, `test -z "$(gofmt -l .)"`, `go vet ./...`, `make lint` and `openspec validate add-application-resource --strict`; verify all pass
- [x] 6.5 Commit with semantic messages (e.g. `feat: add aidial_application resource and data sources`); verify `git status` is clean

## Workflow follow-up

- Archive the change after review so the new capabilities are created and the modified ones updated.
