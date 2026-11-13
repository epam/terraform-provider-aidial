# Design

**Object per rate, one level deep.** DIAL Core's `PricingRate` nests recursively, which a Terraform schema cannot express. The schema models `rate` or `test` plus flat `if_true` and `if_false`, which covers tiered pricing. Deeper nesting is rejected on read rather than silently lost.

**Wire format.** Core accepts the legacy plain-string form for a flat rate, so it is kept for writes, and branches use it too. Verified against Core 0.50.1 by `TestAccModel_conditionalPricing` (create, import, update, empty plan after apply). Reads accept a string or an object, and a number for `rate` or `test.value`.

**Condition value.** `value` is a string in Terraform. It is sent as a JSON number when it matches the JSON number grammar, because Core compares it with token counts, and otherwise as a string. The text is preserved on read.

**Breaking change.** Replacing the string attributes is deliberate, to keep one shape. Existing configurations must be rewritten (`prompt = { rate = "..." }`); this is stated in the CHANGELOG.
