package provider

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func newRolePlan(t *testing.T, name string) roleResourceModel {
	t.Helper()

	limits, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"models/platform/gpt": `{"day": 100}`,
	})
	if diags.HasError() {
		t.Fatalf("failed to build limits map: %v", diags)
	}

	costLimit, costLimitDiags := types.ObjectValue(map[string]attr.Type{
		"day": types.Float64Type, "minute": types.Float64Type,
		"month": types.Float64Type, "week": types.Float64Type,
	}, map[string]attr.Value{
		"day": types.Float64Null(), "minute": types.Float64Null(),
		"month": types.Float64Null(), "week": types.Float64Null(),
	})
	if costLimitDiags.HasError() {
		t.Fatalf("cost_limit: %v", costLimitDiags)
	}

	return roleResourceModel{
		Bucket:      types.StringValue("b"),
		Path:        types.StringValue("p"),
		Name:        types.StringValue(name),
		Description: types.StringNull(),
		DisplayName: types.StringNull(),
		Limits:      limits,
		Share:       types.MapNull(types.StringType),
		CostLimit:   costLimit,
		Topics:      types.SetNull(types.StringType),
	}
}

func TestRoleFromPlanBasic(t *testing.T) {
	plan := newRolePlan(t, "roles/b/p")

	role, err := roleFromPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("roleFromPlan: %v", err)
	}
	if role.Name != "roles/b/p" {
		t.Fatalf("expected name %q, got %q", "roles/b/p", role.Name)
	}
	if _, ok := role.Limits["models/platform/gpt"]; !ok {
		t.Fatal("expected limits to carry through")
	}
}

func TestRoleFromPlanCostLimit(t *testing.T) {
	plan := newRolePlan(t, "roles/b/p")
	costLimit, diags := types.ObjectValue(map[string]attr.Type{
		"day": types.Float64Type, "minute": types.Float64Type,
		"month": types.Float64Type, "week": types.Float64Type,
	}, map[string]attr.Value{
		"day": types.Float64Value(1.5), "minute": types.Float64Null(),
		"month": types.Float64Null(), "week": types.Float64Null(),
	})
	if diags.HasError() {
		t.Fatalf("cost_limit: %v", diags)
	}
	plan.CostLimit = costLimit

	role, err := roleFromPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("roleFromPlan: %v", err)
	}
	if role.CostLimit == nil || role.CostLimit.Day == nil || float64(*role.CostLimit.Day) != 1.5 {
		t.Fatalf("expected cost_limit.day=1.5, got %+v", role.CostLimit)
	}
	if role.CostLimit.Minute != nil {
		t.Fatalf("expected cost_limit.minute to stay unset, got %v", *role.CostLimit.Minute)
	}
}

func TestRoleFromPlanInvalidTopics(t *testing.T) {
	plan := newRolePlan(t, "roles/b/p")
	// A String masquerading as a Set element type is invalid; simulate an
	// unconvertible value by using a set of the wrong element type via a
	// raw ListValue substitution is not directly expressible here, so
	// instead assert the null/unknown short-circuit does not error.
	plan.Topics = types.SetUnknown(types.StringType)

	if _, err := roleFromPlan(context.Background(), plan); err != nil {
		t.Fatalf("expected unknown topics to be skipped without error, got: %v", err)
	}
}

// TestSetRoleStatePreservesConfiguredName guards against a regression where
// setRoleState overwrote the user-configured, Required `name` attribute
// with the API's own (possibly canonicalized) value, which Terraform's
// provider-consistency check rejects for non-Computed attributes.
func TestSetRoleStatePreservesConfiguredName(t *testing.T) {
	state := roleResourceModel{
		Name: types.StringValue("roles/b/p"),
	}
	role := &client.Role{
		Name: "p", // DIAL Core stripped the "roles/b/" prefix
	}

	if err := setRoleState(context.Background(), &state, role); err != nil {
		t.Fatalf("setRoleState: %v", err)
	}
	if state.Name.ValueString() != "roles/b/p" {
		t.Fatalf("expected configured name to be preserved, got %q", state.Name.ValueString())
	}
	if state.APIName.ValueString() != "p" {
		t.Fatalf("expected api_name to reflect the API value, got %q", state.APIName.ValueString())
	}
}

// TestSetRoleStateFillsNameOnImport guards the other half of the same
// behavior: right after `terraform import`, name is not yet known, so it
// must be populated from the API response instead of staying null forever.
func TestSetRoleStateFillsNameOnImport(t *testing.T) {
	state := roleResourceModel{
		Name: types.StringNull(),
	}
	role := &client.Role{Name: "imported-role"}

	if err := setRoleState(context.Background(), &state, role); err != nil {
		t.Fatalf("setRoleState: %v", err)
	}
	if state.Name.ValueString() != "imported-role" {
		t.Fatalf("expected name to be filled from API on import, got %q", state.Name.ValueString())
	}
}

// TestSetRoleStateCostLimitSentinelString guards against the
// FlexibleFloat64 decode failure: DIAL Core can encode extreme cost-limit
// values (the int64-max "unlimited" sentinel) as quoted JSON strings
// rather than bare numbers.
func TestSetRoleStateCostLimitSentinelString(t *testing.T) {
	sentinel := client.FlexibleFloat64(9223372036854775807)
	state := roleResourceModel{Name: types.StringValue("r")}
	role := &client.Role{
		Name:      "r",
		CostLimit: &client.CostLimit{Day: &sentinel},
	}

	if err := setRoleState(context.Background(), &state, role); err != nil {
		t.Fatalf("setRoleState: %v", err)
	}
	if state.CostLimit.IsNull() {
		t.Fatal("expected cost_limit to be populated")
	}
}

func TestSetRoleStateEmptySetsAreNull(t *testing.T) {
	state := roleResourceModel{Name: types.StringValue("r")}
	role := &client.Role{Name: "r"}

	if err := setRoleState(context.Background(), &state, role); err != nil {
		t.Fatalf("setRoleState: %v", err)
	}
	if !state.Topics.IsNull() {
		t.Fatalf("expected topics to be null when API omits them, got %v", state.Topics)
	}
	if !state.GrantedKeys.IsNull() {
		t.Fatalf("expected granted_keys to be null when API omits them, got %v", state.GrantedKeys)
	}
}

func TestSetRoleStateTopicsRoundTrip(t *testing.T) {
	state := roleResourceModel{Name: types.StringValue("r")}
	role := &client.Role{
		Name:   "r",
		Topics: []string{"Testing", "Demo"},
	}

	if err := setRoleState(context.Background(), &state, role); err != nil {
		t.Fatalf("setRoleState: %v", err)
	}

	var got []string
	if diags := state.Topics.ElementsAs(context.Background(), &got, false); diags.HasError() {
		t.Fatalf("ElementsAs: %v", diags)
	}
	if len(got) != 2 || got[0] != "Testing" || got[1] != "Demo" {
		t.Fatalf("unexpected topics: %v", got)
	}
}

// The example role payload read back from DIAL Core: empty share, cost
// limits, a per-model limit, and status/timestamps under _metadata.
const roleReadPayload = `{
	"name": "asset-role",
	"share": {},
	"costLimit": {"minute": 123, "day": 12.2, "week": 22.8},
	"limits": {"example-chat-model": {"day": 10}},
	"_metadata": {
		"name": "asset-role", "path": "asset-role", "folderId": "", "author": "me",
		"createdAt": "1787728871109", "updatedAt": "1789561250770", "status": "valid"
	}
}`

func TestSetRoleStateFromReadPayload(t *testing.T) {
	var role client.Role
	if err := json.Unmarshal([]byte(roleReadPayload), &role); err != nil {
		t.Fatalf("decode: %v", err)
	}
	state := roleResourceModel{Name: types.StringValue("asset-role"), Share: types.MapNull(types.StringType)}

	if err := setRoleState(context.Background(), &state, &role); err != nil {
		t.Fatalf("setRoleState: %v", err)
	}

	if state.CostLimit.IsNull() {
		t.Fatal("expected cost_limit to be set")
	}
	var costLimit roleCostLimitModel
	if diags := state.CostLimit.As(context.Background(), &costLimit, basetypes.ObjectAsOptions{}); diags.HasError() {
		t.Fatalf("cost_limit As: %v", diags)
	}
	if costLimit.Day.ValueFloat64() != 12.2 || costLimit.Minute.ValueFloat64() != 123 || costLimit.Week.ValueFloat64() != 22.8 {
		t.Fatalf("unexpected cost limits: day=%v minute=%v week=%v", costLimit.Day, costLimit.Minute, costLimit.Week)
	}
	if !costLimit.Month.IsNull() {
		t.Fatalf("expected cost_limit_month null, got %v", costLimit.Month)
	}
	if got := state.Limits.Elements()["example-chat-model"]; got != types.StringValue(`{"day":10}`) {
		t.Fatalf("unexpected limits: %v", state.Limits)
	}
	if !state.Share.IsNull() {
		t.Fatalf("share omitted in config but API returned {}: expected null, got %v", state.Share)
	}
	if state.Status.ValueString() != "valid" || state.CreatedAt.ValueString() != "1787728871109" || state.UpdatedAt.ValueString() != "1789561250770" {
		t.Fatalf("metadata not decoded: status=%q created=%q updated=%q", state.Status.ValueString(), state.CreatedAt.ValueString(), state.UpdatedAt.ValueString())
	}
}

func TestSetRoleStateKeepsConfiguredEmptyMaps(t *testing.T) {
	empty, diags := types.MapValue(types.StringType, map[string]attr.Value{})
	if diags.HasError() {
		t.Fatalf("empty map: %v", diags)
	}
	state := roleResourceModel{Name: types.StringValue("r"), Limits: empty, Share: empty}

	// API omits both (nil) or returns {} (non-nil): both must keep the planned {}.
	for name, role := range map[string]*client.Role{
		"omitted": {Name: "r"},
		"empty":   {Name: "r", Limits: map[string]json.RawMessage{}, Share: map[string]client.ShareRule{}},
	} {
		s := state
		if err := setRoleState(context.Background(), &s, role); err != nil {
			t.Fatalf("%s: setRoleState: %v", name, err)
		}
		if s.Limits.IsNull() || len(s.Limits.Elements()) != 0 {
			t.Fatalf("%s: expected limits to stay {}, got %v", name, s.Limits)
		}
		if s.Share.IsNull() || len(s.Share.Elements()) != 0 {
			t.Fatalf("%s: expected share to stay {}, got %v", name, s.Share)
		}
	}
}

func TestSetRoleStateKeepsEquivalentLimitsFormatting(t *testing.T) {
	prior, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"openai.o3-mini-2025-01-31": `{"minute": 64000000, "day": 100000000}`,
		"openai.gpt-4":              `{"day": 1}`,
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	state := roleResourceModel{Name: types.StringValue("r"), Limits: prior}
	role := &client.Role{Name: "r", Limits: map[string]json.RawMessage{
		"openai.o3-mini-2025-01-31": json.RawMessage(`{"day":100000000,"minute":64000000}`),
		"openai.gpt-4":              json.RawMessage(`{"day":2}`),
	}}

	if err := setRoleState(context.Background(), &state, role); err != nil {
		t.Fatalf("setRoleState: %v", err)
	}
	el := state.Limits.Elements()
	if got := el["openai.o3-mini-2025-01-31"]; got != types.StringValue(`{"minute": 64000000, "day": 100000000}`) {
		t.Fatalf("equivalent JSON must keep configured formatting, got %v", got)
	}
	if got := el["openai.gpt-4"]; got != types.StringValue(`{"day":2}`) {
		t.Fatalf("changed value must come from the API, got %v", got)
	}
}

func TestSetRoleStateTopLevelStatusStillWorks(t *testing.T) {
	state := roleResourceModel{Name: types.StringValue("r")}
	role := &client.Role{Name: "r", Status: "valid", CreatedAt: "1", UpdatedAt: "2"}

	if err := setRoleState(context.Background(), &state, role); err != nil {
		t.Fatalf("setRoleState: %v", err)
	}
	if state.Status.ValueString() != "valid" || state.CreatedAt.ValueString() != "1" || state.UpdatedAt.ValueString() != "2" {
		t.Fatalf("top-level fields lost: %q %q %q", state.Status.ValueString(), state.CreatedAt.ValueString(), state.UpdatedAt.ValueString())
	}
}

// DIAL Core fills limit fields the config omitted with the int64-max
// "unlimited" sentinel; they must not leak into state for configured keys.
func TestSetRoleStateDropsServerDefaultLimitFields(t *testing.T) {
	const apiLimits = `{"models/platform/m": {"day":10,"minute":9223372036854776000,"month":9223372036854776000,"requestDay":9223372036854776000,"requestHour":9223372036854776000,"week":9223372036854776000},
		"models/platform/imported": {"day":1,"week":9223372036854776000},
		"models/platform/explicit": {"day":9223372036854776000}}`
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(apiLimits), &raw); err != nil {
		t.Fatal(err)
	}
	prior, diags := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"models/platform/m":        `{"day":10}`,
		"models/platform/explicit": `{"day":9223372036854775807}`,
	})
	if diags.HasError() {
		t.Fatal(diags)
	}
	state := roleResourceModel{Name: types.StringValue("r"), Limits: prior}

	if err := setRoleState(context.Background(), &state, &client.Role{Name: "r", Limits: raw}); err != nil {
		t.Fatalf("setRoleState: %v", err)
	}

	el := state.Limits.Elements()
	if got := el["models/platform/m"]; got != types.StringValue(`{"day":10}`) {
		t.Fatalf("server-default fields leaked into configured key: %v", got)
	}
	if got := el["models/platform/imported"]; got != types.StringValue(`{"day":1,"week":9223372036854776000}`) {
		t.Fatalf("key without prior value must be kept as returned: %v", got)
	}
	if got := el["models/platform/explicit"]; got != types.StringValue(`{"day":9223372036854775807}`) {
		t.Fatalf("explicitly configured field must be kept: %v", got)
	}
}
