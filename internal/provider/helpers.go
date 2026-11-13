package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// platformBucket is the only bucket DIAL Core accepts for the resources this provider manages.
const platformBucket = "platform"

// dropServerDefaultLimits removes limit fields that DIAL Core filled in on its
// own. When a limit object omits some of minute/day/week/month/requestHour/
// requestDay, Core stores them as the int64-max "unlimited" sentinel and
// returns them on read, so {"day":10} comes back with five extra sentinel
// fields and Terraform reports an inconsistent result. A field is dropped only
// when it is absent from the prior (planned/state) value for that key and holds
// the sentinel; keys without a prior value (e.g. import) are returned as-is.
func dropServerDefaultLimits(api map[string]json.RawMessage, prior types.Map) map[string]json.RawMessage {
	if len(api) == 0 || prior.IsNull() || prior.IsUnknown() {
		return api
	}

	result := make(map[string]json.RawMessage, len(api))
	for key, raw := range api {
		result[key] = raw

		priorValue, ok := prior.Elements()[key].(types.String)
		if !ok || priorValue.IsNull() || priorValue.IsUnknown() {
			continue
		}
		var priorObj, apiObj map[string]any
		if json.Unmarshal([]byte(priorValue.ValueString()), &priorObj) != nil || json.Unmarshal(raw, &apiObj) != nil {
			continue
		}

		changed := false
		for field, value := range apiObj {
			if _, configured := priorObj[field]; configured {
				continue
			}
			if isUnlimitedLimit(value) {
				delete(apiObj, field)
				changed = true
			}
		}
		if changed {
			if trimmed, err := json.Marshal(apiObj); err == nil {
				result[key] = trimmed
			}
		}
	}
	return result
}

// isUnlimitedLimit reports whether a decoded JSON value is DIAL Core's
// int64-max "unlimited" sentinel (a number, or a quoted numeric string).
func isUnlimitedLimit(value any) bool {
	switch v := value.(type) {
	case float64:
		return v >= float64(math.MaxInt64)
	case string:
		return v == "9223372036854775807"
	}
	return false
}

// mapOrPlannedEmpty reconciles an API map with the planned value for an
// optional, non-computed map attribute. DIAL Core may return an empty object
// ({}) or omit the field when nothing is set, so the two are
// indistinguishable on read. When the API map is empty, the planned
// (prior) value decides: a configured empty map stays empty, anything else
// (unset, null, unknown) becomes null. Without this Terraform reports
// "inconsistent result after apply" (planned null, got {} or the reverse).
func mapOrPlannedEmpty(prior, fromAPI types.Map, apiLen int) types.Map {
	if apiLen > 0 {
		return fromAPI
	}
	if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
		return prior
	}
	return types.MapNull(types.StringType)
}

// preserveEquivalentJSON keeps the prior string for map elements whose JSON is
// semantically equal to the API value, so formatting or key order in the
// config (e.g. hand-written .tf.json strings) does not cause a diff.
func preserveEquivalentJSON(prior, fromAPI types.Map) types.Map {
	if fromAPI.IsNull() || fromAPI.IsUnknown() || prior.IsNull() || prior.IsUnknown() {
		return fromAPI
	}
	priorElems := prior.Elements()
	result := make(map[string]attr.Value, len(fromAPI.Elements()))
	for key, value := range fromAPI.Elements() {
		result[key] = value
		priorValue, ok := priorElems[key].(types.String)
		apiValue, apiOK := value.(types.String)
		if !ok || !apiOK || priorValue.IsNull() || priorValue.IsUnknown() {
			continue
		}
		var a, b any
		if json.Unmarshal([]byte(priorValue.ValueString()), &a) == nil &&
			json.Unmarshal([]byte(apiValue.ValueString()), &b) == nil &&
			reflect.DeepEqual(a, b) {
			result[key] = priorValue
		}
	}
	return types.MapValueMust(types.StringType, result)
}

// upstreamWriteOnly holds each upstream's key and secret_extra_data from the
// prior plan or state, by list index. DIAL Core never returns these fields
// (writeOnly in the API), so state keeps the configured values.
type upstreamWriteOnly []upstreamWriteOnlyEntry

type upstreamWriteOnlyEntry struct {
	key    types.String
	secret types.Map
}

func upstreamWriteOnlyValues(prior types.List) upstreamWriteOnly {
	if prior.IsNull() || prior.IsUnknown() {
		return nil
	}
	result := make(upstreamWriteOnly, 0, len(prior.Elements()))
	for _, element := range prior.Elements() {
		entry := upstreamWriteOnlyEntry{key: types.StringNull(), secret: types.MapNull(types.StringType)}
		if obj, ok := element.(types.Object); ok {
			attrs := obj.Attributes()
			if key, ok := attrs["key"].(types.String); ok && !key.IsUnknown() {
				entry.key = key
			}
			if secret, ok := attrs["secret_extra_data"].(types.Map); ok && !secret.IsUnknown() {
				entry.secret = secret
			}
		}
		result = append(result, entry)
	}
	return result
}

// at returns the prior values for upstream i. Without a prior entry (e.g.
// after import) it falls back to whatever the API returned.
func (u upstreamWriteOnly) at(ctx context.Context, i int, apiKey *string, apiSecret map[string]string) (types.String, types.Map) {
	if i < len(u) {
		return u[i].key, u[i].secret
	}
	key := types.StringNull()
	if apiKey != nil {
		key = types.StringValue(*apiKey)
	}
	secret := types.MapNull(types.StringType)
	if len(apiSecret) > 0 {
		if m, diags := types.MapValueFrom(ctx, types.StringType, apiSecret); !diags.HasError() {
			secret = m
		}
	}
	return key, secret
}

// stringSetFromAPI maps an optional API list to a set: a missing list (nil)
// stays null and an empty list stays an empty set, since DIAL Core treats
// null and [] differently.
func stringSetFromAPI(ctx context.Context, values *[]string) (types.Set, error) {
	if values == nil {
		return types.SetNull(types.StringType), nil
	}
	set, diags := types.SetValueFrom(ctx, types.StringType, *values)
	if diags.HasError() {
		return types.SetNull(types.StringType), fmt.Errorf("%s", diags.Errors()[0].Summary())
	}
	return set, nil
}

// stringSetToAPI is the inverse: null/unknown is not sent, a configured empty
// set is sent as [].
func stringSetToAPI(ctx context.Context, value types.Set) (*[]string, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	values := []string{}
	if diags := value.ElementsAs(ctx, &values, false); diags.HasError() {
		return nil, fmt.Errorf("%s", diags.Errors()[0].Summary())
	}
	if values == nil {
		values = []string{}
	}
	return &values, nil
}

// stringPointer, boolPointer, and int64Pointer convert an
// Optional/Computed Terraform attribute into the *string/*bool/*int64
// pointer shape the DIAL API client types use, treating both null and
// unknown as "not set". stringValue, boolValue, and int64Value perform
// the inverse conversion when populating state from an API response.
// These are shared across all resources with optional scalar attributes
// (currently aidial_model and aidial_route).

func stringPointer(value types.String) *string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	result := value.ValueString()
	return &result
}

func boolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	result := value.ValueBool()
	return &result
}

func int64Pointer(value types.Int64) *int64 {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}

	result := value.ValueInt64()
	return &result
}

func stringValue(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}

	return types.StringValue(*value)
}

func boolValue(value *bool) types.Bool {
	if value == nil {
		return types.BoolNull()
	}

	return types.BoolValue(*value)
}

func int64Value(value *int64) types.Int64 {
	if value == nil {
		return types.Int64Null()
	}

	return types.Int64Value(*value)
}

func limitsFromState(value types.Map) (map[string]json.RawMessage, diag.Diagnostics) {
	result := make(map[string]json.RawMessage)
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return result, diags
	}
	for key, raw := range value.Elements() {
		item, ok := raw.(types.String)
		if !ok || item.IsNull() || item.IsUnknown() {
			diags.AddError("Invalid limits value", fmt.Sprintf("limits[%q] must be a known JSON string", key))
			continue
		}
		var decoded any
		if err := json.Unmarshal([]byte(item.ValueString()), &decoded); err != nil {
			diags.AddError("Invalid limits JSON", fmt.Sprintf("limits[%q] must contain valid JSON: %s", key, err))
			continue
		}
		object, ok := decoded.(map[string]any)
		if !ok {
			diags.AddError("Invalid limits JSON", fmt.Sprintf("limits[%q] must be a JSON object", key))
			continue
		}
		canonical, _ := json.Marshal(object)
		result[key] = canonical
	}
	return result, diags
}

func limitsToState(value map[string]json.RawMessage) types.Map {
	if value == nil {
		return types.MapNull(types.StringType)
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make(map[string]attr.Value, len(value))
	for _, key := range keys {
		var decoded any
		if json.Unmarshal(value[key], &decoded) == nil {
			if canonical, err := json.Marshal(decoded); err == nil {
				result[key] = types.StringValue(string(canonical))
				continue
			}
		}
		result[key] = types.StringValue(string(value[key]))
	}
	return types.MapValueMust(types.StringType, result)
}

func shareFromState(value types.Map) (map[string]client.ShareRule, diag.Diagnostics) {
	result := make(map[string]client.ShareRule)
	var diags diag.Diagnostics
	if value.IsNull() || value.IsUnknown() {
		return result, diags
	}
	for key, raw := range value.Elements() {
		item, ok := raw.(types.String)
		if !ok || item.IsNull() || item.IsUnknown() {
			diags.AddError("Invalid share value", fmt.Sprintf("share[%q] must be a known JSON string", key))
			continue
		}
		var rule client.ShareRule
		if err := json.Unmarshal([]byte(item.ValueString()), &rule); err != nil {
			diags.AddError("Invalid share JSON", fmt.Sprintf("share[%q] must contain valid JSON: %s", key, err))
			continue
		}
		result[key] = rule
	}
	return result, diags
}

func shareToState(value map[string]client.ShareRule) types.Map {
	if value == nil {
		return types.MapNull(types.StringType)
	}
	result := make(map[string]attr.Value, len(value))
	for key, rule := range value {
		raw, _ := json.Marshal(rule)
		result[key] = types.StringValue(string(raw))
	}
	return types.MapValueMust(types.StringType, result)
}

// optionalStringMap converts an optional map(string) attribute into the value
// sent to DIAL Core; null or unknown becomes nil (omitted).
func optionalStringMap(ctx context.Context, value types.Map, name string) (map[string]string, error) {
	if value.IsNull() || value.IsUnknown() {
		return nil, nil
	}
	result := make(map[string]string, len(value.Elements()))
	if diags := value.ElementsAs(ctx, &result, false); diags.HasError() {
		return nil, fmt.Errorf("invalid %s", name)
	}
	return result, nil
}

// optionalStringMapToState maps an optional map(string) returned by DIAL Core
// (which omits empty maps) back to state: absent is null, except that an
// explicitly empty prior value is kept so the plan stays consistent.
func optionalStringMapToState(ctx context.Context, prior types.Map, api map[string]string) types.Map {
	if len(api) == 0 {
		if !prior.IsNull() && !prior.IsUnknown() && len(prior.Elements()) == 0 {
			return prior
		}
		return types.MapNull(types.StringType)
	}
	value, diags := types.MapValueFrom(ctx, types.StringType, api)
	if diags.HasError() {
		return types.MapNull(types.StringType)
	}
	return value
}

// authHeaderNameRequiredForCustomHeader fails when auth_type is CUSTOM_HEADER
// and the sibling auth_header_name is configured as null. DIAL Core rejects
// CUSTOM_HEADER without a header name.
type authHeaderNameRequiredForCustomHeader struct{}

func (authHeaderNameRequiredForCustomHeader) Description(context.Context) string {
	return "auth_header_name is required when auth_type is CUSTOM_HEADER"
}

func (v authHeaderNameRequiredForCustomHeader) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (authHeaderNameRequiredForCustomHeader) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.ConfigValue.ValueString() != "CUSTOM_HEADER" {
		return
	}
	var header types.String
	if diags := req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("auth_header_name"), &header); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if header.IsNull() {
		resp.Diagnostics.AddAttributeError(req.Path, "Missing auth_header_name", "auth_header_name is required when auth_type is CUSTOM_HEADER.")
	}
}

// authHeaderNameOnlyForCustomHeader fails when auth_header_name is configured
// next to an auth_type other than CUSTOM_HEADER. DIAL Core rejects that
// combination; an unset auth_type counts as not CUSTOM_HEADER.
type authHeaderNameOnlyForCustomHeader struct{}

func (authHeaderNameOnlyForCustomHeader) Description(context.Context) string {
	return "auth_header_name is only allowed when auth_type is CUSTOM_HEADER"
}

func (v authHeaderNameOnlyForCustomHeader) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (authHeaderNameOnlyForCustomHeader) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	var authType types.String
	if diags := req.Config.GetAttribute(ctx, req.Path.ParentPath().AtName("auth_type"), &authType); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}
	if authType.IsUnknown() || authType.ValueString() == "CUSTOM_HEADER" {
		return
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Unexpected auth_header_name", "auth_header_name is only allowed when auth_type is CUSTOM_HEADER.")
}
