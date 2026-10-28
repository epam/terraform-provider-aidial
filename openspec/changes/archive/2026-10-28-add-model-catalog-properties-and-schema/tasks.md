## 1. Client

- [x] 1.1 Add `CatalogSchemaID *string` (`catalogSchemaId`) and `CatalogProperties json.RawMessage` (`catalogProperties`) to `client.Model` in `internal/client/client.go`
- [x] 1.2 Add catalog schema URL builder and `GetCatalogSchema`/`SaveCatalogSchema`/`DeleteCatalogSchema`, stripping `name`/`status` on GET
- [x] 1.3 Client tests in `internal/client` (404 handling, metadata stripping, delete 404, API error)

## 2. Model attributes

- [x] 2.1 Generalise the `defaults` JSON helpers in `internal/provider/model_defaults.go` for reuse by `catalog_properties`
- [x] 2.2 Add `catalog_schema_id` and `catalog_properties` to the schema, `modelFromPlan` and `setModelState` in `model_resource.go`
- [x] 2.3 Expose both on `aidial_file_model` (`file_model_data_source.go`)
- [x] 2.4 Tests: payload mapping, non-string values, null/empty/`{}` handling, equivalent-JSON preservation, extend `model_null_vs_empty_test.go`

## 3. Catalog schema resource and data source

- [x] 3.1 Implement `catalog_schema_resource.go` (schema, plan-time `$id` validation, CRUD, import, semantic-JSON state preservation) and register it in `provider.go`
- [x] 3.2 Implement `catalog_schema_data_source.go` and register it
- [x] 3.3 Unit tests mirroring `key_resource_test.go`/`role_resource_test.go`

## 4. Integration and docs

- [x] 4.1 Ran the new `tests/integration/tf_aidial_integration_catalog_schema.tf` manifest plus a temporary probe against ephemeral Core in CI; findings recorded in design.md ("Verified Against Core")
- [x] 4.2 Add `examples/` snippets, run `make docs`, commit `docs/`
- [x] 4.3 Update legacy specs by hand: add `catalog-schema-*` notes and remove the "not implemented" mentions of `catalog_schema_id`/`catalog_properties` in `openspec/model-schema/spec.md`; update `openspec/file-model-schema/spec.md`, and the path-default exception in `openspec/resource-identity-defaults`
- [x] 4.4 Run `go build`, `go test ./... -race`, `go vet`, `gofmt`, `make lint`, `terraform fmt -check -recursive`
