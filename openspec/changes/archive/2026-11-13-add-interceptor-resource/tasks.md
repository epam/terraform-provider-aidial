# Tasks

## 1. Verify against a real Core

- [x] 1.1 Start the ephemeral Core (`scripts/core.sh up`) and probe `/v1/interceptors/...` with throwaway interceptors: JSON key casing (camelCase against snake_case), minimum accepted payload (`{}` and `endpoint` only), nested `path` with slashes (raw against `%2F`), `bucket` values (`platform`, `public`), values Core fills in on a minimal PUT, whether `GET` returns `author`/`createdAt`/`updatedAt`, and whether the ephemeral Core defines a static interceptor; delete the probes afterwards; record the findings in `design.md` and, where they differ from the specs, update `interceptor-resource`, `resource-identity` and `core-client-requests` deltas first

## 2. Client

- [x] 2.1 Add `Interceptor` type and `GetInterceptor`, `SaveInterceptor`, `DeleteInterceptor`, `GetFileInterceptor` to `internal/client/interceptor.go` (reusing `ModelFeatures`, `ModelInterface` and the generic helpers, URL building as in `toolset.go`); verify `go build ./...` passes
- [x] 2.2 Add client unit tests in the style of `toolset_test.go`: headers, no ETag headers, 404 nil, delete 404 success, PUT `{"name"}` response, error with status and body, encoded paths, key casing, omitted fields not sent; verify `go test ./internal/client -count=1 -race` passes

## 3. Resource

- [x] 3.1 Add `internal/provider/interceptor_resource.go`: schema (attributes in the `interceptor-resource` spec, `bucket` validated with `platformBucket`, non-empty `path`, required `endpoint`), plan-to-client conversion and client-to-state mapping with null-vs-empty rules, `features` and `interfaces` through the shared helpers, `defaults` and `catalog_properties` through `model_defaults.go`, Core-defaulted attributes optional and computed per task 1.1; verify `go vet ./...` passes
- [x] 3.2 Implement Create, Read, Update, Delete and ImportState (`<bucket>/<path>`) with PUT then GET and 404 removal; register in `internal/provider/provider.go`
- [x] 3.3 Add `interceptor_resource_test.go` against a fake server: minimal body, full body, non-`platform` bucket and empty path rejected, missing `endpoint` rejected, immutable identity, create re-read, in-place update, out-of-band deletion, omitted collections null, configured empty set stays empty, free-form key order no-op, malformed import id, `features` and `interfaces` parity with the model; verify `go test ./internal/provider -count=1 -race` passes

## 4. Data sources

- [x] 4.1 Add `interceptor_data_source.go` (`bucket` and `path` required, computed attributes equal to the resource's, not-found error naming bucket and path) and register it; verify unit tests for found, missing, API error and resource parity pass
- [x] 4.2 Add `file_interceptor_data_source.go` (`name` required, non-empty, same computed attributes minus identity, not-found error) and register it; verify unit tests for found, missing and empty `name`, and a registration test listing all three new type names, pass
- [x] 4.3 Extend `identity_test.go` (`TestResourceIdentityRequired`, `TestDataSourceIdentityRequired`) to cover the interceptor resource and data source; verify the tests pass

## 5. Integration and acceptance

- [x] 5.1 Add `TestAcc*` tests in `internal/provider/interceptor_resource_acc_test.go` (create, update, import, data source, file data source only if the Core ships a static interceptor) and a manifest `tests/integration/tf_aidial_integration_interceptor.tf` using `platform`; verify `make testacc` and `make test-integration` pass, and that the acceptance tests skip without `TF_ACC`

## 6. Docs and final checks

- [x] 6.1 Add `examples/resources/aidial_interceptor/` (`resource.tf`, `import.sh`, `import-by-string-id.tf`), `examples/data-sources/aidial_interceptor/data-source.tf` and `examples/data-sources/aidial_file_interceptor/data-source.tf`; verify `terraform fmt -check -recursive` passes
- [x] 6.2 Run `make docs`; verify `git diff --exit-code docs` is clean afterwards and the new pages exist under `docs/resources` and `docs/data-sources`
- [x] 6.3 Update the README (resource and data source tables, an Interceptor section noting the unexposed fields and that models reference interceptors by name through their `interceptors` list, import section, spec pointers), add a `CHANGELOG.md` entry under `Unreleased`, and update the `AGENTS.md` architecture and identity bullets; verify `make lint-md` passes
- [x] 6.4 Run `go test ./... -count=1 -race`, `test -z "$(gofmt -l .)"`, `go vet ./...`, `make lint` and `openspec validate add-interceptor-resource --strict`; verify all pass
- [x] 6.5 Create branch `feat/add-interceptor-resource` off `main` (the toolset change is already merged there) and commit with semantic messages (for example `feat: add aidial_interceptor resource and data sources`); verify `git status` is clean

## Workflow follow-up

- Archive the change after review so the two new capabilities are created and the three modified ones are updated.
