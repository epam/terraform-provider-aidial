# Design

## Context

`aidial_file_model` reads static-config models through `GET /v1/admin/config/file/models/{name}`. API-managed models live at `/v1/models/{bucket}/{path}` and are written by the `aidial_model` resource. `client.GetModel` already returns `nil, nil` on 404, and `setModelState` (model_resource.go) maps a `client.Model` to state, including null-vs-empty normalization and the catalog attributes. `aidial_file_model` already calls it through a throwaway `modelResourceModel`. See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- A read-only `aidial_model` data source with the same identity defaults as the resource.
- Attribute values identical to the resource's state for the same model.

**Non-Goals:**
- Merging with `aidial_file_model` or changing it.
- Extracting a shared schema or field-copy helper between the two model data sources.
- Listing or filtering models.

## Decisions

**Separate data source, not an extension of `aidial_file_model`.** The two read different API paths with different identity (`name` only vs `bucket`/`path`). Mixing them would blur `id` semantics and force a fallback rule. Alternative considered: optional `bucket`/`path` on `aidial_file_model` that switch the lookup. Rejected for that reason.

**Lookup arguments mirror the resource.** `name` required, `bucket` optional defaulting to `platform`, `path` optional defaulting to `name`. Unlike `aidial_role` and `aidial_bucket` (required `bucket`/`path`), this lets the same values used in the resource be reused as-is. Defaults are applied in `Read`, since data source schemas do not support plan modifiers; `bucket` and `path` stay Optional and are written back resolved into state so `id` is consistent. Because the framework rejects setting a non-computed Optional attribute to a value the config did not give, `bucket` and `path` are declared `Optional` and `Computed`.

**Copy the `aidial_file_model` schema.** The computed attribute list is copied into the new data source, with `bucket`, `path` and `id` added. Matches current style and keeps `aidial_file_model` untouched. Cost: a third copy of the attribute list (resource, file data source, new data source); a shared helper is a possible follow-up.

**Reuse `modelResourceModel` and `setModelState` directly.** The data source's attribute set (`id`, `bucket`, `path`, `name` plus the file-model attributes) is exactly the resource model's, so Read decodes config into a `modelResourceModel`, calls `setModelState`, and sets state from the same struct. No field-by-field copy is needed, unlike `aidial_file_model`, and parity with the resource holds by construction. This includes the resource's mapping of omitted `interfaces` and `default_headers` to empty maps.

**Not found is an error.** `GetModel` returning `nil` yields an error diagnostic naming `<bucket>/<path>`. The resource's "404 means absent" rule applies only to resource Read.

## Risks / Trade-offs

- [Attribute drift: a new model attribute is added to the resource but missed in a data source] → a unit test compares the data source schema's attributes against the file model's, and the acceptance test checks parity with the resource.
- [Three copies of the schema] → accepted for now; flagged as a follow-up refactor.
- [`defaults` is a dynamic attribute and has no prior value to preserve] → `setModelState` already handles a null prior; the data source returns the API JSON as-is.
