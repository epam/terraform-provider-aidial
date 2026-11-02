# Tasks

## 1. Client and schemas

- [x] 1.1 In `internal/client/client.go` change the `Name` JSON tags on `Model`, `Role` and `Route` to `name,omitempty`; verify `go test ./internal/client -count=1 -race` passes and a request test shows no `name` key when it is empty
- [x] 1.2 In `model_resource.go`, `role_resource.go` and `route_resource.go` make `name` `Optional` and `Computed` with `stringplanmodifier.UseStateForUnknown()` and update its description; verify `go build ./...` passes
- [x] 1.3 In `model_data_source.go` make `name` `Optional` and `Computed` and drop any use of it for the lookup; verify `go build ./...` passes

## 2. Tests

- [x] 2.1 Add unit tests for model, role and route: `name` omitted sets `name` equal to `api_name` after create, a configured `name` is kept when the API returns another value, and a known `name` is not rewritten on read; verify `go test ./internal/provider -count=1 -race` passes
- [x] 2.2 Add a test that state written with `name` as a required attribute decodes unchanged against the new schema (no schema version bump); verify it passes
- [x] 2.3 Add acceptance tests with `name` omitted for model, role and route that apply, re-plan with no changes, import, and then add a `name` as an in-place update; verify `make testacc` passes against a local Core

## 3. Fixtures, examples and docs

- [x] 3.1 Drop `name` from `examples/` and README snippets where it only repeats `path`, and keep it in at least one integration manifest per resource; verify `terraform fmt -check -recursive` and `make test-integration` pass
- [x] 3.2 Run `make docs`; verify `name` is listed as optional and `git diff --exit-code docs` is clean afterwards
- [x] 3.3 Update the `name` mentions in `AGENTS.md` if any; verify `markdownlint-cli2` passes

## 4. Specs

- [x] 4.1 Apply the edits from this change's deltas to `openspec/{model,role,route}-schema/spec.md`, `openspec/{model,role,route}-state-consistency/spec.md` and `openspec/role-read/spec.md`, and replace the "`name` is `Required`" wording in the Implementation Notes of `openspec/role-import/spec.md` and the other schema specs; verify `grep -rn "Required.*name\|required, .*name" openspec --include=spec.md` shows no stale claim
- [x] 4.2 Apply the `model-data-source` delta to the spec (in `add-model-data-source` if it is still unarchived); verify `openspec validate --specs` passes
- [x] 4.3 Run `openspec validate make-name-optional --strict`; verify it reports no errors

## 5. Integration checks

- [x] 5.1 Run `go test ./... -count=1 -race`, `test -z "$(gofmt -l .)"`, `go vet ./...` and `make lint`; verify all pass

## Workflow follow-up

- Apply and archive `require-bucket-path-identity` (and `add-model-data-source`) first.
- Archive this change after review with `--skip-specs`, because tasks 4.1 and 4.2 apply the deltas by hand.
