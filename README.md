# Terraform Provider for DIAL

This provider manages [DIAL Core](https://github.com/epam/ai-dial-core) resources — roles, API keys, models, routes, and catalog schemas — plus read-only bucket, role, model, file-sourced model and catalog schema lookups, through the DIAL Core API.

- Registry: [`epam/aidial`](https://registry.terraform.io/providers/epam/aidial)
- Documentation: [`docs/`](docs/) (also rendered on the Registry)
- License: [Apache-2.0](LICENSE)

## Table of Contents

- [Installation](#installation)
- [Authentication](#authentication)
  - [Precedence](#precedence)
  - [GitLab CI](#gitlab-ci)
  - [GitHub Actions](#github-actions)
  - [DIAL Core setup](#dial-core-setup)
  - [Limitations](#limitations)
- [Resources and data sources](#resources-and-data-sources)
- [Bucket data source](#bucket-data-source)
- [Role resource](#role-resource)
- [Key resource](#key-resource)
- [Model resource](#model-resource)
- [Route resource](#route-resource)
- [Role data source](#role-data-source)
- [File-sourced model data source](#file-sourced-model-data-source)
- [Import](#import)
  - [Worked example: importing an existing model](#worked-example-importing-an-existing-model)
- [State](#state)
- [Specifications](#specifications)
- [API Contract](#api-contract)
- [Repository layout](#repository-layout)
  - [Why this split](#why-this-split)
  - [Structure of the repository](#structure-of-the-repository)
  - [Updating the Core OpenAPI contract](#updating-the-core-openapi-contract)
- [Devcontainer](#devcontainer)
- [Build](#build)
- [Local builds](#local-builds)
  - [Option A: dev overrides (recommended)](#option-a-dev-overrides-recommended)
  - [Option B: local filesystem mirror (persistent install, no `.terraformrc` needed per-shell)](#option-b-local-filesystem-mirror-persistent-install-no-terraformrc-needed-per-shell)
- [Releasing](#releasing)

## Installation

Terraform 1.5 or later is required. Declare the provider and configure it. Terraform downloads and verifies it from the Registry on `terraform init`.

```hcl
terraform {
  required_providers {
    aidial = {
      source  = "epam/aidial"
      version = "~> 0.2"
    }
  }
}

provider "aidial" {
  base_url = "http://127.0.0.1:8080"
  api_key  = var.aidial_api_key
}
```

`base_url` accepts `http` and `https`. It must not include an API path suffix. `api_key` is the value sent in the `Api-Key` header on every request. In CI/CD you can authenticate with an OIDC ID token instead; see [Authentication](#authentication).

## Authentication

The provider authenticates to DIAL Core with exactly one credential:

| Attribute | Environment variable | Sent as |
| --- | --- | --- |
| `api_key` | `AIDIAL_API_KEY` | `Api-Key: <key>` |
| `id_token` | `AIDIAL_ID_TOKEN` | `Authorization: Bearer <token>` |

`id_token` is a ready-made OIDC ID token (JWT), for example the one a GitLab CI job or a GitHub Actions job obtains for itself. The provider does not fetch, exchange or refresh it. It only checks that the value looks like a JWT (three dot-separated segments); DIAL Core validates the signature, issuer, expiry and audience. Both attributes are sensitive. Keep using `api_key` for local development.

```hcl
provider "aidial" {
  base_url = "https://dial-core.example.com"
  id_token = var.aidial_id_token # or leave it out and export AIDIAL_ID_TOKEN
}
```

### Precedence

A credential in the provider block always beats the environment. With nothing in the block, `AIDIAL_ID_TOKEN` beats `AIDIAL_API_KEY`. Whenever an environment variable is set but ignored, the provider emits a warning that names the variable (never its value). An empty value counts as not set.

| `api_key` | `id_token` | `AIDIAL_API_KEY` | `AIDIAL_ID_TOKEN` | Result |
| --- | --- | --- | --- | --- |
| set | set | any | any | Error: both set in configuration |
| set | not set | any | any | `api_key` is used; a warning names each environment variable that is set |
| not set | set | any | any | `id_token` is used; a warning names each environment variable that is set |
| not set | not set | set | set | `AIDIAL_ID_TOKEN` is used; warning that `AIDIAL_API_KEY` was ignored |
| not set | not set | not set | set | `AIDIAL_ID_TOKEN` is used |
| not set | not set | set | not set | `AIDIAL_API_KEY` is used |
| not set | not set | not set | not set | Error: no credential |

### GitLab CI

Request a token with the audience that DIAL Core expects, and let the provider read it from the environment:

```yaml
terraform-apply:
  id_tokens:
    AIDIAL_ID_TOKEN:
      aud: https://dial-core.example.com
  script:
    - terraform apply -auto-approve
```

### GitHub Actions

Grant the job `id-token: write`, request a token for your audience, mask it and export it to the step that runs Terraform:

```yaml
jobs:
  apply:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write
    steps:
      - uses: actions/checkout@v7
      - name: Terraform apply
        env:
          OIDC_AUDIENCE: https://dial-core.example.com
        run: |
          token="$(curl -sSf -H "Authorization: Bearer ${ACTIONS_ID_TOKEN_REQUEST_TOKEN}" \
            "${ACTIONS_ID_TOKEN_REQUEST_URL}&audience=${OIDC_AUDIENCE}" | jq -r .value)"
          echo "::add-mask::${token}"
          AIDIAL_ID_TOKEN="${token}" terraform apply -auto-approve
```

Pull requests from forks and from Dependabot receive no OIDC token, so run the job only for events that do.

### DIAL Core setup

Core must trust the CI system as an identity provider and map the token to a role that may call the admin API. In the Core static settings (`aidial.settings.json`), this example trusts GitHub Actions tokens issued for one repository:

```json
{
  "identityProviders": {
    "github": {
      "jwksUrl": "https://token.actions.githubusercontent.com/.well-known/jwks",
      "issuerPattern": "^https://token\\.actions\\.githubusercontent\\.com$",
      "rolePath": "repository",
      "rolesDelimiter": " ",
      "audience": "https://dial-core.example.com"
    }
  },
  "access": {
    "admin": {
      "rules": [
        { "function": "EQUAL", "source": "roles", "targets": ["my-org/my-infra-repo"] }
      ]
    }
  }
}
```

GitHub tokens have no roles claim and you cannot add custom claims, so the example uses the `repository` claim as the role and grants admin to that exact repository. For GitLab, point `jwksUrl` at `https://gitlab.example.com/oauth/discovery/keys`, `issuerPattern` at your GitLab URL and `rolePath` at a claim you control (for example `project_path`). Pick a claim that a pipeline author cannot choose freely, and always set `audience`. See the [DIAL identity provider guides](https://docs.dialx.ai/tutorials/devops/auth-and-access-control/configure-idps/keycloak) for the full list of settings.

The repository's own CI uses this setup against a throwaway Core (`tests/core/docker-compose.yml`, job `integration-jwt`).

### Limitations

- The provider reads the token once, when Terraform configures it, and never refreshes it. The token must stay valid for the whole `terraform apply`, so set a token lifetime longer than the longest run where your CI system allows it.
- Providers are configured at the start of a run: a token minted after that point is not seen.

## Resources and data sources

| Type | Name | Purpose |
| --- | --- | --- |
| Resource | `aidial_role` | Create/update/delete a role (limits, share, cost limits, topics, ...). |
| Resource | `aidial_key` | Create/update/delete an API key bound to a role. |
| Resource | `aidial_model` | Create/update/delete a model deployment (endpoint, features, pricing, upstreams, ...). |
| Resource | `aidial_route` | Create/update/delete a routing rule (methods, paths, permissions, upstreams, static response, ...). |
| Resource | `aidial_catalog_schema` | Create/update/delete a catalog schema (a JSON Schema that models reference through `catalog_schema_id`). |
| Resource | `aidial_application_type_schema` | Create/update/delete a custom application type schema (a JSON Schema identified by its `$id`). |
| Resource | `aidial_toolset` | Create/update/delete a toolset (an MCP tool server: endpoint, transport, allowed tools, upstream authentication, ...). |
| Resource | `aidial_interceptor` | Create/update/delete an interceptor (a service in the request pipeline that models reference by name through `interceptors`). |
| Resource | `aidial_translator` | Create/update/delete a translator (a service that converts requests from one interface type to another). |
| Resource | `aidial_application` | Create/update/delete an application (an endpoint-based or application-type-schema-based chat deployment). |
| Resource | `aidial_alias` | Create/update/delete an alias (an alternative invocation name that points at a model, application or toolset). |
| Data source | `aidial_bucket` | Look up a bucket's `bucket`/`appdata`. |
| Data source | `aidial_role` | Look up an existing role without managing it. |
| Data source | `aidial_model` | Look up an existing API-managed model by `bucket`/`path` without managing it. |
| Data source | `aidial_file_model` | Look up a file-sourced model (statically configured in DIAL Core, not bucket-scoped, no resource equivalent). |
| Data source | `aidial_catalog_schema` | Look up an existing catalog schema without managing it. |
| Data source | `aidial_application_type_schema` | Look up an existing API-managed application type schema by `bucket`/`path` without managing it. |
| Data source | `aidial_file_application_type_schema` | Look up a file-sourced application type schema (statically configured in DIAL Core, by name only, no resource equivalent). |
| Data source | `aidial_toolset` | Look up an existing API-managed toolset by `bucket`/`path` without managing it. |
| Data source | `aidial_file_toolset` | Look up a file-sourced toolset (statically configured in DIAL Core, by name only, no resource equivalent). |
| Data source | `aidial_interceptor` | Look up an existing API-managed interceptor by `bucket`/`path` without managing it. |
| Data source | `aidial_file_interceptor` | Look up a file-sourced interceptor (statically configured in DIAL Core, by name only, no resource equivalent). |
| Data source | `aidial_translator` | Look up an existing API-managed translator by `bucket`/`path` without managing it. |
| Data source | `aidial_file_translator` | Look up a file-sourced translator (statically configured in DIAL Core, by name only, no resource equivalent). |
| Data source | `aidial_application` | Look up an existing API-managed application by `bucket`/`path` without managing it. |
| Data source | `aidial_file_application` | Look up a file-sourced application (statically configured in DIAL Core, by name only, no resource equivalent). |
| Data source | `aidial_alias` | Look up an existing API-managed alias by `bucket`/`path` without managing it. |
| Data source | `aidial_file_alias` | Look up a file-sourced alias (statically configured in DIAL Core, by name only, no resource equivalent). |

All eleven resources, and the `aidial_role`, `aidial_model`, `aidial_catalog_schema`, `aidial_application_type_schema`, `aidial_toolset`, `aidial_interceptor`, `aidial_translator`, `aidial_application` and `aidial_alias` data sources, share the same identity model: `bucket` and `path` are required, with no defaults. `path` must be non-empty and may contain `/` (except for interceptors, translators and aliases, where DIAL Core rejects it, and for applications outside the `public` bucket). Omitting either fails validation at plan time. Both force replacement on change, and `id` is persisted as `<bucket>/<path>`. `bucket` must be `platform` for every type except toolsets, applications and aliases (DIAL Core stores those resources only there; the bucket returned by `aidial_bucket` is not accepted). Toolsets and applications accept any non-empty bucket, for example `public`, and leave it to DIAL Core to reject the ones it does not allow. Aliases accept `public` and `platform`.

## Bucket data source

`aidial_bucket` takes no arguments — it always resolves to the bucket owned by the configured `api_key`.

```hcl
data "aidial_bucket" "current" {}
```

The API response is stored in state as `bucket` and `appdata`.

## Role resource

```hcl
resource "aidial_role" "example" {
  bucket = "platform"
  path   = "my-role"

  description  = "Example role"
  display_name = "Example Role"

  cost_limit = {
    day   = 10
    week  = 50
    month = 200
  }

  topics = ["Example"]

  limits = {
    sample = jsonencode({ actions = ["read"] })
  }

  share = {
    sample = jsonencode({
      invitation_ttl     = 3600
      max_accepted_users = 10
    })
  }
}
```

`limits` and `share` are maps whose values are JSON strings. Use `jsonencode(...)` so Terraform serializes nested objects safely; the provider decodes those strings only far enough to canonicalize them and stores them back as JSON strings, so `limits` values are not required to match any particular shape (DIAL Core's documented `Limit` object, or ad-hoc shapes such as `{"enabled": true}`, both work).

`name` is optional: DIAL Core addresses the role by `bucket` and `path` and derives its name from `path`, so you can leave `name` out and it is read back from the API. When you do configure it, the value is sent and never rewritten by the provider, even if the API returns a canonicalized name (for example DIAL Core may strip a `roles/<bucket>/` prefix from the value it echoes back). The API's own value is exposed separately as the computed `api_name` attribute. `description`, `display_name`, and `topics` are optional/computed and are preserved across updates when left unset in config, so applying changes to an existing role does not clear metadata Terraform isn't managing. `created_at`, `updated_at`, and `granted_keys` are computed/read-only.

## Key resource

```hcl
resource "aidial_key" "example" {
  bucket = "platform"
  path   = "my-key"

  key     = var.aidial_key_value
  project = "my-project"
  role    = "developer"
  roles   = ["developer"]
  secured = true
}
```

## Model resource

```hcl
resource "aidial_model" "example" {
  bucket       = "platform"
  path         = "my-model"
  type         = "CHAT"
  display_name = "My Model"
  endpoint     = "https://example.invalid/v1/chat/completions"

  limits = {
    max_prompt_tokens     = 200000
    max_completion_tokens = 128000
    max_total_tokens      = 300000
  }

  upstreams = [
    {
      endpoint = "https://example.invalid/v1/chat/completions"
      tier     = 0
      weight   = 1
    },
  ]
}
```

`api_name` is the API's canonical name, analogous to `aidial_role`. `type` is one of `CHAT`, `COMPLETION`, `EMBEDDING`. See `openspec/model-schema` for the full attribute list and the known gaps versus the DIAL Core `Model` schema (for example `responses_defaults` and per-interface settings are not yet exposed).

`defaults` is a free-form object passed to DIAL Core as-is, so keys inside it keep the API's spelling:

```hcl
  defaults = {
    max_tokens = 64000
    custom_fields = {
      configuration = { thinking = { type = "disabled" } }
    }
  }
```

Attribute names *outside* `defaults` are snake_case in Terraform and camelCase in the API (for example `upstreams[].extra_data` is sent as `extraData`). Terraform silently ignores a camelCase key copied from an API response, so rename keys when copying.

## Route resource

```hcl
resource "aidial_route" "example" {
  bucket = "platform"
  path   = "my-route"

  methods = ["GET", "POST"]
  paths   = ["/my-route/.*"]

  upstreams = [
    {
      endpoint = "https://example.invalid/v1/chat/completions"
      tier     = 0
      weight   = 1
    },
  ]
}
```

A route may instead configure `response` (a static `body`/`status` returned without forwarding upstream) in place of `upstreams`. See `openspec/route-schema` for the full attribute list.

## Toolset resource

```hcl
resource "aidial_toolset" "example" {
  bucket      = "platform"
  path        = "my-toolset"
  description = "Search and fetch tools served over MCP"
  endpoint    = "https://mcp.example.invalid/mcp"
  transport   = "HTTP"

  allowed_tools = ["search", "fetch"]

  auth_settings = {
    authentication_type = "API_KEY"
    api_key_header      = "X-Api-Key"
  }
}
```

Attribute names are snake_case in Terraform (and in DIAL Core's toolset JSON). `provider_name` is the API's `provider`, which Terraform reserves as an attribute name. A few notes:

- DIAL Core fills in values you leave out: `reference` (a generated identifier), `transport` (`HTTP`), `forward_auth_token`, `forward_per_request_key` (`false`), `max_retry_attempts` (`1`), empty `allowed_tools`, `dependencies`, `description_keywords` and `interceptors`, and `auth_settings` of type `NONE`. They are read back into state, so deleting such a line from the configuration does not reset it; set it explicitly instead. `author` is set by DIAL Core from the caller (for toolsets outside the `platform` bucket).
- `auth_settings.client_secret` is sensitive and never returned by DIAL Core. State keeps the configured value, and it is null after import. DIAL Core derives `code_challenge` itself, and the sign-in status fields and `code_verifier` are session state, so none of them are attributes. For OAuth, set the full block, as in `examples/resources/aidial_toolset`: DIAL Core probes the endpoints when it saves OAuth settings in the `platform` bucket, or incomplete OAuth settings in other buckets.
- `features` and `interfaces` have the same shape as on `aidial_model`. `defaults`, `responses_defaults` and `catalog_properties` are free-form objects.
- `display_name` and `intro` are plain strings. The locale-map form DIAL Core allows is not supported, and neither are `created_at`/`updated_at`.

`aidial_toolset` (data source) reads one by `bucket` and `path`, and `aidial_file_toolset` reads a toolset from DIAL Core's static configuration by `name`. Neither exposes `client_secret`. See `openspec/specs/toolset-resource` for the full behavior.

## Interceptor resource

```hcl
resource "aidial_interceptor" "example" {
  bucket       = "platform"
  path         = "my-interceptor"
  display_name = "Content filter"
  endpoint     = "https://interceptor.example.invalid/intercept"

  forward_auth_token = true
  max_retry_attempts = 2
}
```

`aidial_interceptor` registers an interceptor in DIAL Core. Models list interceptor names in their `interceptors` attribute; the provider does not link the two resources or check that the names exist, so see the DIAL Core documentation for how a name resolves to an interceptor. A few notes:

- `endpoint` is required. `forward_auth_token` (`false`) and `max_retry_attempts` (`1`) are filled in by DIAL Core when you leave them out, so they are read back into state.
- `features` and `interfaces` have the same shape as on `aidial_model`. `defaults` and `catalog_properties` are free-form objects.
- Only the attributes above and in the generated docs are managed. DIAL Core replaces the whole interceptor on every save, so a value set out of band in a field the provider does not expose (for example `dependencies`, `description_keywords`, `responses_endpoint`, `input_attachment_types` or `override_name`) is removed by the next apply. `author`, `created_at` and `updated_at` are not attributes.
- DIAL Core accepts only the `platform` bucket for interceptors, and rejects `/` in the `path`.

`aidial_interceptor` (data source) reads one by `bucket` and `path`, and `aidial_file_interceptor` reads an interceptor from DIAL Core's static configuration by `name`. See `openspec/specs/interceptor-resource` for the full behavior.

## Translator resource

```hcl
resource "aidial_translator" "example" {
  bucket   = "platform"
  path     = "openai-to-claude"
  base_url = "https://translator.example.invalid"
  in       = "openaiChatCompletions"
  out      = "anthropicMessages"
}
```

`aidial_translator` registers a translator in DIAL Core: a service that converts requests from the `in` interface type to the `out` interface type. A few notes:

- `base_url`, `in` and `out` are all required, as DIAL Core rejects a translator without them. `in` and `out` accept `openaiChatCompletions`, `openaiEmbeddings`, `openaiResponses` and `anthropicMessages`. `in` is a keyword in HCL expressions but is a valid attribute name.
- The provider does not link translators to the `translator` entry of a model or toolset interface, which is still not modeled.
- DIAL Core replaces the whole translator on every save, so a value set out of band in a field the provider does not expose is removed by the next apply.
- DIAL Core accepts only the `platform` bucket for translators, and rejects `/` in the `path`.

`aidial_translator` (data source) reads one by `bucket` and `path`, and `aidial_file_translator` reads a translator from DIAL Core's static configuration by `name`. See `openspec/specs/translator-resource` for the full behavior.

## Alias resource

```hcl
resource "aidial_alias" "example" {
  bucket      = "public"
  path        = "fast-chat"
  present_as  = "model"
  target_path = "applications/my-application"

  display_name = "Fast chat"
  user_roles   = ["admin"]
}
```

`aidial_alias` creates an alternative invocation name for a model, application or toolset. A few notes:

- `present_as` (`model`, `application` or `toolset`) and `target_path` (`models/<name>`, `applications/<name>` or `toolsets/<name>`) are required. An alias shown as a toolset must point at a toolset, and one shown as a model or application cannot; the provider checks that at plan time. Core does not check that the target exists.
- The other attributes are `display_name`, `description`, `display_version`, `icon_url`, `intro`, `user_roles` and `description_keywords`. DIAL Core fills in nothing but the name.
- DIAL Core accepts the `public` and `platform` buckets for aliases (the OpenAPI document says `public` only), and rejects `/` in the `path`.
- Core accepts only `path` in a target, so there is no `target.id` or `target.version`, and it rejects `endpoint`, `features`, `catalog_properties` and `catalog_schema_id` on an alias. DIAL Core replaces the whole alias on every save, so a value set out of band in a field the provider does not expose (`author`, `created_at`, `updated_at`) is removed by the next apply. Core accepts a locale map for `display_name` but returns a plain string, so only the plain form is supported.

`aidial_alias` (data source) reads one by `bucket` and `path`, and `aidial_file_alias` reads an alias from DIAL Core's static configuration by `name`. See `openspec/specs/alias-resource` for the full behavior.

## Application resource

```hcl
resource "aidial_application" "example" {
  bucket       = "platform"
  path         = "my-application"
  display_name = "Support assistant"
  endpoint     = "https://application.example.invalid/chat"

  interceptors = ["my-interceptor"]
}
```

`aidial_application` registers an application in DIAL Core. A few notes:

- Set exactly one of `endpoint` (endpoint-based) and `application_type_schema_id` (schema-based, with its values in `application_properties`). Core rejects both together and rejects neither, so the provider fails at plan time. `application_type_schema_id` takes the `schema_id` of an `aidial_application_type_schema`.
- `forward_auth_token` (`false`), `max_retry_attempts` (`1`) and `interceptors` (empty) are filled in by DIAL Core when you leave them out, so they are read back into state. `interceptors` lists interceptor names; the provider does not check that they exist.
- `features` and `interfaces` have the same shape as on `aidial_model`. `defaults`, `application_properties` and `catalog_properties` are free-form objects.
- Only the attributes above and in the generated docs are managed. DIAL Core replaces the whole application on every save, so code-based applications (`function`) and values set out of band in `mcp`, `routes`, `external_services` or other fields the provider does not expose are removed by the next apply. `author`, `created_at` and `updated_at` are not attributes.
- DIAL Core accepts the `platform` and `public` buckets. A path with `/` works only in `public`. For a `public` application Core returns `name` as `applications/public/<path>`.

`aidial_application` (data source) reads one by `bucket` and `path`, and `aidial_file_application` reads an application from DIAL Core's static configuration by `name`. See `openspec/specs/application-resource` for the full behavior.

## Role data source

```hcl
data "aidial_role" "existing" {
  bucket = "platform"
  path   = "my-role"
}
```

## File-sourced model data source

DIAL Core has two separate, non-interchangeable storage backends for models: **resource-storage models**, created dynamically via `PUT /v1/models/{bucket}/{path}` and managed by `aidial_model` above, and **file-sourced models**, defined in DIAL Core's own static configuration at startup and exposed only read-only, with no bucket/path at all. `GET /openai/models/{model_name}` (the endpoint chat clients use to discover available models) succeeds for a model regardless of which backend defines it, so it *cannot* tell you which one to use — if `terraform import aidial_model.x <bucket>/<path>` 404s no matter what bucket you try, the model is very likely file-sourced, not missing.

`aidial_file_model` reads a file-sourced model by name — there is no resource equivalent, since the API has no `PUT`/`DELETE` for this backend:

```hcl
data "aidial_file_model" "gpt5" {
  name = "gpt-5.6-terra-2026-07-09"
}
```

It returns the same computed attributes as `aidial_model` (`type`, `features`, `limits`, `pricing`, `upstreams`, etc.), plus `api_name`, but has no `id`/`bucket`/`path`.

## Import

The `aidial_role`, `aidial_key`, `aidial_model`, `aidial_route`, `aidial_toolset`, `aidial_interceptor`, `aidial_translator`, `aidial_application` and `aidial_alias` resources support import using `<bucket>/<path>`:

```bash
terraform import aidial_role.example '<bucket>/<path>'
terraform import aidial_key.example '<bucket>/<path>'
terraform import aidial_model.example '<bucket>/<path>'
terraform import aidial_route.example '<bucket>/<path>'
terraform import aidial_toolset.example '<bucket>/<path>'
terraform import aidial_interceptor.example '<bucket>/<path>'
terraform import aidial_translator.example '<bucket>/<path>'
terraform import aidial_application.example '<bucket>/<path>'
terraform import aidial_alias.example '<bucket>/<path>'
```

Import only sets `id`, `bucket`, and `path`; every other attribute, including `name`, is populated by the subsequent refresh.

### Worked example: importing an existing model

`type` is `Required`, so Terraform still needs *some* value for it in your config before it will let you write a resource block at all — even though import itself doesn't read your config values for it. (`name` is optional and can be left out.) The practical flow is: write a placeholder block, import, inspect what actually came back, then fix the config to match.

1. Write a minimal block for the model you want to bring under management, using whatever bucket/path it actually lives at remotely. The `type` value here is just a placeholder to satisfy the schema — they'll be corrected in step 3:

   ```hcl
   resource "aidial_model" "gpt5" {
     bucket = "platform"
     path   = "gpt-5-2025-08-07"
     type   = "CHAT"
   }
   ```

2. Import it:

   ```bash
   terraform import aidial_model.gpt5 platform/gpt-5-2025-08-07
   ```

   This populates `id`, `bucket`, and `path` directly, then performs a `Read` that fills in every other attribute — including the real `name` — from the live API response.
3. Inspect what actually got imported, and copy the real values back into your config:

   ```bash
   terraform state show aidial_model.gpt5
   ```

   Update `type` (and `endpoint`, etc. if they differ from your placeholders) in the `.tf` file to match what `state show` printed.
4. Confirm the config and state now agree:

   ```bash
   terraform plan
   ```

   A clean `No changes. Your infrastructure matches the configuration.` means the import is fully reconciled. If `plan` still shows a change to `name` (or any other `Required` attribute), your config doesn't yet match the value `state show` printed in step 3 — copy it over exactly, including case and any provider-side canonicalization (e.g. a stripped path prefix), and re-run `plan`.

This same four-step flow applies to `aidial_role`, `aidial_key`, and `aidial_route` — only the attribute names differ.

**Local names can't contain dots.** Many real model/route paths look like version strings (e.g. `gpt-5.6-terra-2026-07-09`). Terraform resource addresses use `.` as a segment separator (`type.name.attribute`), so if you reuse that path verbatim as the local name in `resource "aidial_model" "gpt-5.6-terra-2026-07-09"`, both the config and any `terraform import`/`state show` command referencing it will fail to parse — typically with `Error: Attribute name required`, since Terraform tries to read whatever follows the extra dot as an attribute name. Hyphens and underscores are fine; dots aren't. Use a dot-free local name (e.g. `gpt-5-6-terra-2026-07-09`) while keeping the dotted value in `path`/`bucket`, which are just string values and unaffected by this rule:

```hcl
resource "aidial_model" "gpt-5-6-terra-2026-07-09" {
  bucket = "platform"
  path   = "gpt-5.6-terra-2026-07-09"
  type   = "CHAT"
}
```

```bash
terraform import aidial_model.gpt-5-6-terra-2026-07-09 platform/gpt-5.6-terra-2026-07-09
```

## State

- `aidial_role` writes `bucket`, `path`, `name`, `api_name`, `description`, `display_name`, `limits`, `share`, cost-limit fields, `topics`, `granted_keys`, `created_at`, `updated_at`, and API-returned `status` to Terraform state.
- `aidial_key` writes `bucket`, `path`, `key`, `project`, `role`, `roles`, `secured`, `name`, and `status`.
- `aidial_model` and `aidial_route` write their full attribute sets, including nested objects/lists (`features`, `limits`, `pricing`, `upstreams` for models; `response`, `attachment_paths`, `upstreams` for routes), normalized to explicit null objects/lists/sets when the API omits them, to keep repeated plans free of diff churn.

## Specifications

Detailed, per-resource behavior specs (schema, client contract, CRUD semantics, state-consistency guarantees, and known gaps against the DIAL Core OpenAPI spec) live under `openspec/`, one directory per capability (e.g. `role-schema`, `model-client`, `route-state-consistency`).

## API Contract

The provider uses these endpoints:

- `GET /v1/bucket` for the `aidial_bucket` data source (returns the calling API key's own bucket; it takes no arguments).
- `GET /v1/admin/config/file/models/{name}` for the `aidial_file_model` data source (file-sourced models: no bucket, read-only).
- `GET|PUT|DELETE /v1/roles/{bucket}/{path}` for `aidial_role`.
- `GET|PUT|DELETE /v1/keys/{bucket}/{path}` for `aidial_key`.
- `GET|PUT|DELETE /v1/models/{bucket}/{path}` for `aidial_model`.
- `GET|PUT|DELETE /v1/routes/{bucket}/{path}` for `aidial_route`.
- `GET|PUT|DELETE /v1/toolsets/{bucket}/{path}` for `aidial_toolset`. DIAL Core 0.50.1 reads and writes toolsets with snake_case keys and expects `/` in the path to stay a `/`, which differs from the OpenAPI document, so the client follows what Core does.
- `GET /v1/admin/config/file/toolsets/{name}` for the `aidial_file_toolset` data source (file-sourced toolsets: no bucket, read-only).
- `GET|PUT|DELETE /v1/interceptors/{bucket}/{path}` for `aidial_interceptor`.
- `GET /v1/admin/config/file/interceptors/{name}` for the `aidial_file_interceptor` data source (file-sourced interceptors: no bucket, read-only).
- `GET|PUT|DELETE /v1/translators/{bucket}/{path}` for `aidial_translator`.
- `GET /v1/admin/config/file/translators/{name}` for the `aidial_file_translator` data source (file-sourced translators: no bucket, read-only).
- `GET|PUT|DELETE /v1/applications/{bucket}/{path}` for `aidial_application`. DIAL Core 0.50.1 reads and writes applications with snake_case keys and expects `/` in the path to stay a `/`, so the client follows what Core does.
- `GET /v1/admin/config/file/applications/{name}` for the `aidial_file_application` data source (file-sourced applications: no bucket, read-only).
- `GET|PUT|DELETE /v1/aliases/{bucket}/{path}` for `aidial_alias`.
- `GET /v1/admin/config/file/aliases/{name}` for the `aidial_file_alias` data source (file-sourced aliases: no bucket, read-only).

Every request uses the `Api-Key` header. `GET` returns `404` for a missing resource (treated as absent, not an error); `DELETE` treats `404` as a successful no-op; `PUT` on save returns only `{"name": ...}`, so every create/update re-reads the resource via `GET` to obtain the authoritative state. DIAL Core also supports `If-Match`/`If-None-Match`/`ETag`/`304`/`412` optimistic-concurrency headers on all three mutating operations for models and routes; the provider does not currently send or interpret them (see `openspec/model-client` and `openspec/route-client`).

## Repository layout

### Why this split

The provider code, its behavior specs, the local DIAL Core stack and the integration manifests are kept in separate top-level directories on purpose, so each has a single source of truth and a change touches one place:

- Resource behavior is specified once under `openspec/` and implemented once under `internal/`. When a resource's behavior changes, update the spec and the code together; don't describe behavior only in a README or a commit message. System-wide rules that apply to every resource (for example bucket/path defaults) live in their own spec (`openspec/resource-identity`), not copied into each per-resource spec.
- The disposable DIAL Core stack used by CI and by local runs is defined once under `stub/`. Model configs for it live in `stub/models/`; don't copy them into `examples/` or `tests/`.
- `tests/integration/` holds manifests that run in CI and locally (`make test-integration`) against an ephemeral DIAL Core defined in `tests/core/`. `stub/models/` holds manifests that run against the local stub. Keep them separate: they have different backends and different expectations.
- CI and release pipelines live in `.github/workflows/`; release packaging is configured in `.goreleaser.yml`.

### Structure of the repository

```text
.
├── main.go                     # Provider entry point.
├── internal/
│   ├── provider/               # Terraform resources and data sources (role, key, model, route,
│   │                           # bucket, file model), shared helpers, and their unit tests.
│   └── client/                 # DIAL Core HTTP client and its tests.
├── openspec/                   # Behavior specs, one directory per capability
│                               # (e.g. model-schema, role-client, resource-identity).
├── openapicore/                # DIAL Core OpenAPI contract the client and specs are checked against
│                               # (copied from an upstream release tag; see SOURCE.md).
├── stub/                       # Local DIAL Core (docker-compose) plus model configs in stub/models/;
│                               # used by the opt-in stub-core workflow.
├── tests/integration/          # Integration manifests applied against an ephemeral DIAL Core in CI.
├── examples/                   # Snippets embedded into the generated docs (tfplugindocs layout).
├── docs/                       # Generated provider documentation (tfplugindocs; committed, CI checks for drift).
├── scripts/                    # rename-provider-name.sh and update-openapi.sh (maintainers).
├── .github/                    # CI, release workflow and Dependabot configuration.
└── .goreleaser.yml             # Release packaging for the Terraform Registry.
```

### Updating the Core OpenAPI contract

`openapicore/open_api_core.yaml` is an unmodified copy of `docs/open_api_core.yaml` from a tagged [epam/ai-dial-core](https://github.com/epam/ai-dial-core) release; `openapicore/SOURCE.md` records which tag. To refresh it:

```bash
make openapi TAG=0.50.1
```

The target also sets the default image tag in `scripts/core.sh` to the same tag. It fails without changing anything if the tag or file does not exist. After a refresh, diff the contract and update any notes in `openspec/*-schema` and `openspec/*-client` that cite it. CI uses the checked-in copy and never fetches it.

## Devcontainer

`.devcontainer/` gives a ready-to-use development environment (VS Code "Reopen in Container", or any devcontainer-compatible tool). A fresh container provides:

| Tool | Purpose |
|------|---------|
| `go`, `gopls`, `goimports` | build, language server, import formatting |
| `dlv` | debugger (used by the Go extension) |
| `golangci-lint`, `tfproviderlint` | linting (`.golangci.yml`) |
| `markdownlint-cli2` | Markdown linting (`.markdownlint-cli2.jsonc`; excludes `openspec/`) |
| `govulncheck`, `zizmor`, `gitleaks` | local runs of the `Security` workflow checks |
| `goreleaser`, `syft` | local release dry run (`goreleaser release --snapshot --clean`) with SBOMs |
| `tfplugindocs` | provider documentation generation |
| `terraform` | running configurations against a local build |
| `gh` | pushing to GitHub and reading CI status (see below) |

Tool versions are pinned as `ARG`s at the top of `.devcontainer/Dockerfile`. Go tools are installed into `/usr/local/go-tools` (`GOBIN`, on `PATH`); `GOPATH` is `/home/node/go`.

On creation, `post-create.sh` runs `go mod download`, installs the pre-commit hooks and, if `~/.terraformrc` does not exist, writes it with a `dev_overrides` entry for `epam/aidial` pointing at `<workspace>/dist/linux_amd64`. An existing `~/.terraformrc` (the home directory is a persistent volume) is never overwritten; edit it by hand if needed.

GitHub access: put `GH_TOKEN=<token>` in `~/devcontainer.env` on the host and rebuild the container. Use a fine-grained token for this repository with Contents, Pull requests and Workflows read/write plus Actions read (a classic token needs `repo` and `workflow`). `post-create.sh` then configures the git credential helper (`gh auth setup-git`) and adds a `github` remote next to `origin`. Push with `git push -u github <branch>` and check CI with `gh run list --branch <branch>` or `gh run watch`. Never commit the token.

Common tasks:

```bash
make install-dev   # build into dist/linux_amd64, picked up by dev_overrides (skip `terraform init`)
make build         # build ./terraform-provider-aidial
make test          # go test ./... -count=1 -race
make vet fmt lint  # go vet; gofmt/goimports/terraform fmt; golangci-lint + tfproviderlint
make lint-md       # markdownlint-cli2 on README/AGENTS and other repo Markdown
make docs          # tfplugindocs generate (writes docs/, which is committed; CI fails on drift)
```

To debug a test, open a `_test.go` file and run **Debug test (current file's package)** from `.vscode/launch.json` (or select a test name and use **Debug test (selected name)**). Debugging a running provider binary is not set up, since `main.go` has no `-debug` flag.

## Testing against a local Core

The CI integration job and the Go acceptance tests run against a throwaway DIAL Core plus Valkey defined once in `tests/core/docker-compose.yml`. Neither the devcontainer nor the agent running in it needs Docker access: you start the Core on the host and the devcontainer only makes HTTP calls to it. This part is optional; skip it if you only run unit tests.

### Core on the host, tests in the devcontainer (recommended)

1. On the host (not in the devcontainer), from a checkout of this repository:

   ```bash
   scripts/core.sh up                          # CORE_VERSION=latest scripts/core.sh up to use another tag (default: the tag in `openapicore/SOURCE.md`)
   ```

   It starts Core and Valkey, waits for `/health` and prints the URL. The Core is published on `0.0.0.0:8080` so containers can reach it, which also exposes it to your network, so firewall the port or set `CORE_BIND=127.0.0.1` when you only use it from the host. It uses the throwaway API key `ci-check-key` and holds no real data. `CORE_PORT` changes the port. A `/health` answer of `401` still counts as up.

2. In the devcontainer, nothing needs to be set. With no Docker daemon reachable, the scripts default to a Core on the host at `http://host.docker.internal:8080` (`CORE_PORT` changes the port). If that name does not resolve, they fall back to the container's default gateway. `host.docker.internal` is mapped by `runArgs` in `.devcontainer/devcontainer.json`, which only takes effect after you rebuild the devcontainer. Set `CORE_URL` (for example in `~/devcontainer.env`) to override the address.

   ```bash
   scripts/core.sh url     # shows the Core URL the scripts will use
   make test-integration   # validate, plan, apply, null-vs-empty drift check, destroy (same as CI)
   make testacc            # Go acceptance tests (TestAcc*, gated by TF_ACC)
   ```

   In this mode the scripts only wait for the Core's `/health`; they never start, stop or delete it. For manual `terraform apply`, use `base_url = "$(scripts/core.sh url)"` and `api_key = "ci-check-key"`.

3. Back on the host, remove the Core when you are done, or run `up` again for a clean one (it recreates the Core):

   ```bash
   scripts/core.sh down
   ```

The runs create resources with per-run unique paths, but the Core keeps its state between runs until `down` or `up`.

### Core started by the scripts (host with Docker)

On a machine with a reachable Docker daemon (not the devcontainer, which has no Docker) and without `CORE_URL`, `make test-integration` and `make testacc` start a fresh Core themselves, run, and remove it. To use an already running Core there instead, set `CORE_URL=http://localhost:8080`. The devcontainer deliberately has no Docker client: any daemon it could control would be a way out of it.

`CORE_VERSION=<tag>` picks the image in both modes. Plain `go test ./...` skips the acceptance tests and needs no Core or Docker. Without a reachable Core or daemon the scripts stop with a message pointing here.

## Build

```bash
go mod tidy
go build -o terraform-provider-aidial
```

To run Terraform against a local build, see [Local builds](#local-builds).

## Local builds

Unreleased builds are not on the Registry, so point Terraform at a local binary.

### Option A: dev overrides (recommended)

Point Terraform straight at the freshly-built binary's directory — no version number, no reinstalling after every rebuild.

1. Build the binary into its own directory (matching your OS/arch), e.g. `dist/darwin_arm64/terraform-provider-aidial`:

   ```bash
   mkdir -p dist/darwin_arm64
   go build -o dist/darwin_arm64/terraform-provider-aidial
   ```

2. Create (or extend) a CLI configuration file — by default `~/.terraformrc` (`%APPDATA%\terraform.rc` on Windows), or any file referenced by the `TF_CLI_CONFIG_FILE` environment variable:

   ```hcl
   provider_installation {
     dev_overrides {
       "epam/aidial" = "/absolute/path/to/dist/darwin_arm64"
     }
     direct {}
   }
   ```

   The path must be a directory, not the binary itself, and must be absolute.
3. Run `terraform plan`/`apply` as usual — **skip `terraform init`** with dev overrides in place; Terraform will warn ("provider development overrides are in effect") and use the override instead of resolving `required_providers` at all. Any `version` constraint in `required_providers` is ignored while the override is active.
4. Rebuilding the binary picks up immediately on the next `terraform` command — no reinstall step.

### Option B: local filesystem mirror (persistent install, no `.terraformrc` needed per-shell)

Terraform also supports installing providers into `~/.terraform.d/plugins/` so a plain `terraform init` finds them like a normal registry provider. Unlike dev overrides, this layout is picky about both the directory *and* the binary name: the binary must be suffixed `_v<version>`, and `required_providers` must pin that exact `version`.

```bash
VERSION=0.0.1
mkdir -p ~/.terraform.d/plugins/registry.terraform.io/epam/aidial/${VERSION}/darwin_arm64
cp terraform-provider-aidial "~/.terraform.d/plugins/registry.terraform.io/epam/aidial/${VERSION}/darwin_arm64/terraform-provider-aidial_v${VERSION}"
```

For Intel macOS use `darwin_amd64`; for Linux use `linux_amd64` (or the appropriate `<os>_<arch>` pair). Then pin the version so Terraform's local-mirror resolution can match it:

```hcl
terraform {
  required_providers {
    aidial = {
      source  = "epam/aidial"
      version = "0.0.1"
    }
  }
}
```

With this in place, `terraform init` (no dev overrides needed) finds and uses the locally-installed binary.

## Releasing

Pushing a `vX.Y.Z` tag runs `.github/workflows/release.yml`, which uses GoReleaser (`.goreleaser.yml`) to build the zips for every supported platform, `terraform-provider-aidial_<version>_SHA256SUMS`, its detached GPG signature and the Registry manifest, and attaches them to a GitHub release. The Registry ingests new versions from those releases.

One-time setup (repository owner; the namespace is the GitHub account that owns the repository):

1. Create a GPG key without expiry for signing: `gpg --full-generate-key`.
2. Add the public key (`gpg --armor --export <id>`) under the Registry namespace's signing keys in [HashiCorp Cloud / Registry settings](https://registry.terraform.io/settings/gpg-keys).
3. Add the secrets `GPG_PRIVATE_KEY` (`gpg --armor --export-secret-keys <id>`) and `PASSPHRASE` to the `release` environment (see `SECURITY.md`).
4. Publish the provider once from the Registry UI (Publish → Provider, GitHub sign-in).

For each release (maintainers):

1. Move the `CHANGELOG.md` entries from `Unreleased` into a `## X.Y.Z - YYYY-MM-DD` section. That section becomes the GitHub release notes, and the release fails if it is missing or empty.
2. Create an annotated tag and push it: `git tag -a vX.Y.Z -m "vX.Y.Z" && git push origin vX.Y.Z`. Tags must match `vX.Y.Z`.
3. Approve the deployment to the `release` environment when the workflow asks for it.
4. The workflow then downloads the published assets and verifies the GPG signature, every checksum and the build attestation. Confirm that it passed and that the GitHub release lists the zips, `_SHA256SUMS`, `_SHA256SUMS.sig` and `_manifest.json`; the Registry picks the version up from there.

Test a release locally without publishing or signing:

```bash
goreleaser release --snapshot --clean --skip=sign
```
