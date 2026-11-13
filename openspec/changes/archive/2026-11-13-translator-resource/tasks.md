# Tasks

## 1. Client

- [x] 1.1 Add `Translator` type (`base_url`, `in`, `out` with camelCase JSON keys) and `GetTranslator`, `SaveTranslator`, `DeleteTranslator`, `GetFileTranslator` to `internal/client/translator.go` (generic helpers, URL building as in `interceptor.go`); verify `go build ./...` passes
- [x] 1.2 Add client unit tests in the style of `interceptor_test.go`: headers, no ETag headers, 404 nil, delete 404 success, PUT `{"name"}` response, error with status and body, encoded paths (`team%2Fopenai`, file name `a%2Fb`), key casing, extra response fields ignored; verify `go test ./internal/client -count=1 -race` passes

## 2. Resource

- [x] 2.1 Add `internal/provider/translator_resource.go`: schema (`bucket` validated with `platformBucket`, non-empty `path`, required `base_url`, `in` and `out` with `OneOf(modelInterfaceTypes...)`), plan-to-client conversion and client-to-state mapping; verify `go vet ./...` passes
- [x] 2.2 Implement Create, Read, Update, Delete and ImportState (`<bucket>/<path>`) with PUT then GET and 404 removal; register in `internal/provider/provider.go`
- [x] 2.3 Add `translator_resource_test.go` against a fake server: full body, non-`platform` bucket and empty path rejected, each missing attribute rejected, unknown `in`/`out` rejected, config using the `in` attribute applies, immutable identity, create re-read, in-place update, out-of-band deletion, out-of-band `base_url` change shows as drift, malformed import id; verify `go test ./internal/provider -count=1 -race` passes

## 3. Data sources

- [x] 3.1 Add `translator_data_source.go` (`bucket` and `path` required, computed `id`, `base_url`, `in`, `out`, not-found error naming bucket and path) and register it; verify unit tests for found, missing, API error and resource parity pass
- [x] 3.2 Add `file_translator_data_source.go` (`name` required, non-empty, computed `base_url`, `in`, `out`, not-found error) and register it; verify unit tests for found, missing and empty `name`, and a registration test listing all three new type names, pass
- [x] 3.3 Extend `identity_test.go` (`TestResourceIdentityRequired`, `TestDataSourceIdentityRequired`) to cover the translator resource and data source; verify the tests pass

## 4. Integration and acceptance

- [x] 4.1 Add `TestAcc*` tests in `internal/provider/translator_resource_acc_test.go` (create, update, import, data source; no file data source test, because the ephemeral Core defines no static translator) and a manifest `tests/integration/tf_aidial_integration_translator.tf` using `platform`; verify `make testacc` and `make test-integration` pass, and that the acceptance tests skip without `TF_ACC`

## 5. Docs and final checks

- [x] 5.1 Add `examples/resources/aidial_translator/` (`resource.tf`, `import.sh`, `import-by-string-id.tf`), `examples/data-sources/aidial_translator/data-source.tf` and `examples/data-sources/aidial_file_translator/data-source.tf`; verify `terraform fmt -check -recursive` passes
- [x] 5.2 Run `make docs`; verify `git diff --exit-code docs` is clean afterwards and the new pages exist under `docs/resources` and `docs/data-sources`
- [x] 5.3 Update the README (resource and data source tables, a Translator section noting that models do not reference translators through the provider yet and that unexposed fields are replaced on save, import section, endpoint list, spec pointers), add a `CHANGELOG.md` entry under `Unreleased`, and update the `AGENTS.md` architecture and identity bullets; verify `make lint-md` passes
- [x] 5.4 Run `go test ./... -count=1 -race`, `test -z "$(gofmt -l .)"`, `go vet ./...`, `make lint` and `openspec validate translator-resource --strict`; verify all pass
- [x] 5.5 Create branch `feat/add-translator-resource` off `main`; do not commit or push until the user asks, then use `feat: add aidial_translator resource and data sources`

## Workflow follow-up

- Archive the change after review so the two new capabilities are created and the three modified ones are updated.
