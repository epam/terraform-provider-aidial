# route-client Specification

## Purpose

Define the DIAL Core Route API client support in `internal/client/client.go` used by the `aidial_route` Terraform resource.

## Requirements

### Requirement: Route API client methods

The provider SHALL implement route client support in `internal/client/client.go`, consistent with existing role client patterns.

It SHALL provide:

- Route URL builder for resource endpoint (`/v1/routes/{bucket}/{path}`)
- `GetRoute(ctx, bucket, path)`
- `SaveRoute(ctx, bucket, path, route)`
- `DeleteRoute(ctx, bucket, path)`

Client behavior SHALL follow existing conventions:

- `GetRoute` returns `(nil, nil)` on HTTP `404`
- `DeleteRoute` treats HTTP `404` as success
- Non-2xx responses are returned as API errors with status code and response body
- Request headers use the same `Api-Key` and JSON conventions as role operations

#### Scenario: Successful read
- **WHEN** `GetRoute` receives `200 OK` with a valid route payload
- **THEN** it returns a parsed route object

#### Scenario: Route not found
- **WHEN** `GetRoute` receives `404 Not Found`
- **THEN** it returns `nil, nil`

#### Scenario: Successful save
- **WHEN** `SaveRoute` receives a 2xx response
- **THEN** it returns success

#### Scenario: Idempotent delete
- **WHEN** `DeleteRoute` receives `404 Not Found`
- **THEN** it returns success

#### Scenario: Upstream error
- **WHEN** route operations receive non-2xx (excluding handled 404 cases)
- **THEN** client returns a formatted API error

### Requirement: DIAL Core response contract

`SaveRoute` SHALL treat a successful `PUT` response body as containing only `{"name": ...}` (DIAL Core `ConfigResourceControllerConfigWriteResponse`), not the full route, and rely on a subsequent `GetRoute` call to obtain the authoritative saved representation.

`DeleteRoute` SHALL treat DIAL Core's documented success status for delete as `204 No Content`.

#### Scenario: Save response does not contain full route
- **WHEN** `SaveRoute` receives a 2xx response containing only a `name` field
- **THEN** the client does not attempt to parse a full route from the save response

#### Scenario: Delete success status
- **WHEN** `DeleteRoute` receives `204 No Content`
- **THEN** it returns success

## Implementation Notes

- API methods live in `client.go`, mirroring the role client's request/response conventions.
- Verified against the DIAL Core OpenAPI spec (`openapicore/open_api_core.yaml`, `/v1/routes/{bucket}/{path}` path). DIAL Core additionally supports optimistic-concurrency headers on all three operations (`If-Match`/`If-None-Match` request headers, an `ETag` response header on GET/PUT, and `304`/`412` responses), structurally identical to the `/v1/models/{bucket}/{path}` contract. This client does not currently send or interpret ETags — same gap as the existing model and role clients — and is tracked as a known limitation rather than a met requirement.
