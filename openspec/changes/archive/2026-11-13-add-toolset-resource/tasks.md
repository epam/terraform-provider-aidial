# Tasks

## 1. Share model features and interfaces code

- [x] 1.1 Extract the `features` and `interfaces` schema builders, plan-to-client conversion and client-to-state mapping (including null normalization) from `internal/provider/model_resource.go` into shared helpers; verify `go test ./internal/provider -count=1 -race` passes with no test edits, and `git diff` shows no change to the model schema
- [x] 1.2 Confirm there is no shared identity helper to parameterize (each schema declares `OneOf(platformBucket)` inline), so the toolset schemas simply declare their own non-empty `bucket` validator; verify `TestResourceIdentityRequired` and `TestDataSourceIdentityRequired` still pass unchanged

## 2. Client

- [x] 2.1 Add `ToolSet`, `ResourceAuthSettings` types and `GetToolset`, `SaveToolset`, `DeleteToolset`, `GetFileToolset` to `internal/client/client.go` (URLs `/v1/toolsets/{bucket}/{path}` and `/v1/admin/config/file/toolsets/{name}` via `escapeSegment`, reusing the generic helpers, reusing `ModelFeatures` and the interface type); verify `go build ./...` passes
- [x] 2.2 Add client unit tests in the style of `crud_test.go` and `file_model_test.go`: headers, no ETag headers, 404 nil, delete 404 success, PUT `{"name"}` response, error with status and body, escaped paths `/v1/toolsets/public/team%2Ftool` and `/v1/admin/config/file/toolsets/a%2Fb`, omitted fields not sent, `client_secret` redacted from logs (added to `sensitiveJSONFields`); verify `go test ./internal/client -count=1 -race` passes

## 3. Resource

- [x] 3.1 Add `internal/provider/toolset_resource.go`: schema (all attributes in the `toolset-resource` spec, non-empty `bucket` validator, `transport` and `authentication_type` enum validators, `client_secret` sensitive), plan-to-client conversion, client-to-state mapping with null-vs-empty rules, free-form fields via `model_defaults.go`; verify `go vet ./...` passes
- [x] 3.2 Implement Create, Read, Update, Delete and ImportState (`<bucket>/<path>`) with PUT then GET, 404 removal, `client_secret` kept from prior state; register in `internal/provider/provider.go`
- [x] 3.3 Add `toolset_resource_test.go` against a fake server: minimal body, full body, `public` bucket accepted, empty bucket and path rejected, immutable identity, create re-read, in-place update, out-of-band deletion, secret kept across refresh and null after import, ignored runtime status fields, omitted collections null, configured empty set stays empty, free-form key order no-op, malformed import id, `features` and `interfaces` parity with the model; verify `go test ./internal/provider -count=1 -race` passes
- [x] 3.4 Extend the sensitivity test (`sensitive_test.go`) to assert `auth_settings.client_secret` is sensitive on the resource and absent from both data sources; verify the test passes

## 4. Data sources

- [x] 4.1 Add `toolset_data_source.go` (`bucket` and `path` required, computed attributes equal to the resource's, no `client_secret`, not-found error naming bucket and path) and register it; verify unit tests for found, missing, API error and resource parity pass
- [x] 4.2 Add `file_toolset_data_source.go` (`name` required, non-empty, same computed attributes minus identity and secrets, not-found error) and register it; verify unit tests for found, missing and empty `name`, and a registration test listing all three new type names, pass

## 5. Verify against a real Core

- [x] 5.1 Start the ephemeral Core (`scripts/core.sh up`); find Core's minimum accepted payload (an almost-empty toolset and a typical `endpoint` + `transport` toolset), a nested `path`, and whether a `public` bucket works for PUT, GET and DELETE; record the findings in `design.md`, and adjust required attributes or path encoding only if Core demands it
- [x] 5.2 Add `TestAcc*` tests in `internal/provider/toolset_resource_acc_test.go` (create, update, import, data source, file data source if the Core ships a file toolset) and a manifest `tests/integration/tf_aidial_integration_toolset.tf` using `platform`, plus a `public` case only if 5.1 shows it works; verify `make testacc` and `make test-integration` pass, and that the acceptance tests skip without `TF_ACC`

## 6. Docs and final checks

- [x] 6.1 Add `examples/resources/aidial_toolset/` (`resource.tf`, `import.sh`, `import-by-string-id.tf`), `examples/data-sources/aidial_toolset/data-source.tf` and `examples/data-sources/aidial_file_toolset/data-source.tf`; verify `terraform fmt -check -recursive` passes
- [x] 6.2 Run `make docs`; verify `git diff --exit-code docs` is clean afterwards and the new pages exist under `docs/resources` and `docs/data-sources`
- [x] 6.3 Update the README (resource and data source tables, a Toolset section with the bucket note, import section, spec pointers) and add a `CHANGELOG.md` entry under `Unreleased`; update the `AGENTS.md` architecture and identity bullets, which state that all resources require `platform`; verify `make lint-md` passes
- [x] 6.4 Run `go test ./... -count=1 -race`, `test -z "$(gofmt -l .)"`, `go vet ./...`, `make lint` and `openspec validate add-toolset-resource --strict`; verify all pass
- [ ] 6.5 Commit with semantic messages (for example `refactor: share model features and interfaces helpers`, then `feat: add aidial_toolset resource and data sources`); verify `git status` is clean

## Workflow follow-up

- Archive the change after review so the two new capabilities are created and the four modified ones are updated.
- Candidate follow-up changes: bucket validation review for the other resources, and the `locale map` form of `display_name` and `intro`.
