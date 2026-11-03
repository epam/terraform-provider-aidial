package provider

import (
	"context"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestKeyFromPlanBasic(t *testing.T) {
	roles, diags := types.SetValueFrom(context.Background(), types.StringType, []string{"developer"})
	if diags.HasError() {
		t.Fatalf("failed to build roles set: %v", diags)
	}

	plan := keyResourceModel{
		Bucket:  types.StringValue("b"),
		Path:    types.StringValue("p"),
		Key:     types.StringValue("super-secret"),
		Project: types.StringValue("demo"),
		Role:    types.StringValue("developer"),
		Roles:   roles,
		Secured: types.BoolValue(true),
	}

	key, err := keyFromPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("keyFromPlan: %v", err)
	}
	if key.Key != "super-secret" {
		t.Fatalf("expected key %q, got %q", "super-secret", key.Key)
	}
	if key.Secured == nil || !*key.Secured {
		t.Fatal("expected secured=true")
	}
}

// TestSetKeyStatePreservesConfiguredKey guards against the DIAL Core
// `Key.key` field being `writeOnly` per the OpenAPI spec: a GET response
// never echoes the secret value back, so client.Key.Key decodes as "" on
// refresh. Since `key` is Required (not Computed) in the schema,
// unconditionally overwriting state.Key with that empty string would trip
// the same Terraform provider-consistency failure the role name/api_name
// bug did.
func TestSetKeyStatePreservesConfiguredKey(t *testing.T) {
	state := keyResourceModel{
		Key: types.StringValue("super-secret"),
	}
	// Simulates the API response to a post-create GET: `key` is absent
	// (writeOnly), so it decodes to Go's zero value.
	key := &client.Key{
		Key:  "",
		Name: "b/p",
	}

	if err := setKeyState(context.Background(), &state, key); err != nil {
		t.Fatalf("setKeyState: %v", err)
	}
	if state.Key.ValueString() != "super-secret" {
		t.Fatalf("expected configured key to be preserved, got %q", state.Key.ValueString())
	}
}

func TestSetKeyStateRolesRoundTrip(t *testing.T) {
	state := keyResourceModel{Key: types.StringValue("k")}
	key := &client.Key{
		Key:   "k",
		Roles: []string{"developer", "admin"},
	}

	if err := setKeyState(context.Background(), &state, key); err != nil {
		t.Fatalf("setKeyState: %v", err)
	}

	var got []string
	if diags := state.Roles.ElementsAs(context.Background(), &got, false); diags.HasError() {
		t.Fatalf("ElementsAs: %v", diags)
	}
	if len(got) != 2 || got[0] != "developer" || got[1] != "admin" {
		t.Fatalf("unexpected roles: %v", got)
	}
}

func TestSetKeyStateSecuredNull(t *testing.T) {
	state := keyResourceModel{Key: types.StringValue("k")}
	key := &client.Key{Key: "k"}

	if err := setKeyState(context.Background(), &state, key); err != nil {
		t.Fatalf("setKeyState: %v", err)
	}
	if !state.Secured.IsNull() {
		t.Fatalf("expected secured to be null when API omits it, got %v", state.Secured)
	}
}

// The key-state-consistency spec: omitted project/role are null, except that
// an explicit "" in the plan or prior state is kept.
func TestSetKeyStateProjectRoleNullVsEmpty(t *testing.T) {
	cases := []struct {
		name        string
		prior       types.String
		api         string
		wantProject types.String
	}{
		{"omitted, unknown plan", types.StringUnknown(), "", types.StringNull()},
		{"omitted, null prior", types.StringNull(), "", types.StringNull()},
		{"explicit empty string kept", types.StringValue(""), "", types.StringValue("")},
		{"removed outside terraform", types.StringValue("demo"), "", types.StringNull()},
		{"value returned", types.StringNull(), "demo", types.StringValue("demo")},
		{"value returned over empty prior", types.StringValue(""), "demo", types.StringValue("demo")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := keyResourceModel{
				Key:     types.StringValue("k"),
				Project: tc.prior,
				Role:    tc.prior,
			}
			key := &client.Key{Project: tc.api, Role: tc.api}

			if err := setKeyState(context.Background(), &state, key); err != nil {
				t.Fatalf("setKeyState: %v", err)
			}
			if !state.Project.Equal(tc.wantProject) {
				t.Errorf("project: want %v, got %v", tc.wantProject, state.Project)
			}
			if !state.Role.Equal(tc.wantProject) {
				t.Errorf("role: want %v, got %v", tc.wantProject, state.Role)
			}
		})
	}
}

// allowed_ip_address_ranges: unset means any client (null), an empty set means
// no client, and Core's CIDR normalization must not cause a diff.
func TestKeyAllowedIPAddressRanges(t *testing.T) {
	ctx := context.Background()
	set := func(values ...string) types.Set {
		v, diags := types.SetValueFrom(ctx, types.StringType, append([]string{}, values...))
		if diags.HasError() {
			t.Fatalf("SetValueFrom: %v", diags)
		}
		return v
	}
	api := func(cidrs ...string) *client.AllowedIPAddressRanges {
		if cidrs == nil {
			cidrs = []string{}
		}
		return &client.AllowedIPAddressRanges{CIDRs: cidrs}
	}

	cases := []struct {
		name  string
		prior types.Set
		api   *client.AllowedIPAddressRanges
		want  types.Set
	}{
		{"omitted by API", types.SetNull(types.StringType), nil, types.SetNull(types.StringType)},
		{"empty set", set(), api(), set()},
		{"normalized spelling kept", set("198.51.100.14/24"), api("198.51.100.0/24"), set("198.51.100.14/24")},
		{"changed outside Terraform", set("10.0.0.0/8"), api("192.0.2.0/24"), set("192.0.2.0/24")},
		{"undecodable keeps prior", set("10.0.0.0/8"), &client.AllowedIPAddressRanges{Undecodable: true}, set("10.0.0.0/8")},
		{"import", types.SetNull(types.StringType), api("192.0.2.0/24"), set("192.0.2.0/24")},
	}
	for _, tc := range cases {
		got, err := allowedIPRangesToState(ctx, tc.prior, tc.api)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if !got.Equal(tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}

	plan := keyResourceModel{Key: types.StringValue("k"), AllowedIPAddressRanges: set("198.51.100.14/24")}
	key, err := keyFromPlan(ctx, plan)
	if err != nil {
		t.Fatal(err)
	}
	if key.AllowedIPAddressRanges == nil || len(key.AllowedIPAddressRanges.CIDRs) != 1 {
		t.Fatalf("expected the configured CIDRs to be sent, got %+v", key.AllowedIPAddressRanges)
	}
	key, _ = keyFromPlan(ctx, keyResourceModel{Key: types.StringValue("k"), AllowedIPAddressRanges: types.SetNull(types.StringType)})
	if key.AllowedIPAddressRanges != nil {
		t.Fatal("an unset allowed_ip_address_ranges must be omitted")
	}
}
