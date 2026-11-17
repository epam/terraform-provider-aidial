package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// coreFillsNestedDefaults imitates what Core 0.50.1 adds to mcp and routes and
// what it never returns for external services.
func coreFillsNestedDefaults(doc map[string]any) {
	if mcp, ok := doc["mcp"].(map[string]any); ok {
		for key, value := range map[string]any{"transport": "HTTP", "configDelivery": "META", "forwardPerRequestKey": true, "allowedTools": []any{}} {
			if _, set := mcp[key]; !set {
				mcp[key] = value
			}
		}
	}
	if routes, ok := doc["routes"].(map[string]any); ok {
		for _, route := range routes {
			r := route.(map[string]any)
			for key, value := range map[string]any{
				"order": 2147483647, "methods": []any{}, "permissions": []any{}, "upstreams": []any{},
				"rewritePath": false, "maxRetryAttempts": 1,
				"attachmentPaths": map[string]any{"requestBody": []any{}, "responseBody": []any{}},
			} {
				if _, set := r[key]; !set {
					r[key] = value
				}
			}
		}
	}
	if services, ok := doc["external_services"].(map[string]any); ok {
		for _, service := range services {
			if auth, ok := service.(map[string]any)["auth_settings"].(map[string]any); ok {
				delete(auth, "client_secret")
				delete(auth, "code_verifier")
				auth["client_secret_hint"] = "2345"
				auth["user_level_auth_status"] = "SIGNED_OUT"
			}
		}
	}
}

func appRouteValue(paths ...string) attr.Value {
	pathValues := make([]attr.Value, len(paths))
	for i, p := range paths {
		pathValues[i] = types.StringValue(p)
	}
	return types.ObjectValueMust(appRouteAttrTypes, map[string]attr.Value{
		"order":              types.Int64Unknown(),
		"methods":            types.SetUnknown(types.StringType),
		"paths":              types.SetValueMust(types.StringType, pathValues),
		"permissions":        types.SetUnknown(types.StringType),
		"user_roles":         types.SetUnknown(types.StringType),
		"max_retry_attempts": types.Int64Unknown(),
		"rewrite_path":       types.BoolUnknown(),
		"response": types.ObjectValueMust(appRouteResponseAttrTypes, map[string]attr.Value{
			"body": types.StringValue("hi"), "status": types.Int64Value(200),
		}),
		"attachment_paths": types.ObjectUnknown(appRouteAttachmentAttrTypes),
		"upstreams":        types.ListUnknown(types.ObjectType{AttrTypes: appRouteUpstreamAttrTypes}),
	})
}

func TestApplicationSimpleAttributesRoundTrip(t *testing.T) {
	core := &fakeApplicationCore{}
	r := &applicationResource{client: core.client(t)}

	plan := newApplicationPlan("http://e")
	plan.InputAttachmentTypes = stringSet(t, "*/*")
	plan.DescriptionKeywords = stringSet(t, "a", "b")
	plan.Dependencies = stringSet(t, "m1")
	plan.MaxInputAttachments = types.Int64Value(3)
	plan.AllowUserExternalServices = types.BoolValue(true)
	plan.ResponsesEndpoint, plan.Intro = types.StringValue("http://r"), types.StringValue("hi")
	plan.OverrideName, plan.BaseURL = types.StringValue("ov"), types.StringValue("http://b")
	plan.Reference = types.StringValue("my-ref")

	resp, state := runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	body := core.body(t)
	for key, want := range map[string]any{
		"max_input_attachments": float64(3), "allow_user_external_services": true, "responses_endpoint": "http://r",
		"intro": "hi", "override_name": "ov", "base_url": "http://b", "reference": "my-ref",
	} {
		if body[key] != want {
			t.Errorf("body[%s] = %v, want %v", key, body[key], want)
		}
	}
	for _, key := range []string{"input_attachment_types", "description_keywords", "dependencies"} {
		if list, _ := body[key].([]any); len(list) == 0 {
			t.Errorf("body[%s] = %v", key, body[key])
		}
	}
	if !state.InputAttachmentTypes.Equal(plan.InputAttachmentTypes) || !state.Dependencies.Equal(plan.Dependencies) ||
		!state.DescriptionKeywords.Equal(plan.DescriptionKeywords) || state.MaxInputAttachments.ValueInt64() != 3 ||
		!state.AllowUserExternalServices.ValueBool() || state.Reference.ValueString() != "my-ref" {
		t.Fatalf("simple attributes did not round trip: %+v", state.applicationBody)
	}
}

func TestApplicationMinimalConfigLeavesNewAttributesDefaulted(t *testing.T) {
	core := &fakeApplicationCore{}
	r := &applicationResource{client: core.client(t)}

	resp, state := runApplicationCreate(t, r, newApplicationPlan("http://e"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	body := core.body(t)
	for _, key := range []string{"input_attachment_types", "description_keywords", "dependencies", "mcp", "routes", "external_services", "max_input_attachments", "allow_user_external_services", "reference"} {
		if _, ok := body[key]; ok {
			t.Errorf("%s must not be sent when unset: %v", key, body)
		}
	}
	if !state.InputAttachmentTypes.IsNull() || !state.MaxInputAttachments.IsNull() || !state.AllowUserExternalServices.IsNull() {
		t.Errorf("plain optional attributes must stay null: %+v", state.applicationBody)
	}
	if state.DescriptionKeywords.IsNull() || len(state.DescriptionKeywords.Elements()) != 0 || len(state.Dependencies.Elements()) != 0 {
		t.Errorf("Core-defaulted sets must read back empty: %+v", state.applicationBody)
	}
	if state.Routes.IsNull() || len(state.Routes.Elements()) != 0 {
		t.Errorf("routes must read back as an empty map, got %v", state.Routes)
	}
	if !state.Mcp.IsNull() || !state.ExternalServices.IsNull() {
		t.Errorf("mcp and external_services must be null: %v %v", state.Mcp, state.ExternalServices)
	}
	if state.Reference.ValueString() != "generated-ref" {
		t.Errorf("reference = %v", state.Reference)
	}
}

func TestApplicationPartialMcpReadsBackCoreDefaults(t *testing.T) {
	core := &fakeApplicationCore{transform: coreFillsNestedDefaults}
	r := &applicationResource{client: core.client(t)}

	plan := newApplicationPlan("http://e")
	plan.Mcp = types.ObjectValueMust(appMcpAttrTypes, map[string]attr.Value{
		"endpoint":                types.StringValue("http://h/mcp"),
		"transport":               types.StringUnknown(),
		"allowed_tools":           types.SetUnknown(types.StringType),
		"config_delivery":         types.StringUnknown(),
		"forward_per_request_key": types.BoolUnknown(),
		"mcp_apps": types.ObjectValueMust(appMcpAppsAttrTypes, map[string]attr.Value{
			"domain_override": types.StringValue("x.com"),
		}),
	})
	resp, state := runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	mcp, _ := core.body(t)["mcp"].(map[string]any)
	if mcp["endpoint"] != "http://h/mcp" || mcp["mcpApps"].(map[string]any)["domainOverride"] != "x.com" {
		t.Errorf("mcp body = %v", mcp)
	}
	for _, key := range []string{"transport", "allowedTools", "configDelivery", "forwardPerRequestKey"} {
		if _, ok := mcp[key]; ok {
			t.Errorf("unknown %s must not be sent: %v", key, mcp)
		}
	}
	attrs := state.Mcp.Attributes()
	if attrs["config_delivery"].(types.String).ValueString() != "META" || !attrs["forward_per_request_key"].(types.Bool).ValueBool() ||
		attrs["transport"].(types.String).ValueString() != "HTTP" || len(attrs["allowed_tools"].(types.Set).Elements()) != 0 {
		t.Fatalf("Core defaults not read back: %v", state.Mcp)
	}
}

func TestApplicationMcpRemovedIsNotSent(t *testing.T) {
	core := &fakeApplicationCore{transform: coreFillsNestedDefaults}
	r := &applicationResource{client: core.client(t)}

	plan := newApplicationPlan("http://e")
	plan.Mcp = types.ObjectValueMust(appMcpAttrTypes, map[string]attr.Value{
		"endpoint": types.StringValue("http://h/mcp"), "transport": types.StringUnknown(),
		"allowed_tools": types.SetUnknown(types.StringType), "config_delivery": types.StringUnknown(),
		"forward_per_request_key": types.BoolUnknown(), "mcp_apps": types.ObjectNull(appMcpAppsAttrTypes),
	})
	if resp, _ := runApplicationCreate(t, r, plan); resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	again := newApplicationPlan("http://e")
	resp, state := runApplicationCreate(t, r, again)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	if _, ok := core.body(t)["mcp"]; ok || !state.Mcp.IsNull() {
		t.Errorf("mcp must be gone: body %v state %v", core.body(t), state.Mcp)
	}
}

func TestApplicationMcpValidators(t *testing.T) {
	mcp := applicationNestedResourceAttributes()["mcp"].(schema.SingleNestedAttribute)
	check := func(name, value string, wantErr bool) {
		t.Helper()
		attribute := mcp.Attributes[name].(schema.StringAttribute)
		for _, v := range attribute.Validators {
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(value)}, resp)
			if resp.Diagnostics.HasError() != wantErr {
				t.Errorf("%s=%q: error=%v, want %v", name, value, resp.Diagnostics.HasError(), wantErr)
			}
		}
	}
	check("config_delivery", "HEADER", false)
	check("config_delivery", "META", false)
	check("config_delivery", "BODY", true)
	check("transport", "HTTP", false)
	check("transport", "SSE", false)
	check("transport", "GRPC", true)
}

func TestApplicationRoutesRoundTripAndCoreDefaults(t *testing.T) {
	core := &fakeApplicationCore{transform: coreFillsNestedDefaults}
	r := &applicationResource{client: core.client(t)}

	plan := newApplicationPlan("http://e")
	plan.Routes = types.MapValueMust(types.ObjectType{AttrTypes: appRouteAttrTypes}, map[string]attr.Value{"r1": appRouteValue("/a")})
	resp, state := runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	route := core.body(t)["routes"].(map[string]any)["r1"].(map[string]any)
	if route["response"].(map[string]any)["body"] != "hi" || route["paths"].([]any)[0] != "/a" {
		t.Errorf("route body = %v", route)
	}
	for _, key := range []string{"order", "rewritePath", "maxRetryAttempts", "attachmentPaths", "upstreams", "methods"} {
		if _, ok := route[key]; ok {
			t.Errorf("unknown %s must not be sent: %v", key, route)
		}
	}
	got := state.Routes.Elements()["r1"].(types.Object).Attributes()
	if got["order"].(types.Int64).ValueInt64() != 2147483647 || got["max_retry_attempts"].(types.Int64).ValueInt64() != 1 ||
		got["rewrite_path"].(types.Bool).ValueBool() || got["upstreams"].(types.List).IsNull() || len(got["upstreams"].(types.List).Elements()) != 0 {
		t.Fatalf("Core defaults not read back: %v", got)
	}
	for _, key := range []string{"id", "name", "status"} {
		if _, ok := got[key]; ok {
			t.Errorf("route entry must not have %q", key)
		}
	}
}

func TestApplicationRoutesEntryRemovedAndEmptyMap(t *testing.T) {
	core := &fakeApplicationCore{transform: coreFillsNestedDefaults}
	r := &applicationResource{client: core.client(t)}
	objType := types.ObjectType{AttrTypes: appRouteAttrTypes}

	plan := newApplicationPlan("http://e")
	plan.Routes = types.MapValueMust(objType, map[string]attr.Value{"r1": appRouteValue("/a"), "r2": appRouteValue("/b")})
	if resp, _ := runApplicationCreate(t, r, plan); resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	plan.Routes = types.MapValueMust(objType, map[string]attr.Value{"r1": appRouteValue("/a")})
	resp, state := runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	if routes := core.body(t)["routes"].(map[string]any); len(routes) != 1 || len(state.Routes.Elements()) != 1 {
		t.Errorf("removed route still present: body %v state %v", routes, state.Routes)
	}

	plan.Routes = types.MapValueMust(objType, map[string]attr.Value{})
	resp, state = runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("clear: %v", resp.Diagnostics)
	}
	if state.Routes.IsNull() || len(state.Routes.Elements()) != 0 {
		t.Errorf("routes = {} must stay an empty map, got %v", state.Routes)
	}
}

func appServiceValue(secret string) attr.Value {
	auth := types.ObjectValueMust(appAuthAttrTypes, map[string]attr.Value{
		"authentication_type":    types.StringValue("OAUTH"),
		"client_id":              types.StringValue("cid"),
		"client_secret":          types.StringValue(secret),
		"authorization_endpoint": types.StringValue("http://a"),
		"token_endpoint":         types.StringValue("http://t"),
		"scopes_supported":       types.SetValueMust(types.StringType, []attr.Value{types.StringValue("s")}),
		"api_key_header":         types.StringNull(),
		"code_verifier":          types.StringNull(),
	})
	return types.ObjectValueMust(appServiceAttrTypes, map[string]attr.Value{
		"display_name": types.StringValue("S"), "description": types.StringNull(), "auth_settings": auth,
	})
}

func TestApplicationExternalServiceSecretIsKeptFromState(t *testing.T) {
	core := &fakeApplicationCore{transform: coreFillsNestedDefaults}
	r := &applicationResource{client: core.client(t)}
	objType := types.ObjectType{AttrTypes: appServiceAttrTypes}

	plan := newApplicationPlan("http://e")
	plan.AllowUserExternalServices = types.BoolValue(true)
	plan.ExternalServices = types.MapValueMust(objType, map[string]attr.Value{"svc": appServiceValue("secret-12345")})
	resp, state := runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	sent := core.body(t)["external_services"].(map[string]any)["svc"].(map[string]any)["auth_settings"].(map[string]any)
	if sent["client_secret"] != "secret-12345" || sent["authentication_type"] != "OAUTH" {
		t.Errorf("auth_settings sent = %v", sent)
	}
	if !state.ExternalServices.Equal(plan.ExternalServices) {
		t.Fatalf("state must equal the plan (secret kept):\n got %v\nwant %v", state.ExternalServices, plan.ExternalServices)
	}

	// A refresh keeps the secret too, and a changed secret is sent again.
	_, refreshed := runApplicationRead(t, r, state)
	if !refreshed.ExternalServices.Equal(plan.ExternalServices) {
		t.Errorf("refresh lost the secret: %v", refreshed.ExternalServices)
	}
	plan.ExternalServices = types.MapValueMust(objType, map[string]attr.Value{"svc": appServiceValue("another-secret")})
	if resp, _ := runApplicationCreate(t, r, plan); resp.Diagnostics.HasError() {
		t.Fatalf("update: %v", resp.Diagnostics)
	}
	sent = core.body(t)["external_services"].(map[string]any)["svc"].(map[string]any)["auth_settings"].(map[string]any)
	if sent["client_secret"] != "another-secret" {
		t.Errorf("changed secret not sent: %v", sent)
	}
}

func TestApplicationExternalServiceSecretNullAfterImport(t *testing.T) {
	core := &fakeApplicationCore{transform: coreFillsNestedDefaults}
	r := &applicationResource{client: core.client(t)}
	objType := types.ObjectType{AttrTypes: appServiceAttrTypes}

	plan := newApplicationPlan("http://e")
	plan.ExternalServices = types.MapValueMust(objType, map[string]attr.Value{"svc": appServiceValue("secret-12345")})
	if resp, _ := runApplicationCreate(t, r, plan); resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	imported := newApplicationPlan("")
	imported.Endpoint = types.StringNull()
	imported.ExternalServices = types.MapNull(objType)
	imported.Routes = types.MapNull(types.ObjectType{AttrTypes: appRouteAttrTypes})
	_, state := runApplicationRead(t, r, applicationModel{
		ID: types.StringValue("platform/f"), Bucket: types.StringValue("platform"), Path: types.StringValue("f"),
		Name: types.StringNull(), APIName: types.StringNull(), applicationBody: imported.applicationBody,
	})
	auth := state.ExternalServices.Elements()["svc"].(types.Object).Attributes()["auth_settings"].(types.Object).Attributes()
	if !auth["client_secret"].(types.String).IsNull() || !auth["code_verifier"].(types.String).IsNull() {
		t.Errorf("secrets must be null after import: %v", auth)
	}
	if auth["client_id"].(types.String).ValueString() != "cid" {
		t.Errorf("non-secret fields must be read: %v", auth)
	}
	for _, runtime := range []string{"client_secret_hint", "user_level_auth_status", "app_level_auth_status"} {
		if _, ok := auth[runtime]; ok {
			t.Errorf("runtime field %q must not be exposed", runtime)
		}
	}
}

func TestApplicationExternalServiceSecretValidation(t *testing.T) {
	services := applicationNestedResourceAttributes()["external_services"].(schema.MapNestedAttribute)
	auth := services.NestedObject.Attributes["auth_settings"].(schema.SingleNestedAttribute)
	secret := auth.Attributes["client_secret"].(schema.StringAttribute)
	if !secret.Sensitive || !auth.Attributes["code_verifier"].(schema.StringAttribute).Sensitive {
		t.Error("client_secret and code_verifier must be sensitive")
	}
	for value, wantErr := range map[string]bool{"short": true, "12345678": false} {
		resp := &validator.StringResponse{}
		for _, v := range secret.Validators {
			v.ValidateString(context.Background(), validator.StringRequest{ConfigValue: types.StringValue(value)}, resp)
		}
		if resp.Diagnostics.HasError() != wantErr {
			t.Errorf("client_secret %q: error=%v, want %v", value, resp.Diagnostics.HasError(), wantErr)
		}
	}
}

func TestApplicationDescriptionNamesUnmanagedFunction(t *testing.T) {
	description := applicationResourceSchema(t).Schema.MarkdownDescription
	if !strings.Contains(description, "`function`") || !strings.Contains(description, "dropped") {
		t.Errorf("description must name function as unmanaged and dropped on apply: %s", description)
	}
}
