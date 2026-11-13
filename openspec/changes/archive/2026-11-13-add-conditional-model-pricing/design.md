# Design

**Object per rate, dynamic branches.** DIAL Core's `PricingRate` nests recursively, which a typed Terraform schema cannot express. The schema models `rate` or `test` with `if_true` and `if_false`, and types the two branches dynamically: a price string, or an object with the same attributes, to any depth. Plan-time validation checks the shape of nested objects. On read, the prior value is kept when it converts to the same rate as Core's answer, so `"1"` versus `{ rate = "1" }` does not cause drift; after import the canonical form is used.

**Wire format.** Core accepts the legacy plain-string form for a flat rate, so it is kept for writes, and branches use it too. Verified against Core 0.50.1 by `TestAccModel_conditionalPricing` (create, import, update, empty plan after apply). Reads accept a string or an object, and a number for `rate` or `test.value`.

**Condition value.** `value` is a string in Terraform. It is sent as a JSON number when it matches the JSON number grammar, because Core compares it with token counts, and otherwise as a string. The text is preserved on read.

**Breaking change.** Replacing the string attributes is deliberate, to keep one shape. Existing configurations must be rewritten (`prompt = { rate = "..." }`); this is stated in the CHANGELOG.
