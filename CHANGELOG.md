# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## Unreleased

## 0.2.5 - 2026-11-03

### Fixed

- Creating or reading an `aidial_model` or `aidial_route` no longer fails with `json: cannot unmarshal object into Go struct field ... authType of type string` when DIAL Core returns an upstream's `authType` as a JSON object instead of the documented string. An object carrying a string `name`, `type` or `value` is unwrapped, and any other object (including `{}`) is treated as unset, so `auth_type` stays null in state.

## 0.2.4 - 2026-11-03

This release brings `aidial_model` and `aidial_route` in line with the DIAL Core OpenAPI contract. Every new attribute is optional or read-only, so existing configurations plan without changes.

### Added

- `aidial_model` gains the model-level `base_url` (sent as `baseUrl`), which completes `interfaces` entries that declare no `base_url` of their own and on its own serves only the interface types you declare. It also gains `embedding_dimensions`, `responses_defaults` (a free-form object handled like `defaults`) and the read-only `created_at` and `updated_at` timestamps.
- Each `aidial_model` `interfaces` entry accepts `mode` (`passthrough` or `translator`), `override_paths` and `default_headers`, next to `base_url`.
- Each `upstreams` entry on `aidial_model` and `aidial_route` accepts `base_url` and `auth_type` (`API_KEY` or `BEARER`).
- `interfaces` keys are validated at plan time: only `openaiChatCompletions`, `openaiEmbeddings`, `openaiResponses` and `anthropicMessages` are accepted, so a typo or snake_case spelling fails early instead of reaching Core.
- The `aidial_model` and `aidial_file_model` data sources expose the new model attributes, including the new interface fields.

### Fixed

- Reading an `aidial_key` that has `allowedIpAddressRanges` set in DIAL Core no longer fails. Core returns it as an object, and the client expected a list of strings.

### Notes

- `aidial_key` does not expose `allowedIpAddressRanges`. With Core 0.49.3, a key created with ranges through the API cannot be reloaded by Core later: it reports the key as `invalid` and fails to delete it with HTTP 500.
- Not yet modeled: interface `translator`, `defaults` and `features`; per-upstream `interfaces`; localized `display_name` and `intro`; and `PricingRate` objects for `pricing`.

## 0.2.3 - 2026-11-02

This release adds application type schema support and makes the identity of every resource explicit. **Upgrade note:** `bucket` and `path` are now required on most resources and data sources, so configurations that omitted them fail validation at plan time until they are added. See Changed.

### Added

- Resource `aidial_application_type_schema` and data source `aidial_application_type_schema` manage and look up DIAL Core custom application type schemas (`/v1/schemas/{bucket}/{path}`). `schema` is a JSON object string with a non-blank `$id` (use `jsonencode`), `schema_id` exposes the `$id`, and the configured text is kept when Core returns semantically equal JSON.
- Data source `aidial_file_application_type_schema` reads an application type schema defined in DIAL Core's static configuration, by `name`, and exposes `schema`, `schema_id` and `status`.

### Changed

- **Breaking:** `bucket` and `path` are required, with no defaults, on the `aidial_model`, `aidial_role`, `aidial_route`, `aidial_key` and `aidial_catalog_schema` resources and on the `aidial_role`, `aidial_model` and `aidial_catalog_schema` data sources. `bucket` must be `platform` (DIAL Core rejects any other bucket, including the one `aidial_bucket` returns) and `path` must be non-empty. Both still force replacement, and `id` is `<bucket>/<path>`. The former `bucket` default and the `path` fallback to `name` are removed, so add `bucket = "platform"` and an explicit `path` to configurations that relied on them.
- `name` is optional and computed on `aidial_model`, `aidial_role` and `aidial_route`, and on the `aidial_model` data source. DIAL Core derives the name from the path and ignores it in the request body, so an unset `name` is no longer sent. A configured `name` is kept as written, and `api_name` holds the value Core returns.
- The README and examples no longer pass the `aidial_bucket` result as `bucket`, which Core rejects.
- The checked-in DIAL Core OpenAPI contract is refreshed to tag 0.49.4.

## 0.2.2 - 2026-10-28

Version 0.2.1 was published on GitHub but never reached the Terraform Registry, because its `SHA256SUMS` listed the SBOM files and the Registry rejects such releases. This release fixes that and includes everything from 0.2.1, so if you are upgrading from 0.2.0, read the security and behavior notes below.

### Security

- The `aidial_model` and `aidial_file_model` data sources now mark upstream `key` and `secret_extra_data` as sensitive, so Terraform redacts them in plan, apply and console output. A root module `output` that exposes them must now set `sensitive = true`.
- Releases are built with the newest Go 1.26 patch release instead of the runner image's cached one, which picks up standard library security fixes (GO-2026-6809 in `crypto/x509`, fixed in Go 1.26.10).

### Fixed

- Path parameters (`bucket`, `path` and the file model `name`) are now percent-encoded as a single URL path segment. Previously only `/` was encoded, and file model names were not encoded at all, so names containing `?`, `#`, `%` or spaces produced wrong requests. Names without reserved characters are unaffected.
- `aidial_key` stores null for `project` and `role` when DIAL Core omits them, instead of an empty string. An explicitly configured `""` is still kept, and existing state is unchanged.
- `SHA256SUMS` lists only the provider zips and the manifest again. SBOMs remain release assets and are covered by a build provenance attestation instead.

### Changed

- Documentation covers every resource and data source, requires Terraform 1.5 or later, and shows an `import` block example for each resource.
- The release pipeline publishes this changelog section as the release notes, pins GoReleaser and syft, and verifies the signature, checksums and attestation after publishing. It fails when `SHA256SUMS` lists anything other than zips and the manifest. See 0.2.1 below for details.
- GitHub Actions dependencies were updated (GoReleaser action 7.3.0, CodeQL 4.39.1, dependency review 5.0.0, zizmor 0.7.1).

## 0.2.1 - 2026-10-28

### Security

- The `aidial_model` and `aidial_file_model` data sources now mark upstream `key` and `secret_extra_data` as sensitive, so Terraform redacts them in plan, apply and console output. A root module `output` that exposes them must now set `sensitive = true`.

### Changed

- GitHub releases now use the matching `CHANGELOG.md` section as their description, and the release fails before publishing when it is missing. After publishing, the release workflow verifies the signature, checksums and build attestation of the published assets.
- The release workflow pins GoReleaser (2.18.3) and syft (1.54.1), is limited to 30 minutes and never cancels a running release. CI validates `.goreleaser.yml` with `goreleaser check`.
- The README and example provider block require Terraform 1.5 or later and `~> 0.2`, and each resource documents an `import` block example.

### Fixed

- Path parameters (`bucket`, `path` and the file model `name`) are now percent-encoded as a single URL path segment. Previously only `/` was encoded, and file model names were not encoded at all, so names containing `?`, `#`, `%` or spaces produced wrong requests. Names without reserved characters are unaffected.
- `aidial_key` stores null for `project` and `role` when DIAL Core omits them, instead of an empty string. An explicitly configured `""` is still kept, and existing state is unchanged.

## 0.2.0 - 2026-10-28

### Added

- Resource and data source `aidial_catalog_schema`.
- Data source `aidial_model`, which reads an API-managed model by `bucket`/`path`.
- `catalog_schema_id` and `catalog_properties` on `aidial_model` and the `aidial_file_model` data source.
- Local ephemeral DIAL Core for integration and acceptance tests (`make core-up`, `make test-integration`, `make testacc`).

## 0.1.0 - 2026-10-27

First public release, published to the Terraform Registry as `epam/aidial`.

### Added

- Resources `aidial_role`, `aidial_key`, `aidial_model` and `aidial_route`.
- Data sources `aidial_bucket`, `aidial_role` and `aidial_file_model`.
- Import support (`<bucket>/<path>`) for all resources.
- Signed multi-platform release artifacts and generated documentation.
