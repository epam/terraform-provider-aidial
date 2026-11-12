# provider-authentication Specification

## Purpose

Defines how the provider obtains a credential, either a DIAL API key or a ready-made OIDC ID token (JWT), and sends it to DIAL Core, so pipelines can authenticate without a static admin key.

## Requirements

### Requirement: Credential attributes
The provider schema SHALL expose `base_url` (required), `api_key` (optional, sensitive) and `id_token` (optional, sensitive). The provider SHALL read `AIDIAL_API_KEY` and `AIDIAL_ID_TOKEN` from the environment as fallbacks for `api_key` and `id_token`. An empty string counts as not set, in configuration and in the environment.

#### Scenario: Provider with only an API key
- **WHEN** a configuration sets `base_url` and `api_key` and nothing else
- **THEN** the provider configures successfully and authenticates with that key

#### Scenario: Provider with only an ID token
- **WHEN** a configuration sets `base_url` and `id_token`
- **THEN** the provider configures successfully and authenticates with that token

#### Scenario: Credential from the environment
- **WHEN** a configuration sets only `base_url` and `AIDIAL_ID_TOKEN` is set in the environment
- **THEN** the provider authenticates with that token

#### Scenario: Empty value
- **WHEN** `id_token` is configured as `""` and `AIDIAL_API_KEY` is set
- **THEN** `id_token` counts as not set and the key from the environment is used

### Requirement: Credential precedence
The provider SHALL select exactly one credential. A credential set in configuration SHALL win over any environment variable. With no credential in configuration, `AIDIAL_ID_TOKEN` SHALL win over `AIDIAL_API_KEY`. A configuration that sets both `api_key` and `id_token` SHALL be rejected, and so SHALL the absence of any credential.

#### Scenario: Both set in configuration
- **WHEN** a configuration sets both `api_key` and `id_token`
- **THEN** configuration fails with an error naming both attributes

#### Scenario: Config API key over environment
- **WHEN** `api_key` is configured and `AIDIAL_API_KEY` and `AIDIAL_ID_TOKEN` are set
- **THEN** the configured key is used

#### Scenario: Config ID token over environment
- **WHEN** `id_token` is configured and `AIDIAL_API_KEY` is set
- **THEN** the configured token is used

#### Scenario: Environment tie
- **WHEN** nothing is configured and both `AIDIAL_API_KEY` and `AIDIAL_ID_TOKEN` are set
- **THEN** the token from `AIDIAL_ID_TOKEN` is used

#### Scenario: No credential
- **WHEN** neither attribute is configured and neither environment variable is set
- **THEN** configuration fails with an error naming `api_key`, `id_token`, `AIDIAL_API_KEY` and `AIDIAL_ID_TOKEN`

#### Scenario: Unknown value at plan time
- **WHEN** `api_key` or `id_token` is unknown during configuration
- **THEN** the provider defers configuration without an error, as it does for an unknown `base_url`

### Requirement: Warning for ignored environment variables
Whenever an environment credential variable is set and not used because of precedence, the provider SHALL emit one warning diagnostic naming each ignored variable. A warning SHALL NOT contain a credential value.

#### Scenario: Environment variable overridden by configuration
- **WHEN** `id_token` is configured and `AIDIAL_ID_TOKEN` and `AIDIAL_API_KEY` are set
- **THEN** configuration succeeds with a warning that both variables were ignored

#### Scenario: Environment tie warning
- **WHEN** nothing is configured and both environment variables are set
- **THEN** the token is used and a warning says `AIDIAL_API_KEY` was ignored

#### Scenario: Nothing ignored
- **WHEN** exactly one credential is available from configuration or the environment
- **THEN** no warning is emitted

### Requirement: Token shape check
When the selected credential is an ID token, the provider SHALL reject a value that is not three non-empty dot-separated segments, and SHALL NOT verify its signature, expiry or audience. The error SHALL NOT contain the token.

#### Scenario: Malformed token
- **WHEN** `id_token` is `not-a-jwt`
- **THEN** configuration fails with an error saying the token is not a JWT, without echoing it

#### Scenario: Expired or foreign token
- **WHEN** a well-formed token is expired or issued by an unknown issuer
- **THEN** the provider accepts it at configuration and DIAL Core's rejection is returned by the failing API call

#### Scenario: Whitespace around the token
- **WHEN** the token has leading or trailing whitespace or a trailing newline
- **THEN** the whitespace is trimmed before the check and before it is sent

### Requirement: Credential headers
The client SHALL send an API key in the `Api-Key` header and an ID token as `Authorization: Bearer <token>`, on every request, and SHALL send only one of the two.

#### Scenario: API key request
- **WHEN** the provider is configured with an API key
- **THEN** each request carries `Api-Key` and no `Authorization` header

#### Scenario: ID token request
- **WHEN** the provider is configured with an ID token
- **THEN** each request carries `Authorization: Bearer <token>` and no `Api-Key` header

### Requirement: Credentials are never logged
Request logging SHALL record only whether a credential is present and which kind, and SHALL NOT write an API key or an ID token to logs, diagnostics or error messages.

#### Scenario: Debug log of a request
- **WHEN** a request is logged with `TF_LOG=DEBUG`
- **THEN** the log shows the credential kind and presence and contains neither the key nor the token

### Requirement: Authentication documentation
The README and the generated provider docs SHALL document both credentials, the precedence table, the ignored-variable warning, a GitLab CI and a GitHub Actions example that supply the token, the DIAL Core identity provider and admin-role setup, and the limitation that the provider does not refresh a token.

#### Scenario: Provider docs
- **WHEN** `docs/index.md` is generated
- **THEN** it describes `api_key`, `id_token` and both environment variables

#### Scenario: Token lifetime
- **WHEN** a reader looks for token expiry guidance
- **THEN** the docs state that a token must outlive the whole `terraform apply` and that the provider never refreshes it
