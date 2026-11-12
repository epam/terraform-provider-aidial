# Design

## Context

- `provider.go` has `base_url` and `api_key`, both Required. `Configure` skips when either is unknown, then calls `client.NewClient(baseURL, apiKey)`.
- `client.go` stores one `apiKey` and sets `Api-Key` in `newRequest`. Request logging prints `api_key_present`, not the key. No file logs headers.
- Core's OpenAPI contract declares `BearerAuth` (`Authorization: Bearer <JWT>`) next to `ApiKeyAuth`.
- `tests/core/docker-compose.yml` already has an `identityProviders.ci` placeholder with a dummy `jwksUrl`, `rolePath: roles` and an admin rule `EQUAL roles ["admin"]`. The API key `ci-check-key` carries role `admin`.
- The `integration` job runs `scripts/test-integration.sh`, which starts Core, applies `tests/integration` with a key, runs the drift check and destroys.
- GitHub Actions tokens carry no `roles` claim and cannot get custom claims. They do carry `repository`, `sub`, `ref`, `iss` and the requested `aud`.

See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- One credential resolution rule that is easy to explain and test.
- Token auth proven end to end in CI against a real Core, without exposing the token to jobs that don't need it.
- Existing `api_key` configs and the local API-key workflow unchanged.

**Non-Goals:**
- Minting, exchanging or refreshing tokens, reading a token file, or any OIDC login flow.
- Verifying a JWT in the provider (signature, `exp`, `aud`).
- Acceptance-test (`TestAcc*`) coverage of JWT auth; the CI JWT job covers it.
- GitLab-specific CI automation: GitLab appears only as a documentation example.

## Decisions

**Resolution lives in a pure function.** `resolveCredential(cfgAPIKey, cfgIDToken, envAPIKey, envIDToken) (Credential, []warning, error)` in `internal/provider`. `Configure` reads config and `os.Getenv`, calls it and appends diagnostics. This makes the whole precedence matrix a table test with no Terraform harness. Alternative: validate with `ConfigValidators` (`ExactlyOneOf`). Rejected: it cannot see the environment and would reject the legitimate "both env vars set" and "config overrides env" cases.

**Config beats env, and any ignored env var warns.** This is the rule you chose. Warnings name variables only, never values, and fire only for non-empty variables that lost. The only errors are both-in-config and nothing-at-all. Alternative considered: silent token-over-key among env vars. Dropped in favor of one uniform rule.

**`api_key` becomes Optional.** No `Required` schema validator can express "one of these or its env var", so the check lives in `Configure`. Trade-off: a missing credential now fails at configure time rather than at validate. The unknown-value early return stays as is, extended to `id_token`.

**Shape check in the provider, `Credentials` in the client.** The client gets a small `Credentials` struct (kind plus secret) and picks the header; it keeps its existing "credential required" check. The provider trims whitespace, then checks for three non-empty `.`-separated segments with no further parsing. Alternative: decode the JWT and check `exp` locally. Rejected: it duplicates Core, adds clock-skew failure modes, and invites trust in unverified claims. A client with an invalid `Credentials` value (both or neither) returns an error from `NewClient`.

**Logging.** Replace `api_key_present` with `credential=<api_key|id_token> present=<bool>`. Header values are never logged. A client test captures `log` output and asserts the secret is absent for both kinds. No new redaction code is needed because headers aren't logged; the requirement is enforced by the test.

**Core identity provider (shared compose).** Replace `ci` with `github`: `jwksUrl: https://token.actions.githubusercontent.com/.well-known/jwks`, `issuerPattern: ^https://token\.actions\.githubusercontent\.com$`, `rolePath: repository`. The admin rule's target becomes `["epam/terraform-provider-aidial"]`, plus `admin` stays reachable through the API key's role. Core fetches the JWKS lazily, so offline local runs work. If Core's identity provider config supports an audience setting, set it to a fixed value (`aidial-ci`) and request the token with that audience; verify against Core's source or docs at implementation time. If Core has no such setting, record that in the spec and docs and drop the audience requirement from the scenario text. Alternative: match `sub`, which is stricter but breaks for PR and tag runs.

**Separate JWT job, not an extra step.** `id-token: write` is a job-level permission, so putting the JWT run inside `integration` would expose it to the long full-lifecycle job. A new job `integration-jwt` with the same `needs`, its own Core, `permissions: { contents: read, id-token: write }`, runs a new script `scripts/test-integration-jwt.sh` and `tests/integration-jwt/` (separate manifest and state, one `aidial_role` plus a data source read). It requests the token by `curl` to `ACTIONS_ID_TOKEN_REQUEST_URL` with `&audience=`, masks it via `::add-mask::`, and exports `AIDIAL_ID_TOKEN`. The script unsets `AIDIAL_API_KEY` and uses `-var-file`-style explicit `base_url`, so the ambient environment cannot redirect it. Fork gating: `if: github.event_name != 'pull_request' || github.event.pull_request.head.repo.full_name == github.repository`. Alternative: extend `test-integration.sh` with a mode flag. Rejected: it would share a job, and therefore the permission.

**Documentation.** README "Authentication" section holds the precedence table, GitHub and GitLab snippets and Core settings. `provider.go` descriptions feed the generated `docs/index.md`. `examples/` gets HCL provider blocks only. `make docs` is rerun because CI fails on drift.

## Risks / Trade-offs

- [Short-lived token expires mid-apply] → Documented; recommend a job token lifetime longer than the apply. No refresh.
- [`github` provider mapping `repository` as the role gives admin to anyone who can run workflows in this repo] → The job runs only for non-fork events and no token reaches fork code; the issuer is pinned. Document that Core must map a claim an attacker cannot choose.
- [Tokens from a different repo with the same name in another org] → `repository` includes the owner (`epam/...`), and the audience check, if supported, narrows it further.
- [`api_key` no longer Required changes plan-time errors] → Clear configure-time error naming all four inputs; call it out in the CHANGELOG.
- [Warning noise for developers with exported variables] → Intentional, per the chosen rule; the message names the variable so it can be unset.
- [Core `latest` may change identity provider config keys] → The JWT job runs against `latest` like the existing job, so a break shows up in CI; pin `CORE_VERSION` if it becomes flaky.
- [GitHub JWKS outage fails the JWT job] → The API-key job is independent and still gates correctness; the JWT job fails visibly rather than being skipped.

## Migration Plan

Ship as a minor release. No state or schema migration: existing configs with `api_key` behave identically. Rollback is a normal provider downgrade; configs that adopted `id_token` must switch back to `api_key` first.
