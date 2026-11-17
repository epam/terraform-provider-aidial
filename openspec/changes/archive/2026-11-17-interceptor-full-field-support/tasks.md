# Tasks

## 1. Client

- [x] 1.1 Add `Dependencies`, `DescriptionKeywords`, `InputAttachmentTypes`, `Interceptors`, `MaxInputAttachments`, `Intro`, `Reference`, `OverrideName`, `BaseURL`, `ResponsesEndpoint` and `ResponsesDefaults` to `client.Interceptor` with camelCase tags; verify with client tests in `internal/client` that assert the exact request body and a GET decode of a captured Core 0.50.1 response.

## 2. Resource and data sources

- [x] 2.1 Add the attributes to the interceptor model, schema and plan/state mapping, with Optional+Computed and `UseStateForUnknown` for `dependencies`, `description_keywords` and `interceptors` and plain optional for the rest; verify with provider unit tests for round trip, an empty configured set, no plan churn after a minimal apply and the data source schema validating (`interceptor_resource_test.go`).
- [x] 2.2 Expose the same attributes as computed on `aidial_interceptor` and `aidial_file_interceptor`; verify with data source unit tests that populate them from a captured response (`interceptor_data_source_test.go`).
- [x] 2.3 Update the resource `MarkdownDescription` to say that fields the resource does not model are dropped on apply; verify the text appears in the regenerated docs (task 3.1).

## 3. Integration, docs and release

- [x] 3.1 Extend the acceptance test against the local Core (`TF_ACC`) with all new attributes and a no-diff re-plan; confirm `reference` stays null when unset. Verify with `TF_ACC=1 go test ./internal/provider -run TestAccInterceptor -count=1`. Add an `examples/` snippet and run `make docs`; verify `git diff --exit-code docs/` is clean afterwards.
- [x] 3.2 Add the interceptor entry to the "Unreleased" section of CHANGELOG (shared with the application entry, for 0.3.2); verify `go test ./... -count=1 -race`, `go vet ./...`, `test -z "$(gofmt -l .)"`, `make lint` and `terraform fmt -check -recursive` all pass.

## Workflow follow-up

- Archive this change together with `application-full-field-support` after review.
- Tag `v0.3.2` per CONTRIBUTING "Releasing".
