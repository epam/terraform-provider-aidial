# Tasks

## 1. Verify against a real Core

- [x] 1.1 Start the ephemeral Core (`scripts/core.sh up`) and probe `/v1/aliases/...` with throwaway aliases: JSON key casing, accepted `bucket` values (`public`, `platform`), minimum accepted payload (`{}`, `presentAs` only, `presentAs` plus `target`), `target` typing and combinations (`id`, `path`, `version`; string against number), whether `presentAs` must match the target kind, which other `Alias` fields are accepted or rejected with 400, values Core fills in on a minimal PUT, what GET returns for `author`/`createdAt`/`updatedAt`/`status`, whether `displayName` and `description` can come back as locale maps, nested `path` with slashes (raw against `%2F`), and whether the ephemeral Core defines a static alias; delete the probes afterwards; record the findings in `design.md` and, where they differ from the specs, update the `alias-resource`, `resource-identity` and `core-client-requests` deltas first

## 2. Client

- [x] 2.1 Add `Alias` and `AliasTarget` (only `path`) types and `GetAlias`, `SaveAlias`, `DeleteAlias`, `GetFileAlias` to `internal/client/alias.go` (generic helpers, URL building as in `interceptor.go`); verify `go build ./...` passes
- [x] 2.2 Add client unit tests in the style of `interceptor_test.go`: headers, no ETag headers, 404 nil, delete 404 success, PUT `{"name"}` response, error with status and body, encoded paths, key casing, omitted fields not sent; verify `go test ./internal/client -count=1 -race` passes

## 3. Resource

- [x] 3.1 Add `internal/provider/alias_resource.go`: schema (attributes in the `alias-resource` spec, `bucket` validated as `public` or `platform`, non-empty `path`, required `present_as` with `OneOf`, required `target_path` with the regex and the `present_as` cross-check, `intro` and `description_keywords`), plan-to-client conversion and client-to-state mapping with null-vs-empty rules; add the bucket validator next to the others in `helpers.go`; verify `go vet ./...` passes
- [x] 3.2 Implement Create, Read, Update, Delete and ImportState (`<bucket>/<path>`) with PUT then GET and 404 removal; register in `internal/provider/provider.go`
- [x] 3.3 Add `alias_resource_test.go` against a fake server: minimal body, full body, bucket other than `public`/`platform` and empty path rejected, missing and unknown `present_as` rejected, missing or malformed `target_path` and `present_as`/target-kind mismatches rejected, immutable identity, create re-read, in-place update, out-of-band deletion, omitted attributes null, configured empty set stays empty, malformed import id; verify `go test ./internal/provider -count=1 -race` passes

## 4. Data sources

- [x] 4.1 Add `alias_data_source.go` (`bucket` and `path` required, computed attributes equal to the resource's, not-found error naming bucket and path) and register it; verify unit tests for found, missing, API error and resource parity pass
- [x] 4.2 Add `file_alias_data_source.go` (`name` required, non-empty, same computed attributes minus identity, not-found error) and register it; verify unit tests for found, missing and empty `name`, and a registration test listing all three new type names, pass
- [x] 4.3 Extend `identity_test.go` (`TestResourceIdentityRequired`, `TestDataSourceIdentityRequired`) to cover the alias resource and data source; verify the tests pass

## 5. Integration and acceptance

- [x] 5.1 Add `TestAcc*` tests in `internal/provider/alias_resource_acc_test.go` (create, update, import, data source, file data source only if the Core ships a static alias) and a manifest `tests/integration/tf_aidial_integration_alias.tf` using `public` and `platform`; verify `make testacc` and `make test-integration` pass, and that the acceptance tests skip without `TF_ACC`

## 6. Docs and final checks

- [x] 6.1 Add `examples/resources/aidial_alias/` (`resource.tf`, `import.sh`, `import-by-string-id.tf`), `examples/data-sources/aidial_alias/data-source.tf` and `examples/data-sources/aidial_file_alias/data-source.tf`; verify `terraform fmt -check -recursive` passes
- [x] 6.2 Run `make docs`; verify `git diff --exit-code docs` is clean afterwards and the new pages exist under `docs/resources` and `docs/data-sources`
- [x] 6.3 Update the README (resource and data source tables, an Alias section noting the unexposed fields and the plain-string display values, import section, spec pointers), add a `CHANGELOG.md` entry under `Unreleased`, and update the `AGENTS.md` architecture and identity bullets (aliases accept `public` and `platform`); verify `make lint-md` passes
- [x] 6.4 Run `go test ./... -count=1 -race`, `test -z "$(gofmt -l .)"`, `go vet ./...`, `make lint` and `openspec validate add-aliases-resource --strict`; verify all pass
- [ ] 6.5 Create branch `feat/add-aliases-resource` off `main` and commit with semantic messages (for example `feat: add aidial_alias resource and data sources`); verify `git status` is clean

## Workflow follow-up

- Archive the change after review so the two new capabilities are created and the three modified ones are updated.
