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
