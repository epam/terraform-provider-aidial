# file-model-schema Specification

## Purpose

Define the `aidial_file_model` data source, implemented in `file_model_data_source.go`, which reads a "file-sourced" model: one defined in DIAL Core's static configuration rather than created via the `aidial_model` resource.

## Requirements

### Requirement: Two distinct model storage backends

DIAL Core exposes models through two separate, non-interchangeable storage backends, distinguished in the OpenAPI spec:

- **Resource-storage models**: created dynamically via `PUT /v1/models/{bucket}/{path}`. Managed by the `aidial_model` resource.
- **File-sourced models**: defined in DIAL Core's static configuration at startup, exposed read-only via `GET /v1/admin/config/file/models/{name}` (`operationId: getFileConfigModel`). No bucket/path; keyed only by `name`. No `PUT`/`DELETE` exists for this backend anywhere in the API.

A model existing in `GET /openai/models/{model_name}` (the universal runtime discovery endpoint used by chat clients) does **not** indicate which backend defines it — that endpoint returns a different response shape (`ModelData`) entirely unrelated to either storage backend's `Model` schema, and succeeds for a model regardless of which backend created it.

#### Scenario: aidial_model cannot manage file-sourced models
- **WHEN** a user attempts `terraform import aidial_model.x <bucket>/<path>` for a model that is actually file-sourced
- **THEN** `GET /v1/models/{bucket}/{path}` returns `404` (or a non-JSON error page) for every bucket/path guess, because no such bucket-scoped resource exists — this is expected behavior, not a bug, since the model isn't stored that way

### Requirement: aidial_file_model data source

The provider SHALL expose a read-only data source named `<provider>_file_model` (e.g. `aidial_file_model`) implemented in `file_model_data_source.go`.

The schema SHALL include:

- `name` (required, the file-sourced model name to look up)
- `api_name` (computed, canonical name returned by the API)
- every other `aidial_model` computed attribute (`type`, `display_name`, `description`, `endpoint`, `user_roles`, `responses_endpoint`, `display_version`, `icon_url`, `intro`, `reference`, `forward_auth_token`, `max_retry_attempts`, `author`, `status`, `features`, `limits`, `pricing`, `upstreams`, `dependencies`, `description_keywords`, `input_attachment_types`, `interceptors`, `fields_hashing_order`, `override_name`, `interfaces`, `default_headers`, `defaults`), all `Computed`-only since this data source has no write path

There is no `id`, `bucket`, or `path` attribute — file-sourced models are not bucket-scoped.

#### Scenario: Successful lookup
- **WHEN** `name` matches an existing file-sourced model
- **THEN** the data source populates all computed attributes from the API response

#### Scenario: Model not found
- **WHEN** `name` does not match any file-sourced model
- **THEN** the data source returns a diagnostic error naming the lookup that failed, distinguishing this backend from `aidial_model`'s resource-storage models

#### Scenario: API canonicalizes the looked-up name
- **WHEN** the API returns a `name` value that differs from the configured lookup `name`
- **THEN** the data source does not attempt to reconcile them as a diff (data sources have no prior state to conflict with) — the configured `name` is echoed back unchanged, and the API's own value is exposed via `api_name`

## Implementation Notes

- `Read` reuses `setModelState` (the same field-mapping/object-construction logic `aidial_model` uses) via a throwaway `modelResourceModel`, then copies the relevant fields into `fileModelDataSourceModel`. This avoids duplicating the `features`/`limits`/`pricing`/`upstreams` nested-object construction logic, and keeps both `aidial_model` and `aidial_file_model` decoding a `client.Model` identically.
- `client.GetFileModel(ctx, name)` hits `GET /v1/admin/config/file/models/{name}` via the shared generic `getResource[Model]` helper, so it inherits the same 404-is-nil semantics as `GetModel`.
- Known gap: no data source or resource yet exposes `GET /v1/admin/config/file/models` (the list endpoint, `operationId: listFileConfigModels`), which returns just `{items: [{name}]}` — enumerating available file-sourced model names by name only, without their full config. Not implemented since the immediate need was a single-name lookup.
