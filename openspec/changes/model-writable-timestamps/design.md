# Design

## Context

Both attributes already exist on `aidial_model` as computed `Int64` and are read from `ModelRequest`/model JSON (`createdAt`, `updatedAt` as `*int64`). `created_at` carries `UseStateForUnknown`; `updated_at` has no modifier. Core 0.50.2 stores both verbatim and never fills them in, verified by PUT/GET probes: values sent come back unchanged, and a PUT that omits them returns `null`. Core replaces the whole document on PUT, which is why an omitted field is cleared.

## Goals / Non-Goals

**Goals:**
- Let configuration state both timestamps and have them round-trip without plan churn.

**Non-Goals:**
- Auto-generating timestamps, or ordering checks between the two values.
- Changing data sources or other resources.

## Decisions

- **Optional only, not Optional+Computed.** Core never generates the values, so there is nothing for the provider to compute. Optional+Computed would force the provider to re-send prior state on every PUT to avoid an "inconsistent result after apply" error, for no benefit. Optional-only also makes drift visible and correctable.
- **Reuse the existing `Int64` type and `int64Pointer`/`int64Value` helpers.** `int64Pointer` maps null to a nil pointer, which `omitempty` drops from the body. Read mapping is unchanged.
- **Drop `UseStateForUnknown` from `created_at`.** It only applies to computed attributes, so it has no purpose once `created_at` is optional-only.
- **`int64validator.AtLeast(0)`** on both. A negative epoch is never meaningful. No upper bound.
- **Specs.** A new `model-timestamps` capability holds the behavior. The legacy `openspec/model-schema/spec.md` line stating the attributes are computed is edited in place during apply, because deltas cannot target it.

## Risks / Trade-offs

- [Existing models with externally set timestamps plan a clear-to-null on the next apply] → documented in attribute descriptions, docs and CHANGELOG; users who want to keep the value set it in configuration.
- [Provider-created models show no change] → Core returns `null` for them today, so state already holds null.
- [Another Core version may start auto-populating `updatedAt`] → the attribute would then show drift on every external change; revisit by making it Optional+Computed if that happens.
