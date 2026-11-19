# Tasks

## 1. Schema and mapping

- [x] 1.1 In `internal/provider/model_resource.go`, make `created_at` and `updated_at` `Optional` (not `Computed`) with `int64validator.AtLeast(0)`, drop the `UseStateForUnknown` modifier from `created_at`, and update both descriptions to say that omitting the value clears it in Core. Verify: `go build ./...` succeeds.
- [x] 1.2 In the request builder next to `EmbeddingDimensions` (~line 468), set `CreatedAt: int64Pointer(plan.CreatedAt)` and `UpdatedAt: int64Pointer(plan.UpdatedAt)`; keep the read mapping. Verify: the unit test in 2.1 passes.

## 2. Tests

- [x] 2.1 In `internal/provider/model_resource_test.go`, add tests that the request body contains `createdAt`/`updatedAt` when configured and omits both when null, and that a negative value fails validation. Verify: `go test ./internal/provider -run Model -count=1 -race` passes.
- [x] 2.2 Add `tests/integration/tf_aidial_integration_model_timestamps.tf` that sets both values (including `1630454400000`) and a second model that omits them; apply and confirm a second plan is empty. Verify: `make test-integration` against the local Core passes, and `terraform fmt -check -recursive` is clean.
- [x] 2.3 Add `TestAccModel_timestamps` in `internal/provider/model_resource_acc_test.go` to cover set, change and remove of `updated_at`. Verify: `TF_ACC=1 go test ./internal/provider -run TestAccModel -count=1` passes against the local Core.

## 3. Specs, docs and changelog

- [x] 3.1 Edit `openspec/model-schema/spec.md` (the `created_at`, `updated_at` bullet and the Implementation Notes mention) and, if it lists computed attributes, `openspec/model-update/spec.md` to match the new behavior. Verify: `grep -n "created_at" openspec/model-*/spec.md` shows no "computed" wording for the resource.
- [x] 3.2 Add a `### Changed` entry under `## Unreleased` in `CHANGELOG.md` describing the writable attributes and that an externally set value is cleared on the next apply unless configured. Verify: `make lint-md` passes.
- [x] 3.3 Run `make docs`. Verify: `docs/resources/model.md` shows both attributes under Optional, and `git diff --stat docs` touches only that file.

## 4. Final checks

- [x] 4.1 Run `gofmt -l .`, `go vet ./...`, `make lint` and `go test ./... -count=1 -race`. Verify: all clean.
- [x] 4.2 Create a `feat:` commit (no push, no PR). Verify: `git log -1` shows `feat: make aidial_model created_at and updated_at writable`.
