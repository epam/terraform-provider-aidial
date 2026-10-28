# Changelog

All notable changes to this project are documented here. The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow [Semantic Versioning](https://semver.org/).

## Unreleased

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
