# Tasks

## 1. Client types

- [x] 1.1 Extend `client.Application` with the simple fields (`input_attachment_types`, `description_keywords`, `dependencies`, `max_input_attachments`, `responses_endpoint`, `responses_defaults`, `intro`, `reference`, `override_name`, `base_url`, `allow_user_external_services`) and add `mcp`, `routes` and `external_services` types with the camelCase and snake_case tags from the design; verify with client unit tests asserting the exact request body and a GET decode of a captured Core 0.50.1 response (extend `internal/client/application_test.go`).
- [x] 1.2 Update the client comment on unmanaged fields and verify `go vet ./...` passes.

## 2. Resource schema and mapping

- [x] 2.1 Add the simple attributes to `applicationBody`, the schema and the plan/state mapping, with `defaultedSet` for `dependencies` and `description_keywords`; verify with provider unit tests for round trip and no plan churn after a minimal apply (extend `application_resource_test.go`).
- [x] 2.2 Add `mcp` as a typed single nested attribute, with `transport` validated like the toolset, `config_delivery` one of `HEADER`/`META`, and Optional+Computed for `config_delivery`, `forward_per_request_key` and `allowed_tools`; verify with unit tests for validators, partial `mcp`, removal and null read-back.
- [x] 2.3 Add `routes` as a typed map with the application's own route body schema (no `id`, `name`, `status`); verify with unit tests for a static-response route, Core-filled defaults, an empty map and removal of an entry.
- [x] 2.4 Add `external_services` with sensitive `client_secret` and `code_verifier` kept from prior state and a minimum length of 8 for `client_secret`; verify with unit tests for retention, change, short secret and import leaving the secrets null.
- [x] 2.5 Update the resource `MarkdownDescription` so it names `function` as unmanaged and dropped on apply; verify the text appears in the regenerated docs (task 4.2).

## 3. Data sources

- [x] 3.1 Expose the new attributes as computed on `aidial_application` and `aidial_file_application`, with the secrets null; verify with data source unit tests (extend `application_data_source_test.go` and the file data source test).

## 4. Integration, docs and release

- [x] 4.1 Add acceptance tests against the local Core (`TF_ACC`, `scripts/core.sh up`) for a full application with `mcp`, `routes` and `external_services`, and confirm no diff on a second plan; also record whether Core returns route upstream `key` and adjust the mapping if it is masked. Verify with `make testacc`.
- [x] 4.2 Add an `examples/` snippet and regenerate `docs/` with `make docs`; verify `git diff --exit-code docs/` is clean afterwards.
- [x] 4.3 Add the CHANGELOG entry for 0.3.2, then verify `go test ./... -count=1 -race`, `go vet ./...`, `test -z "$(gofmt -l .)"`, `make lint` and `terraform fmt -check -recursive` all pass.

## Workflow follow-up

- Archive the change after review and sync `openspec/specs/application-resource`.
- Tag `v0.3.2` per CONTRIBUTING "Releasing".
