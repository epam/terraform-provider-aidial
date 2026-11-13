# Terraform Provider for DIAL

This provider manages [DIAL Core](https://github.com/epam/ai-dial-core) resources (roles, API keys, models, routes, schemas, toolsets, interceptors, translators, applications and aliases) and reads buckets and statically configured deployments, through the DIAL Core API.

- Registry: [`epam/aidial`](https://registry.terraform.io/providers/epam/aidial)
- Documentation: [`docs/`](docs/) (also rendered on the Registry)
- Contributing, local builds and releasing: [CONTRIBUTING.md](CONTRIBUTING.md)
- Changelog: [CHANGELOG.md](CHANGELOG.md)
- License: [Apache-2.0](LICENSE)

## Table of Contents

- [Installation](#installation)
- [Authentication](#authentication)
- [Resources and data sources](#resources-and-data-sources)
- [Import](#import)
- [Specifications](#specifications)
- [Contributing](#contributing)

## Installation

Terraform 1.5 or later is required. Declare the provider and configure it. Terraform downloads and verifies it from the Registry on `terraform init`.

```hcl
terraform {
  required_providers {
    aidial = {
      source  = "epam/aidial"
      version = "~> 0.3"
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

For GitLab CI and GitHub Actions recipes, the DIAL Core settings that trust a CI identity provider, and the token limitations, see the [CI/CD authentication guide](docs/guides/ci-authentication.md).

## Resources and data sources

Every resource and data source has a generated page with its attributes and an example. Behavior that those pages do not spell out (identity rules, values DIAL Core fills in, fields the provider does not manage) is in the [resource behavior notes](docs/guides/resource-notes.md).

| Resource | Data sources | Purpose |
| --- | --- | --- |
| [`aidial_alias`](docs/resources/alias.md) | [`aidial_alias`](docs/data-sources/alias.md), [`aidial_file_alias`](docs/data-sources/file_alias.md) | Alternative name for a model, application or toolset. |
| [`aidial_application`](docs/resources/application.md) | [`aidial_application`](docs/data-sources/application.md), [`aidial_file_application`](docs/data-sources/file_application.md) | Endpoint-based or schema-based application. |
| [`aidial_application_type_schema`](docs/resources/application_type_schema.md) | [`aidial_application_type_schema`](docs/data-sources/application_type_schema.md), [`aidial_file_application_type_schema`](docs/data-sources/file_application_type_schema.md) | Custom application type schema. |
| | [`aidial_bucket`](docs/data-sources/bucket.md) | The bucket owned by the configured key. |
| [`aidial_catalog_schema`](docs/resources/catalog_schema.md) | [`aidial_catalog_schema`](docs/data-sources/catalog_schema.md) | JSON Schema that models reference through `catalog_schema_id`. |
| [`aidial_interceptor`](docs/resources/interceptor.md) | [`aidial_interceptor`](docs/data-sources/interceptor.md), [`aidial_file_interceptor`](docs/data-sources/file_interceptor.md) | Service in the request pipeline. |
| [`aidial_key`](docs/resources/key.md) | | API key bound to a role. |
| [`aidial_model`](docs/resources/model.md) | [`aidial_model`](docs/data-sources/model.md), [`aidial_file_model`](docs/data-sources/file_model.md) | Model deployment: endpoint, features, pricing, upstreams. |
| [`aidial_role`](docs/resources/role.md) | [`aidial_role`](docs/data-sources/role.md) | Role: limits, share, cost limits, topics. |
| [`aidial_route`](docs/resources/route.md) | | Routing rule: methods, paths, upstreams or a static response. |
| [`aidial_toolset`](docs/resources/toolset.md) | [`aidial_toolset`](docs/data-sources/toolset.md), [`aidial_file_toolset`](docs/data-sources/file_toolset.md) | Toolset (MCP tool server). |
| [`aidial_translator`](docs/resources/translator.md) | [`aidial_translator`](docs/data-sources/translator.md), [`aidial_file_translator`](docs/data-sources/file_translator.md) | Converts requests between interface types. |

Data sources named `aidial_file_*` read deployments from DIAL Core's static configuration by name. They are read-only and have no resource equivalent.

All resources require `bucket` and `path`, with no defaults. Both force replacement, and `id` is `<bucket>/<path>`. `bucket` must be `platform` for most types; toolsets and applications accept any non-empty bucket (for example `public`), and aliases accept `public` and `platform`. See [identity](docs/guides/resource-notes.md#identity-bucket-and-path).

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

For a step-by-step import of an existing model, see the [worked example](docs/guides/resource-notes.md#worked-example-importing-an-existing-model).

## Specifications

Detailed, per-resource behavior specs (schema, client contract, CRUD semantics, state-consistency guarantees, and known gaps against the DIAL Core OpenAPI spec) live under `openspec/`, one directory per capability (e.g. `role-schema`, `model-client`, `route-state-consistency`).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the repository layout, the devcontainer, testing against a local Core, local builds and the release process.
