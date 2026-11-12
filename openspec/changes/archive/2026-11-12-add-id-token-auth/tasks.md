# Tasks

## 1. Client credentials

- [x] 1.1 Replace `NewClient(baseURL, apiKey)` with a `Credentials` argument (kind plus secret) in `internal/client/client.go`; `newRequest` sets `Api-Key` or `Authorization: Bearer`, never both; `NewClient` rejects an empty or ambiguous credential. Verify with `go build ./...`.
- [x] 1.2 Change request logging to `credential=<kind> present=<bool>`. Verify with a unit test that captures `log` output for both kinds and asserts neither the key nor the token appears.
- [x] 1.3 Update every existing `NewClient` call in `internal/client/*_test.go` and `internal/provider/*_test.go`, and add header-selection tests (API key sends `Api-Key` only; token sends `Authorization: Bearer` only). Verify with `go test ./internal/client -count=1 -race`.

## 2. Provider schema and resolution

- [x] 2.1 Add `id_token` (Optional, Sensitive) and make `api_key` Optional in `internal/provider/provider.go`, with descriptions naming `AIDIAL_ID_TOKEN` and `AIDIAL_API_KEY` and the precedence rule. Verify with a schema test that checks both are sensitive and `base_url` is still required.
- [x] 2.2 Implement a pure `resolveCredential` function (config over env, token over key among env vars, empty counts as unset, errors for both-in-config and none, warnings naming ignored env vars, no values in messages). Verify with a table test covering every row of the precedence matrix in the `provider-authentication` spec.
- [x] 2.3 Implement the token shape check (trim whitespace, three non-empty segments, error without the token). Verify with unit tests for malformed, empty, whitespace-padded and valid tokens.
- [x] 2.4 Wire `Configure` to read config and `os.Getenv`, keep the unknown-value early return for both credentials, emit the warnings and errors, and build `client.Credentials`. Verify with a provider-level test using `t.Setenv` for the combinations in the spec and `go test ./internal/provider -count=1 -race`.
- [x] 2.5 Add an HCL provider block with `id_token` to `examples/`, run `make docs`, and verify `docs/index.md` lists both attributes and `git diff --exit-code docs/` is clean afterwards.

## 3. Ephemeral Core and CI

- [x] 3.1 Look up whether Core's identity provider config supports an audience setting, and GitHub's JWKS URL and issuer. If audience is unsupported, update the `ci-integration-core` and `local-ephemeral-core` delta specs and `design.md` to drop it. Verify the specs still pass `openspec validate add-id-token-auth --strict`.
- [x] 3.2 Replace the placeholder `ci` provider in `tests/core/docker-compose.yml` with the GitHub Actions one (pinned `jwksUrl` and `issuerPattern`, `rolePath: repository`, admin rule for `epam/terraform-provider-aidial`, audience if supported), keeping the fixed API key. Verify that `scripts/core.sh up` still starts a healthy Core and the existing `make test-integration` passes with the API key.
- [x] 3.3 Add `tests/integration-jwt/` (provider with `id_token` from a variable, one `aidial_role`, one data source read) and `scripts/test-integration-jwt.sh` (start Core, take the token from `AIDIAL_ID_TOKEN`, unset `AIDIAL_API_KEY`, pass the Core URL explicitly, validate, apply, destroy, clean up). Verify with `terraform fmt -check -recursive` and by running the script locally against a Core with a locally signed test token if feasible, otherwise by `terraform validate` only.
- [x] 3.4 Add the `integration-jwt` job to `.github/workflows/ci.yml`: same `needs`, `permissions: contents: read, id-token: write`, fork-PR `if`, pinned action SHAs, `persist-credentials: false`, a step that fetches the token with `ACTIONS_ID_TOKEN_REQUEST_URL` and the audience, masks it with `::add-mask::`, and runs the script. Verify with zizmor and `actionlint` if available, then on the pull request that the job runs, passes, and is skipped on a fork.
- [x] 3.5 Update `.github/workflows/ci.yml` `validate` loop to include `tests/integration-jwt`. Verify that the validate job passes.

## 4. Documentation and release notes

- [x] 4.1 Add an "Authentication" section to `README.md`: the two credentials, the precedence table, the ignored-variable warning, GitLab CI (`id_tokens:`) and GitHub Actions (`permissions: id-token: write`, token fetch and mask) snippets, the Core `identityProviders` and admin-rule setup including the `repository` mapping, and the no-refresh limitation. Verify with `markdownlint-cli2 README.md` and by checking every snippet against the specs.
- [x] 4.2 Add a CHANGELOG entry (minor version) covering `id_token`, optional `api_key`, the environment variables, the precedence rule, and the configure-time "no credential" error. Verify with markdownlint.

## 5. Integration checks

- [x] 5.1 Run `go test ./... -count=1 -race`, `go vet ./...`, `test -z "$(gofmt -l .)"`, `make lint`, `terraform fmt -check -recursive` and `openspec validate add-id-token-auth --strict`; all must pass.
- [x] 5.2 Confirm on the pull request that both integration jobs pass and the workflow log shows the ID token only masked.

## Workflow follow-up

- Archive the change after review so the deltas merge into `openspec/specs/`.
