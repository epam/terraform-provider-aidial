# role-client Specification

## Purpose

Define the DIAL Core Role API client support in `internal/client/client.go` used by the `aidial_role` Terraform resource.

## Requirements

### Requirement: Role API client methods

The provider SHALL implement role client support in `internal/client/client.go`.

It SHALL provide:

- Role URL builder for resource endpoint (`/v1/roles/{bucket}/{path}`)
- `GetRole(ctx, bucket, path)`
- `SaveRole(ctx, bucket, path, role)`
- `DeleteRole(ctx, bucket, path)`

Client behavior SHALL follow existing conventions:

- `GetRole` returns `(nil, nil)` on HTTP `404`
- `DeleteRole` treats HTTP `404` as success
- Non-2xx responses are returned as API errors with status code and response body
- Request headers use the same `Api-Key` and JSON conventions as other resource operations

#### Scenario: Successful read
- **WHEN** `GetRole` receives `200 OK` with a valid role payload
- **THEN** it returns a parsed role object

#### Scenario: Role not found
- **WHEN** `GetRole` receives `404 Not Found`
- **THEN** it returns `nil, nil`

#### Scenario: Successful save
- **WHEN** `SaveRole` receives a 2xx response
- **THEN** it returns success

#### Scenario: Idempotent delete
- **WHEN** `DeleteRole` receives `404 Not Found`
- **THEN** it returns success

#### Scenario: Upstream error
- **WHEN** role operations receive non-2xx (excluding handled 404 cases)
- **THEN** client returns a formatted API error

### Requirement: Tolerant cost limit decoding

`client.CostLimit`'s `day`/`minute`/`month`/`week` fields SHALL decode successfully whether the API encodes each value as a bare JSON number or as a quoted numeric string.

#### Scenario: Cost limit encoded as a number
- **WHEN** `GetRole` receives `costLimit.day` as a bare JSON number (e.g. `1`)
- **THEN** it decodes into `cost_limit.day` without error

#### Scenario: Cost limit encoded as a quoted sentinel string
- **WHEN** `GetRole` receives `costLimit.day` as a quoted numeric string (e.g. `"9223372036854775807"`, DIAL Core's observed "unlimited" sentinel)
- **THEN** it decodes into `cost_limit.day` without error, rather than failing with a JSON type-mismatch error

## Implementation Notes

- API methods live in `client.go`, consistent with the model/route client's request/response conventions (log request/response, treat 404 specially on GET/DELETE, format non-2xx as `API error <status>: <body>` on save).
- `CostLimit.Day`/`Minute`/`Month`/`Week` are typed as `*client.FlexibleFloat64` rather than `*float64`. `FlexibleFloat64` implements a custom `UnmarshalJSON` that accepts either representation; this exists because a live DIAL Core deployment was observed returning cost-limit sentinel values (`9223372036854775807`, i.e. int64 max) as quoted JSON strings — likely to avoid float64 precision loss on the server's own encoding — which a plain `*float64` field cannot decode (`json: cannot unmarshal string into Go struct field CostLimit.day of type float64`), causing every `GetRole`/`Read`/post-create refresh to fail for any role with such a limit.
- This is not reflected in the checked-in OpenAPI spec, which types `CostLimit.day`/`minute`/`month`/`week` as plain `number` (`openapicore/open_api_core.yaml`, `CostLimit` schema) with no indication of a string-encoded variant.
- `SaveRole` marshals `*FlexibleFloat64` using the default numeric `MarshalJSON`, i.e. it always sends a bare JSON number on write; only decoding is tolerant of the quoted-string variant.
