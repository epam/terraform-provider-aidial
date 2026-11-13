---
page_title: "Resource behavior notes"
subcategory: ""
description: |-
  Behavior that the generated resource pages do not spell out: identity rules, server-filled defaults, fields the provider does not manage, and import.
---

# Resource behavior notes

The generated pages describe every attribute. This guide collects behavior that applies across resources or that surprises people.

## Identity (`bucket` and `path`)

All eleven resources, and the `aidial_role`, `aidial_model`, `aidial_catalog_schema`, `aidial_application_type_schema`, `aidial_toolset`, `aidial_interceptor`, `aidial_translator`, `aidial_application` and `aidial_alias` data sources, share the same identity model: `bucket` and `path` are required, with no defaults. `path` must be non-empty and may contain `/` (except for interceptors, translators and aliases, where DIAL Core rejects it, and for applications outside the `public` bucket). Omitting either fails validation at plan time. Both force replacement on change, and `id` is persisted as `<bucket>/<path>`. `bucket` must be `platform` for every type except toolsets, applications and aliases (DIAL Core stores those resources only there; the bucket returned by `aidial_bucket` is not accepted). Toolsets and applications accept any non-empty bucket, for example `public`, and leave it to DIAL Core to reject the ones it does not allow. Aliases accept `public` and `platform`.

## aidial_bucket

`aidial_bucket` takes no arguments — it always resolves to the bucket owned by the configured `api_key`.

The API response is stored in state as `bucket` and `appdata`.

## aidial_role

`limits` and `share` are maps whose values are JSON strings. Use `jsonencode(...)` so Terraform serializes nested objects safely; the provider decodes those strings only far enough to canonicalize them and stores them back as JSON strings, so `limits` values are not required to match any particular shape (DIAL Core's documented `Limit` object, or ad-hoc shapes such as `{"enabled": true}`, both work).

`name` is optional: DIAL Core addresses the role by `bucket` and `path` and derives its name from `path`, so you can leave `name` out and it is read back from the API. When you do configure it, the value is sent and never rewritten by the provider, even if the API returns a canonicalized name (for example DIAL Core may strip a `roles/<bucket>/` prefix from the value it echoes back). The API's own value is exposed separately as the computed `api_name` attribute. `description`, `display_name`, and `topics` are optional/computed and are preserved across updates when left unset in config, so applying changes to an existing role does not clear metadata Terraform isn't managing. `created_at`, `updated_at`, and `granted_keys` are computed/read-only.

## aidial_model

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

## aidial_route

A route may instead configure `response` (a static `body`/`status` returned without forwarding upstream) in place of `upstreams`. See `openspec/route-schema` for the full attribute list.

## aidial_toolset

Attribute names are snake_case in Terraform (and in DIAL Core's toolset JSON). `provider_name` is the API's `provider`, which Terraform reserves as an attribute name. A few notes:

- DIAL Core fills in values you leave out: `reference` (a generated identifier), `transport` (`HTTP`), `forward_auth_token`, `forward_per_request_key` (`false`), `max_retry_attempts` (`1`), empty `allowed_tools`, `dependencies`, `description_keywords` and `interceptors`, and `auth_settings` of type `NONE`. They are read back into state, so deleting such a line from the configuration does not reset it; set it explicitly instead. `author` is set by DIAL Core from the caller (for toolsets outside the `platform` bucket).
- `auth_settings.client_secret` is sensitive and never returned by DIAL Core. State keeps the configured value, and it is null after import. DIAL Core derives `code_challenge` itself, and the sign-in status fields and `code_verifier` are session state, so none of them are attributes. For OAuth, set the full block, as in `examples/resources/aidial_toolset`: DIAL Core probes the endpoints when it saves OAuth settings in the `platform` bucket, or incomplete OAuth settings in other buckets.
- `features` and `interfaces` have the same shape as on `aidial_model`. `defaults`, `responses_defaults` and `catalog_properties` are free-form objects.
- `display_name` and `intro` are plain strings. The locale-map form DIAL Core allows is not supported, and neither are `created_at`/`updated_at`.

`aidial_toolset` (data source) reads one by `bucket` and `path`, and `aidial_file_toolset` reads a toolset from DIAL Core's static configuration by `name`. Neither exposes `client_secret`. See `openspec/specs/toolset-resource` for the full behavior.

## aidial_interceptor

`aidial_interceptor` registers an interceptor in DIAL Core. Models list interceptor names in their `interceptors` attribute; the provider does not link the two resources or check that the names exist, so see the DIAL Core documentation for how a name resolves to an interceptor. A few notes:

- `endpoint` is required. `forward_auth_token` (`false`) and `max_retry_attempts` (`1`) are filled in by DIAL Core when you leave them out, so they are read back into state.
- `features` and `interfaces` have the same shape as on `aidial_model`. `defaults` and `catalog_properties` are free-form objects.
- Only the attributes above and in the generated docs are managed. DIAL Core replaces the whole interceptor on every save, so a value set out of band in a field the provider does not expose (for example `dependencies`, `description_keywords`, `responses_endpoint`, `input_attachment_types` or `override_name`) is removed by the next apply. `author`, `created_at` and `updated_at` are not attributes.
- DIAL Core accepts only the `platform` bucket for interceptors, and rejects `/` in the `path`.

`aidial_interceptor` (data source) reads one by `bucket` and `path`, and `aidial_file_interceptor` reads an interceptor from DIAL Core's static configuration by `name`. See `openspec/specs/interceptor-resource` for the full behavior.

## aidial_translator

`aidial_translator` registers a translator in DIAL Core: a service that converts requests from the `in` interface type to the `out` interface type. A few notes:

- `base_url`, `in` and `out` are all required, as DIAL Core rejects a translator without them. `in` and `out` accept `openaiChatCompletions`, `openaiEmbeddings`, `openaiResponses` and `anthropicMessages`. `in` is a keyword in HCL expressions but is a valid attribute name.
- The provider does not link translators to the `translator` entry of a model or toolset interface, which is still not modeled.
- DIAL Core replaces the whole translator on every save, so a value set out of band in a field the provider does not expose is removed by the next apply.
- DIAL Core accepts only the `platform` bucket for translators, and rejects `/` in the `path`.

`aidial_translator` (data source) reads one by `bucket` and `path`, and `aidial_file_translator` reads a translator from DIAL Core's static configuration by `name`. See `openspec/specs/translator-resource` for the full behavior.

## aidial_alias

`aidial_alias` creates an alternative invocation name for a model, application or toolset. A few notes:

- `present_as` (`model`, `application` or `toolset`) and `target_path` (`models/<name>`, `applications/<name>` or `toolsets/<name>`) are required. An alias shown as a toolset must point at a toolset, and one shown as a model or application cannot; the provider checks that at plan time. Core does not check that the target exists.
- The other attributes are `display_name`, `description`, `display_version`, `icon_url`, `intro`, `user_roles` and `description_keywords`. DIAL Core fills in nothing but the name.
- DIAL Core accepts the `public` and `platform` buckets for aliases (the OpenAPI document says `public` only), and rejects `/` in the `path`.
- Core accepts only `path` in a target, so there is no `target.id` or `target.version`, and it rejects `endpoint`, `features`, `catalog_properties` and `catalog_schema_id` on an alias. DIAL Core replaces the whole alias on every save, so a value set out of band in a field the provider does not expose (`author`, `created_at`, `updated_at`) is removed by the next apply. Core accepts a locale map for `display_name` but returns a plain string, so only the plain form is supported.

`aidial_alias` (data source) reads one by `bucket` and `path`, and `aidial_file_alias` reads an alias from DIAL Core's static configuration by `name`. See `openspec/specs/alias-resource` for the full behavior.

## aidial_application

`aidial_application` registers an application in DIAL Core. A few notes:

- Set exactly one of `endpoint` (endpoint-based) and `application_type_schema_id` (schema-based, with its values in `application_properties`). Core rejects both together and rejects neither, so the provider fails at plan time. `application_type_schema_id` takes the `schema_id` of an `aidial_application_type_schema`.
- `forward_auth_token` (`false`), `max_retry_attempts` (`1`) and `interceptors` (empty) are filled in by DIAL Core when you leave them out, so they are read back into state. `interceptors` lists interceptor names; the provider does not check that they exist.
- `features` and `interfaces` have the same shape as on `aidial_model`. `defaults`, `application_properties` and `catalog_properties` are free-form objects.
- Only the attributes above and in the generated docs are managed. DIAL Core replaces the whole application on every save, so code-based applications (`function`) and values set out of band in `mcp`, `routes`, `external_services` or other fields the provider does not expose are removed by the next apply. `author`, `created_at` and `updated_at` are not attributes.
- DIAL Core accepts the `platform` and `public` buckets. A path with `/` works only in `public`. For a `public` application Core returns `name` as `applications/public/<path>`.

`aidial_application` (data source) reads one by `bucket` and `path`, and `aidial_file_application` reads an application from DIAL Core's static configuration by `name`. See `openspec/specs/application-resource` for the full behavior.

## aidial_file_model and the other file data sources

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
