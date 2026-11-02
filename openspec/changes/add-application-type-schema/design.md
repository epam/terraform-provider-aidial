# Design

## Context

`aidial_catalog_schema` (`catalog_schema_resource.go`, `catalog_schema_data_source.go`, `Client.*CatalogSchema`) already manages JSON Schema documents under `/v1/catalog_schemas/{bucket}/{path}`. Core 0.49.4 defines `ApplicationTypeSchema` at `/v1/schemas/{bucket}/{path}` with the same shape: an opaque object with a required non-blank `$id`, PUT returns only `{"name": ...}`, and GET merges `EntityMetadata` (`name`, `status`). Static-config schemas are read from `GET /v1/admin/config/file/schemas/{name}`, like `aidial_file_model` reads `/v1/admin/config/file/models/{name}`. See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- A resource, an API-managed data source and a file-config data source for application type schemas, behaving like their catalog schema and file model counterparts.

**Non-Goals:**
- Metaschema, list, or file-config catalog schema data sources.
- Changing the behavior of `aidial_catalog_schema`.
- Wiring schemas into other resources (the provider has no application resource).

## Decisions

**Mirror the catalog schema, as separate files.** New resource, data source and client methods sit next to the catalog ones. The resource attribute set is identical (`id`, `bucket`, `path`, `schema`, `schema_id`), so users learn one pattern.

**Extract only small pure helpers.** The JSON-object/`$id` parser and validator, the "keep prior text if semantically equal" state mapping and the `name`/`status` stripping are generic to both schema kinds. If moving them out of the catalog files is a trivial rename (unexport-free, no behavior change), share them; otherwise copy. Catalog schema behavior and its tests must not change. Alternative considered: a generic schema resource parameterised by endpoint. Rejected: it touches working, tested code and complicates error text, type names and docs.

**Strip metadata in the client.** `GetApplicationTypeSchema` decodes into `map[string]json.RawMessage`, deletes `name` and `status`, and re-marshals, as `GetCatalogSchema` does. `GetFileApplicationTypeSchema` does the same but also returns `status` (a string when present), because `aidial_file_model` exposes `status`.

**Identity from `resource-identity`.** `bucket` is `OneOf(platformBucket)`, `path` is `LengthAtLeast(1)`, both `RequiresReplace`, and `id` is `<bucket>/<path>`. The data source follows the same required arguments.

**File data source is name-only.** No `bucket`, `path` or `id`, matching `aidial_file_model`. Not found is an error diagnostic. Since `name` is the only key, no state ID attribute is added.

**No conditional-request headers.** ETag, `If-Match` and `If-None-Match` are not used, per the existing API semantics rule.

## Risks / Trade-offs

- [Helper extraction regresses catalog schema] -> keep extraction mechanical, run the existing catalog schema tests unchanged; fall back to copying if not trivial.
- [Core's schema endpoint behaves differently from catalog schemas, e.g. extra server-side validation on `$id`] -> the acceptance test and integration manifest run against a real Core; the spec treats PUT/GET as opaque.
- [`status` shape in the file-config response is not formally typed] -> `EntityMetadata.status` is a string; read it tolerantly and leave null when absent.
