# Proposal

## Why

The provider can only authenticate to DIAL Core with a long-lived `Api-Key`. DIAL Core also accepts a JWT in the `Authorization: Bearer` header, validated against configured OIDC identity providers (GitLab, GitHub and others). CI/CD systems mint short-lived ID tokens per job, so pipelines can manage DIAL Core without storing a static admin key as a secret. The Core OpenAPI contract already allows this (`BearerAuth`); only the provider lacks it.

## What Changes

- Add an optional, sensitive `id_token` provider attribute and make `api_key` optional (it is Required today). `base_url` stays Required.
- Add environment fallbacks `AIDIAL_ID_TOKEN` and `AIDIAL_API_KEY`.
- Resolve the credential with one rule: configuration always beats the environment, and any ignored environment variable produces a warning (never its value). `api_key` and `id_token` both set in configuration is an error. With neither in configuration, `AIDIAL_ID_TOKEN` beats `AIDIAL_API_KEY`. No credential at all is an error naming both options.
- Validate only the shape of the token in the provider (non-empty, three dot-separated segments). Signature, `exp` and `aud` are checked by Core.
- The client takes a `Credentials` value and sends either `Api-Key` or `Authorization: Bearer <token>`. Request logging never prints either credential.
- No token fetching, refresh, file input or OIDC flow: the pipeline supplies a ready-made JWT. Short token lifetime is a documented limitation.
- Shared ephemeral Core (`tests/core/docker-compose.yml`): replace the placeholder `ci` identity provider with GitHub Actions OIDC (pinned issuer and JWKS URL, `rolePath: repository`, admin rule matching this repository, audience check if Core supports one). The fixed CI API key stays for local runs and the existing integration run.
- CI: add a small JWT-only integration run (dedicated manifest and state) in a job that alone gets `id-token: write`, masks the token, and is skipped for fork pull requests.
- Documentation: README "Authentication" section and generated `docs/` with the precedence table, GitLab CI and GitHub Actions snippets, Core `identityProviders` setup and the token lifetime limitation. CHANGELOG entry (minor version).

Not **BREAKING**: every existing config that sets `api_key` keeps working. Configs that set neither now fail at configure time instead of at schema validation.

## Capabilities

### New Capabilities

- `provider-authentication`: how the provider obtains and sends credentials to DIAL Core: `api_key` / `id_token` attributes, environment fallbacks, precedence and warnings, token shape check, `Api-Key` vs `Authorization: Bearer` headers, and no credential leakage in logs.

### Modified Capabilities

- `sensitive-attributes`: `id_token` (provider) joins `api_key` as a sensitive attribute.
- `ci-integration-core`: the integration job gains a JWT-only run with job-scoped `id-token: write`, skipped for fork pull requests. "Credentials are job-local" is extended to the token.
- `local-ephemeral-core`: the shared Core definition configures a GitHub Actions OIDC identity provider in addition to the fixed throwaway API key.

## Impact

- Code: `internal/provider/provider.go` (schema, resolution, warnings, shape check), `internal/client/client.go` (`Credentials`, header selection, logging), and client tests that call `NewClient`.
- Tests: new unit tests for the resolution matrix, shape check, header selection and log redaction.
- CI and test infrastructure: `tests/core/docker-compose.yml`, `.github/workflows/ci.yml`, `scripts/test-integration.sh`, a new `tests/integration`-style JWT manifest.
- Docs: `README.md`, generated `docs/index.md`, `examples/` (HCL provider blocks only), `CHANGELOG.md`.
- Dependencies: none added.
- Security: the new CI job holds `id-token: write`; the token is masked and the job never runs for forks.
