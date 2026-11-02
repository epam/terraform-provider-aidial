package provider

import (
	"context"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNameOptionalComputedWithoutSchemaBump(t *testing.T) {
	// Required -> Optional+Computed keeps the stored type, so state needs no upgrade.
	wantVersion := map[string]int64{"aidial_model": 1, "aidial_role": 0, "aidial_route": 0}
	cases := map[string]resource.Resource{
		"aidial_model": &modelResource{},
		"aidial_role":  &roleResource{},
		"aidial_route": &routeResource{},
	}
	for name, r := range cases {
		t.Run(name, func(t *testing.T) {
			var resp resource.SchemaResponse
			r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
			a, ok := resp.Schema.Attributes["name"].(schema.StringAttribute)
			if !ok {
				t.Fatal("name is not a StringAttribute")
			}
			if a.Required || !a.Optional || !a.Computed || len(a.PlanModifiers) == 0 {
				t.Fatalf("name must be optional+computed with a plan modifier, got %+v", a)
			}
			if resp.Schema.Version != wantVersion[name] {
				t.Fatalf("schema version = %d, want %d (no bump)", resp.Schema.Version, wantVersion[name])
			}
		})
	}
}

func TestNameFromAPIWhenOmitted(t *testing.T) {
	ctx := context.Background()
	t.Run("role", func(t *testing.T) {
		state := roleResourceModel{Name: types.StringUnknown()}
		if err := setRoleState(ctx, &state, &client.Role{Name: "p"}); err != nil {
			t.Fatal(err)
		}
		if state.Name.ValueString() != "p" || state.APIName.ValueString() != "p" {
			t.Fatalf("name=%v api_name=%v, want p", state.Name, state.APIName)
		}
		// a later read with another API value must not rewrite the known name
		if err := setRoleState(ctx, &state, &client.Role{Name: "other"}); err != nil {
			t.Fatal(err)
		}
		if state.Name.ValueString() != "p" || state.APIName.ValueString() != "other" {
			t.Fatalf("name=%v api_name=%v", state.Name, state.APIName)
		}
	})
	t.Run("route", func(t *testing.T) {
		state := routeResourceModel{Name: types.StringUnknown()}
		if err := setRouteState(ctx, &state, &client.Route{Name: "routes/platform/p"}); err != nil {
			t.Fatal(err)
		}
		if state.Name.ValueString() != "routes/platform/p" || state.APIName.ValueString() != "routes/platform/p" {
			t.Fatalf("name=%v api_name=%v", state.Name, state.APIName)
		}
	})
	t.Run("model", func(t *testing.T) {
		state := modelResourceModel{Name: types.StringUnknown()}
		if err := setModelState(ctx, &state, &client.Model{Name: "p", Type: "CHAT"}); err != nil {
			t.Fatal(err)
		}
		if state.Name.ValueString() != "p" || state.APIName.ValueString() != "p" {
			t.Fatalf("name=%v api_name=%v", state.Name, state.APIName)
		}
	})
}

func TestConfiguredNameKeptWhenAPIReturnsAnother(t *testing.T) {
	ctx := context.Background()
	r := roleResourceModel{Name: types.StringValue("custom")}
	if err := setRoleState(ctx, &r, &client.Role{Name: "p"}); err != nil {
		t.Fatal(err)
	}
	m := modelResourceModel{Name: types.StringValue("custom")}
	if err := setModelState(ctx, &m, &client.Model{Name: "p", Type: "CHAT"}); err != nil {
		t.Fatal(err)
	}
	rt := routeResourceModel{Name: types.StringValue("custom")}
	if err := setRouteState(ctx, &rt, &client.Route{Name: "routes/platform/p"}); err != nil {
		t.Fatal(err)
	}
	if r.Name.ValueString() != "custom" || m.Name.ValueString() != "custom" || rt.Name.ValueString() != "custom" {
		t.Fatalf("configured names rewritten: %v %v %v", r.Name, m.Name, rt.Name)
	}
	if r.APIName.ValueString() != "p" || m.APIName.ValueString() != "p" || rt.APIName.ValueString() != "routes/platform/p" {
		t.Fatalf("api_name wrong: %v %v %v", r.APIName, m.APIName, rt.APIName)
	}
}
