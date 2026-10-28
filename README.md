# Terraform Provider for DIAL

This provider manages [DIAL Core](https://github.com/epam/ai-dial-core) resources — roles, API keys, models, and routes — plus read-only bucket/role lookups, through the DIAL Core API.

- Registry: [`epam/aidial`](https://registry.terraform.io/providers/epam/aidial)
- Documentation: [`docs/`](docs/) (also rendered on the Registry)
- License: [Apache-2.0](LICENSE)

## Table of Contents

- [Installation](#installation)
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

Declare the provider and configure it. Terraform downloads and verifies it from the Registry on `terraform init`.

```hcl
terraform {
  required_providers {
    aidial = {
      source  = "epam/aidial"
      version = "~> 0.1"
    }
  }
}

provider "aidial" {
  base_url = "http://127.0.0.1:8080"
  api_key  = var.aidial_api_key
}
```

`base_url` accepts `http` and `https`. It must not include an API path suffix. `api_key` is the value sent in the `Api-Key` header on every request.

## Resources and data sources

| Type | Name | Purpose |
| --- | --- | --- |
| Resource | `aidial_role` | Create/update/delete a role (limits, share, cost limits, topics, ...). |
| Resource | `aidial_key` | Create/update/delete an API key bound to a role. |
| Resource | `aidial_model` | Create/update/delete a model deployment (endpoint, features, pricing, upstreams, ...). |
| Resource | `aidial_route` | Create/update/delete a routing rule (methods, paths, permissions, upstreams, static response, ...). |
| Data source | `aidial_bucket` | Look up a bucket's `bucket`/`appdata`. |
| Data source | `aidial_role` | Look up an existing role without managing it. |
| Data source | `aidial_file_model` | Look up a file-sourced model (statically configured in DIAL Core, not bucket-scoped, no resource equivalent). |

All four resources share the same identity model: `bucket` (default `platform`) and `path` (default: the resource `name`; required for `aidial_key`, which has no configurable name) are optional and force replacement on change, and `id` is persisted as `<bucket>/<path>`.

## Bucket data source

`aidial_bucket` takes no arguments — it always resolves to the bucket owned by the configured `api_key`.

```hcl
data "aidial_bucket" "current" {}
```

The API response is stored in state as `bucket` and `appdata`.

## Role resource

```hcl
resource "aidial_role" "example" {
  bucket = data.aidial_bucket.current.bucket
  path   = "my-role"
  name   = "roles/${data.aidial_bucket.current.bucket}/my-role"

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

`name` is the value you configure and is never rewritten by the provider, even if the API returns a canonicalized name (for example DIAL Core may strip a `roles/<bucket>/` prefix from the value it echoes back). The API's own value is exposed separately as the computed `api_name` attribute. `description`, `display_name`, and `topics` are optional/computed and are preserved across updates when left unset in config, so applying changes to an existing role does not clear metadata Terraform isn't managing. `created_at`, `updated_at`, and `granted_keys` are computed/read-only.

## Key resource

```hcl
resource "aidial_key" "example" {
  bucket = data.aidial_bucket.current.bucket
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
  bucket       = data.aidial_bucket.current.bucket
  path         = "my-model"
  name         = "my-model"
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

`api_name` is the API's canonical name, analogous to `aidial_role`. `type` is one of `CHAT`, `COMPLETION`, `EMBEDDING`. See `openspec/model-schema` for the full attribute list and the known gaps versus the DIAL Core `Model` schema (for example `pricing.cache_read`/`cache_write` are modeled as plain strings rather than `PricingRate` objects, and a handful of `Model` fields such as `responses_defaults` and per-interface settings are not yet exposed).

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
  bucket = data.aidial_bucket.current.bucket
  path   = "my-route"
  name   = "my-route"

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

## Role data source

```hcl
data "aidial_role" "existing" {
  bucket = data.aidial_bucket.current.bucket
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

All four resources support import using `<bucket>/<path>`:

```bash
terraform import aidial_role.example '<bucket>/<path>'
terraform import aidial_key.example '<bucket>/<path>'
terraform import aidial_model.example '<bucket>/<path>'
terraform import aidial_route.example '<bucket>/<path>'
```

Import only sets `id`, `bucket`, and `path`; every other attribute, including `name`, is populated by the subsequent refresh.

### Worked example: importing an existing model

`name` and `type` are `Required` (not `Computed`), so Terraform still needs *some* value for them in your config before it will let you write a resource block at all — even though import itself doesn't read your config values for them. The practical flow is: write a placeholder block, import, inspect what actually came back, then fix the config to match.

1. Write a minimal block for the model you want to bring under management, using whatever bucket/path it actually lives at remotely. The `name`/`type` values here are just placeholders to satisfy the schema — they'll be corrected in step 3:

   ```hcl
   resource "aidial_model" "gpt5" {
     bucket = "platform"
     path   = "gpt-5-2025-08-07"
     name   = "placeholder"
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

   Update `name` (and `type`, `endpoint`, etc. if they differ from your placeholders) in the `.tf` file to match what `state show` printed.
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
  name   = "placeholder"
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

Every request uses the `Api-Key` header. `GET` returns `404` for a missing resource (treated as absent, not an error); `DELETE` treats `404` as a successful no-op; `PUT` on save returns only `{"name": ...}`, so every create/update re-reads the resource via `GET` to obtain the authoritative state. DIAL Core also supports `If-Match`/`If-None-Match`/`ETag`/`304`/`412` optimistic-concurrency headers on all three mutating operations for models and routes; the provider does not currently send or interpret them (see `openspec/model-client` and `openspec/route-client`).

## Repository layout

### Why this split

The provider code, its behavior specs, the local DIAL Core stack and the integration manifests are kept in separate top-level directories on purpose, so each has a single source of truth and a change touches one place:

- Resource behavior is specified once under `openspec/` and implemented once under `internal/`. When a resource's behavior changes, update the spec and the code together; don't describe behavior only in a README or a commit message. System-wide rules that apply to every resource (for example bucket/path defaults) live in their own spec (`openspec/resource-identity-defaults`), not copied into each per-resource spec.
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
│                               # (e.g. model-schema, role-client, resource-identity-defaults).
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
make openapi TAG=0.49.3
```

The target fails without changing anything if the tag or file does not exist. After a refresh, diff the contract and update any notes in `openspec/*-schema` and `openspec/*-client` that cite it. CI uses the checked-in copy and never fetches it.

## Devcontainer

`.devcontainer/` gives a ready-to-use development environment (VS Code "Reopen in Container", or any devcontainer-compatible tool). A fresh container provides:

| Tool | Purpose |
|------|---------|
| `go`, `gopls`, `goimports` | build, language server, import formatting |
| `dlv` | debugger (used by the Go extension) |
| `golangci-lint`, `tfproviderlint` | linting (`.golangci.yml`) |
| `markdownlint-cli2` | Markdown linting (`.markdownlint-cli2.jsonc`; excludes `openspec/`) |
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
   scripts/core.sh up                          # CORE_VERSION=0.48.1 scripts/core.sh up to pin a version (default: latest)
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
3. Add the repository secrets `GPG_PRIVATE_KEY` (`gpg --armor --export-secret-keys <id>`) and `PASSPHRASE`.
4. Publish the provider once from the Registry UI (Publish → Provider, GitHub sign-in).

For each release (maintainers):

1. Move the `CHANGELOG.md` entry from Unreleased to the version.
2. Tag and push: `git tag v0.1.0 && git push origin v0.1.0`. Tags must match `vX.Y.Z`.
3. Confirm the GitHub release lists the zips, `_SHA256SUMS`, `_SHA256SUMS.sig` and `_manifest.json`; the Registry picks the version up from there.

Test a release locally without publishing or signing:

```bash
goreleaser release --snapshot --clean --skip=sign
```
