# model-client Specification

## Purpose

Define the DIAL Core Model API client support in `internal/client/client.go` used by the `aidial_model` Terraform resource.

## Requirements

### Requirement: Model API client methods

The provider SHALL implement model client support in `internal/client/client.go`, consistent with existing role client patterns.

It SHALL provide:

- Model URL builder for resource endpoint (`/v1/models/{bucket}/{path}`)
- `GetModel(ctx, bucket, path)`
- `SaveModel(ctx, bucket, path, model)`
- `DeleteModel(ctx, bucket, path)`

Client behavior SHALL follow existing conventions:

- `GetModel` returns `(nil, nil)` on HTTP `404`
- `DeleteModel` treats HTTP `404` as success
- Non-2xx responses are returned as API errors with status code and response body
- Request headers use the same `Api-Key` and JSON conventions as role operations

#### Scenario: Successful read
- **WHEN** `GetModel` receives `200 OK` with a valid model payload
- **THEN** it returns a parsed model object

#### Scenario: Model not found
- **WHEN** `GetModel` receives `404 Not Found`
- **THEN** it returns `nil, nil`

#### Scenario: Successful save
- **WHEN** `SaveModel` receives a 2xx response
- **THEN** it returns success

#### Scenario: Idempotent delete
- **WHEN** `DeleteModel` receives `404 Not Found`
- **THEN** it returns success

#### Scenario: Upstream error
- **WHEN** model operations receive non-2xx (excluding handled 404 cases)
- **THEN** client returns a formatted API error

### Requirement: DIAL Core response contract

`SaveModel` SHALL treat a successful `PUT` response body as containing only `{"name": ...}` (DIAL Core `ConfigResourceControllerConfigWriteResponse`), not the full model, and rely on a subsequent `GetModel` call to obtain the authoritative saved representation.

`DeleteModel` SHALL treat DIAL Core's documented success status for delete as `204 No Content`.

#### Scenario: Save response does not contain full model
- **WHEN** `SaveModel` receives a 2xx response containing only a `name` field
- **THEN** the client does not attempt to parse a full model from the save response

#### Scenario: Delete success status
- **WHEN** `DeleteModel` receives `204 No Content`
- **THEN** it returns success

## Implementation Notes

- API methods live in `client.go`, mirroring the role client's request/response conventions.
- Verified against the DIAL Core OpenAPI spec (`openapicore/open_api_core.yaml`, `Models` tag). DIAL Core additionally supports optimistic-concurrency headers on all three operations (`If-Match`/`If-None-Match` request headers, an `ETag` response header on GET/PUT, and `304`/`412` responses). This client does not currently send or interpret ETags — same gap as the existing role client — and is tracked as a known limitation rather than a met requirement.
