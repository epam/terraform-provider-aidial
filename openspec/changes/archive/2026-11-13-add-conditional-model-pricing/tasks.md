# Tasks

## 1. Client

- [x] 1.1 Add `PricingRate` and `PricingCondition` with string-or-object JSON handling in `internal/client/pricing.go`; verify `go test ./internal/client -count=1 -race` passes

## 2. Provider

- [x] 2.1 Add the rate schema, validators and plan/state mapping in `internal/provider/model_pricing.go` and use it in the resource and both data sources; verify `go build ./...` passes
- [x] 2.2 Add unit tests and `TestAccModel_conditionalPricing`; verify `go test ./internal/provider -count=1 -race` and the acceptance test against a local Core pass

## 3. Configs and docs

- [x] 3.1 Migrate `stub/models`, `tests/integration` and `examples`; verify `terraform validate` passes in `tests/integration`, `tests/integration-jwt` and `stub/models`
- [x] 3.2 Regenerate `docs/`, update README, CHANGELOG and `openspec/model-schema/spec.md`
