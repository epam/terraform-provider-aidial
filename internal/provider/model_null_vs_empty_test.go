package provider

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// DIAL Core treats a missing value (null) and an empty one ("" or []) as
// different settings, so both directions must keep them apart.

func TestSetModelStateKeepsNullVsEmptyLists(t *testing.T) {
	ctx := context.Background()
	empty := []string{}
	model := &client.Model{
		Name:                 "m",
		Type:                 "CHAT",
		UserRoles:            nil,
		InputAttachmentTypes: nil,
		Interceptors:         &empty,
		DescriptionKeywords:  &empty,
		Dependencies:         &[]string{"a"},
	}
	state := modelResourceModel{Name: types.StringValue("m")}
	if err := setModelState(ctx, &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	for name, value := range map[string]types.Set{"user_roles": state.UserRoles, "input_attachment_types": state.InputAttachmentTypes, "fields_hashing_order": state.FieldsHashingOrder} {
		if !value.IsNull() {
			t.Fatalf("%s: a missing list must stay null, got %v", name, value)
		}
	}
	for name, value := range map[string]types.Set{"interceptors": state.Interceptors, "description_keywords": state.DescriptionKeywords} {
		if value.IsNull() || len(value.Elements()) != 0 {
			t.Fatalf("%s: an empty list must stay an empty set, got %v", name, value)
		}
	}
	if len(state.Dependencies.Elements()) != 1 {
		t.Fatalf("dependencies must round-trip, got %v", state.Dependencies)
	}
}

func TestSetModelStateKeepsEmptyStringVsNull(t *testing.T) {
	ctx := context.Background()
	state := modelResourceModel{Name: types.StringValue("m")}
	model := &client.Model{Name: "m", Type: "CHAT", Description: ptr(""), DisplayVersion: nil}
	if err := setModelState(ctx, &state, model); err != nil {
		t.Fatalf("setModelState: %v", err)
	}
	if state.Description.IsNull() || state.Description.ValueString() != "" {
		t.Fatalf(`description "" must stay "", got %v`, state.Description)
	}
	if !state.DisplayVersion.IsNull() {
		t.Fatalf("missing display_version must stay null, got %v", state.DisplayVersion)
	}
}

func TestModelFromPlanSendsEmptyValuesAndOmitsUnset(t *testing.T) {
	ctx := context.Background()
	emptySet, _ := types.SetValue(types.StringType, []attr.Value{})
	plan := newModelPlan(t)
	plan.UserRoles = emptySet
	plan.InputAttachmentTypes = types.SetNull(types.StringType)
	plan.Interceptors = types.SetUnknown(types.StringType)
	plan.Description = types.StringValue("")
	plan.DisplayVersion = types.StringNull()

	model, err := modelFromPlan(ctx, plan)
	if err != nil {
		t.Fatalf("modelFromPlan: %v", err)
	}
	body, _ := json.Marshal(model)
	for _, want := range []string{`"userRoles":[]`, `"description":""`} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("expected %s in request, got %s", want, body)
		}
	}
	for _, absent := range []string{`"inputAttachmentTypes"`, `"interceptors"`, `"displayVersion"`} {
		if strings.Contains(string(body), absent) {
			t.Fatalf("unset %s must not be sent, got %s", absent, body)
		}
	}
}

func TestUpstreamWithoutEndpointOmitsIt(t *testing.T) {
	body, _ := json.Marshal(client.ModelUpstream{ID: ptr("1")})
	if strings.Contains(string(body), "endpoint") {
		t.Fatalf(`an unset upstream endpoint must not be sent as "", got %s`, body)
	}
}

func TestModelSchemaNullDefaultFieldsAreNotComputed(t *testing.T) {
	var resp resource.SchemaResponse
	(&modelResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	computed := func(name string) bool {
		switch a := resp.Schema.Attributes[name].(type) {
		case schema.StringAttribute:
			return a.Computed
		case schema.Int64Attribute:
			return a.Computed
		case schema.BoolAttribute:
			return a.Computed
		case schema.SetAttribute:
			return a.Computed
		case schema.ListNestedAttribute:
			return a.Computed
		case schema.SingleNestedAttribute:
			return a.Computed
		}
		t.Fatalf("unexpected attribute type for %s", name)
		return false
	}
	// DIAL Core leaves these null when unset: unset in config must mean null.
	for _, name := range []string{"description", "display_name", "endpoint", "responses_endpoint", "override_name", "display_version", "icon_url", "intro", "reference", "tokenizer_model", "max_input_attachments", "user_roles", "input_attachment_types", "features", "limits", "pricing"} {
		if computed(name) {
			t.Errorf("%s must not be computed", name)
		}
	}
	// DIAL Core fills these with defaults: they must stay computed.
	for _, name := range []string{"forward_auth_token", "max_retry_attempts", "interceptors", "description_keywords", "dependencies", "upstreams", "fields_hashing_order"} {
		if !computed(name) {
			t.Errorf("%s must stay computed", name)
		}
	}
}
