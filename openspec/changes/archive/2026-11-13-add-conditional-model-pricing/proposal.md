# Proposal

## Why

`aidial_model.pricing.prompt`, `completion`, `cache_read` and `cache_write` were plain strings, but DIAL Core's `PricingRate` also supports conditional prices (`test`, `ifTrue`, `ifFalse`). A configuration with a tiered price failed at plan time with "Incorrect attribute value type".

## What Changes

- **BREAKING:** each pricing rate becomes a nested object: `rate`, or `test` (`field`, `operator`, `value`) with `if_true` and `if_false`. String prices are no longer accepted.
- Plan-time validation: `rate` conflicts with the conditional attributes, the conditional attributes require each other, and `operator` is one of `==`, `!=`, `>`, `<`, `>=`, `<=`.
- The client reads a rate from either a JSON string or an object. A flat rate is written as a plain string, conditional branches are plain strings, and a numeric `test.value` is written as a JSON number.
- Terraform cannot nest recursively, so `if_true` and `if_false` are dynamically typed: a flat price string, or an object with the same attributes as the rate, to any depth. State keeps the configured form of a branch.
- The `aidial_model` and `aidial_file_model` data sources expose the same shape. Examples, stub models, the integration manifest, docs, README and CHANGELOG are updated.

## Capabilities

### Modified Capabilities

- `model-schema`: the `pricing` attribute and its scenarios.
