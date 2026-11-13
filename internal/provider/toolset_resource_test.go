package provider

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// fakeToolsetCore imitates the parts of DIAL Core 0.50.1 the toolset resource
// depends on, as observed against a real Core: bodies use snake_case keys,
// GET fills in defaults for everything that was not sent (including a freshly
// generated `reference` on every PUT that lacks one), the author is set by the
// server, and the write-only client_secret comes back as a hint.
type fakeToolsetCore struct {
	mu     sync.Mutex
	stored []byte
	// received is the last PUT body exactly as sent, before the server adds defaults.
	received []byte
	puts     int
	paths    []string
	// extra is merged into every GET response (server-added fields).
	extra map[string]any
	// noDefaults turns the server-side defaults off, for tests of pure echo.
	noDefaults bool
}

func (f *fakeToolsetCore) client(t *testing.T) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.paths = append(f.paths, r.Method+" "+r.URL.EscapedPath())
		switch r.Method {
		case http.MethodPut:
			f.stored, _ = io.ReadAll(r.Body)
			f.received = append([]byte(nil), f.stored...)
			f.puts++
			var doc map[string]any
			_ = json.Unmarshal(f.stored, &doc)
			if _, sent := doc["reference"]; !sent && !f.noDefaults {
				doc["reference"] = "generated-" + strconv.Itoa(f.puts)
				f.stored, _ = json.Marshal(doc)
			}
			_, _ = w.Write([]byte(`{"name":"toolsets/platform/t"}`))
		case http.MethodGet:
			if f.stored == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var doc map[string]any
			_ = json.Unmarshal(f.stored, &doc)
			doc["name"] = "toolsets/platform/t"
			if !f.noDefaults {
				doc["author"] = "ci"
				for key, value := range map[string]any{
					"forward_auth_token": false, "forward_per_request_key": false, "max_retry_attempts": 1, "transport": "HTTP",
					"interceptors": []any{}, "description_keywords": []any{}, "dependencies": []any{}, "allowed_tools": []any{},
					"defaults": map[string]any{}, "responses_defaults": map[string]any{},
				} {
					if _, set := doc[key]; !set {
						doc[key] = value
					}
				}
				if _, set := doc["auth_settings"]; !set {
					doc["auth_settings"] = map[string]any{"authentication_type": "NONE"}
				}
				doc["auth_settings"].(map[string]any)["global_auth_status"] = "SIGNED_OUT"
				doc["auth_settings"].(map[string]any)["user_level_auth_status"] = "SIGNED_OUT"
			}
			if auth, ok := doc["auth_settings"].(map[string]any); ok {
				if _, had := auth["client_secret"]; had {
					delete(auth, "client_secret")
					auth["client_secret_hint"] = "s3…"
				}
			}
			for k, v := range f.extra {
				doc[k] = v
			}
			_ = json.NewEncoder(w).Encode(doc)
		case http.MethodDelete:
			f.stored = nil
		}
	}))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(srv.URL, client.APIKeyCredentials("k"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (f *fakeToolsetCore) body(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(f.received, &doc); err != nil {
		t.Fatalf("received body is not JSON: %v", err)
	}
	return doc
}

func toolsetResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	var resp resource.SchemaResponse
	(&toolsetResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp
}

// newToolsetPlan is a plan with the identity set, computed attributes
// unknown and everything else null, as Terraform produces for a minimal
// configuration.
func newToolsetPlan(bucket, path string) toolsetModel {
	interfaceType := types.ObjectType{AttrTypes: modelInterfaceAttrTypes}
	return toolsetModel{
		ID:      types.StringUnknown(),
		Bucket:  types.StringValue(bucket),
		Path:    types.StringValue(path),
		Name:    types.StringUnknown(),
		APIName: types.StringUnknown(),
		toolsetBody: toolsetBody{
			DisplayName: types.StringNull(), Description: types.StringNull(), Author: types.StringUnknown(),
			Intro: types.StringNull(), DisplayVersion: types.StringNull(), IconURL: types.StringNull(),
			Reference: types.StringUnknown(), ProviderName: types.StringNull(), VendorWebsite: types.StringNull(),
			Endpoint: types.StringNull(), ResponsesEndpoint: types.StringNull(), BaseURL: types.StringNull(),
			OverrideName: types.StringNull(), CatalogSchemaID: types.StringNull(), Transport: types.StringUnknown(),
			ForwardAuthToken: types.BoolUnknown(), ForwardPerRequestKey: types.BoolUnknown(),
			MaxInputAttachments: types.Int64Null(), MaxRetryAttempts: types.Int64Unknown(),
			AllowedTools: types.SetUnknown(types.StringType), Dependencies: types.SetUnknown(types.StringType),
			DescriptionKeywords: types.SetUnknown(types.StringType), InputAttachmentTypes: types.SetNull(types.StringType),
			Interceptors: types.SetUnknown(types.StringType), UserRoles: types.SetNull(types.StringType),
			DefaultHeaders:    types.MapUnknown(types.StringType),
			Features:          types.ObjectNull(modelFeaturesAttrTypes),
			Interfaces:        types.MapUnknown(interfaceType),
			AuthSettings:      types.ObjectUnknown(toolsetAuthSettingsAttrTypes),
			Defaults:          types.DynamicNull(),
			ResponsesDefaults: types.DynamicNull(),
			CatalogProperties: types.DynamicNull(),
		},
	}
}

func stringSet(t *testing.T, values ...string) types.Set {
	t.Helper()
	elements := make([]attr.Value, len(values))
	for i, v := range values {
		elements[i] = types.StringValue(v)
	}
	return types.SetValueMust(types.StringType, elements)
}

func authObject(t *testing.T, values map[string]attr.Value) types.Object {
	t.Helper()
	all := map[string]attr.Value{}
	for name, typ := range toolsetAuthSettingsAttrTypes {
		switch typ {
		case types.StringType:
			all[name] = types.StringNull()
		case types.BoolType:
			all[name] = types.BoolNull()
		default:
			all[name] = types.SetNull(types.StringType)
		}
	}
	for name, v := range values {
		all[name] = v
	}
	return types.ObjectValueMust(toolsetAuthSettingsAttrTypes, all)
}

func runToolsetCreate(t *testing.T, r *toolsetResource, plan toolsetModel) (*resource.CreateResponse, toolsetModel) {
	t.Helper()
	ctx := context.Background()
	schema := toolsetResourceSchema(t).Schema
	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	if diags := req.Plan.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("plan: %v", diags)
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, req, resp)
	var state toolsetModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("state: %v", diags)
		}
	}
	return resp, state
}

func runToolsetRead(t *testing.T, r *toolsetResource, prior toolsetModel) (*resource.ReadResponse, toolsetModel) {
	t.Helper()
	ctx := context.Background()
	schema := toolsetResourceSchema(t).Schema
	state := tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, &prior); diags.HasError() {
		t.Fatalf("prior state: %v", diags)
	}
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	var out toolsetModel
	if !resp.Diagnostics.HasError() && !resp.State.Raw.IsNull() {
		if diags := resp.State.Get(ctx, &out); diags.HasError() {
			t.Fatalf("state: %v", diags)
		}
	}
	return resp, out
}

func TestToolsetSchemasAreValidAndMatchModels(t *testing.T) {
	ctx := context.Background()

	rs := toolsetResourceSchema(t).Schema
	if diags := rs.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("resource schema: %v", diags)
	}
	// The model struct must be convertible to and from the schema, which also
	// proves the embedded body struct lines up with the attributes.
	plan := newToolsetPlan("platform", "t")
	state := tfsdk.State{Schema: rs, Raw: tftypes.NewValue(rs.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("resource model does not match schema: %v", diags)
	}

	var api datasource.SchemaResponse
	(&toolsetDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &api)
	if diags := api.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("data source schema: %v", diags)
	}
	var file datasource.SchemaResponse
	(&fileToolsetDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &file)
	if diags := file.Schema.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("file data source schema: %v", diags)
	}

	if _, ok := rs.Attributes["provider"]; ok {
		t.Fatal("`provider` is a reserved attribute name; the API field is exposed as provider_name")
	}
}

func TestToolsetTypesRegistered(t *testing.T) {
	ctx := context.Background()
	resources := map[string]bool{}
	for _, f := range (&AidialProvider{}).Resources(ctx) {
		resp := &resource.MetadataResponse{}
		f().Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "aidial"}, resp)
		resources[resp.TypeName] = true
	}
	sources := map[string]bool{}
	for _, f := range (&AidialProvider{}).DataSources(ctx) {
		resp := &datasource.MetadataResponse{}
		f().Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "aidial"}, resp)
		sources[resp.TypeName] = true
	}
	if !resources["aidial_toolset"] {
		t.Error("aidial_toolset resource is not registered")
	}
	for _, name := range []string{"aidial_toolset", "aidial_file_toolset"} {
		if !sources[name] {
			t.Errorf("%s data source is not registered", name)
		}
	}
}

func TestToolsetIdentityValidation(t *testing.T) {
	rs := toolsetResourceSchema(t).Schema
	bucket, path := identityFromResourceSchema(t, "bucket", rs), identityFromResourceSchema(t, "path", rs)
	if !bucket.required || bucket.optional || bucket.computed || !path.required || path.optional || path.computed {
		t.Fatalf("bucket and path must be required only: %+v %+v", bucket, path)
	}
	for _, ok := range []string{"platform", "public", "some-user-bucket"} {
		if !validate(t, bucket.validators, ok) {
			t.Errorf("bucket %q must be accepted", ok)
		}
	}
	if validate(t, bucket.validators, "") || validate(t, path.validators, "") {
		t.Error("empty bucket or path must be rejected")
	}
	if !validate(t, path.validators, "team/tool") {
		t.Error("nested path must be accepted")
	}
}

func TestToolsetMinimalCreateSendsOnlyConfiguredFields(t *testing.T) {
	core := &fakeToolsetCore{}
	r := &toolsetResource{client: core.client(t)}

	plan := newToolsetPlan("public", "team/t")
	plan.Endpoint = types.StringValue("http://tool/mcp")
	resp, state := runToolsetCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	body := core.body(t)
	if len(body) != 1 || body["endpoint"] != "http://tool/mcp" {
		t.Fatalf("expected only endpoint in the request body (computed attributes are not sent), got %v", body)
	}
	if want := "PUT /v1/toolsets/public/team/t"; core.paths[0] != want {
		t.Fatalf("first request = %q, want %q", core.paths[0], want)
	}
	if core.paths[1] != "GET /v1/toolsets/public/team/t" {
		t.Fatalf("create must re-read with GET, requests: %v", core.paths)
	}
	if state.ID.ValueString() != "public/team/t" {
		t.Errorf("id = %v", state.ID)
	}
	if state.APIName.ValueString() != "toolsets/platform/t" || state.Name.ValueString() != state.APIName.ValueString() {
		t.Errorf("name = %v, api_name = %v", state.Name, state.APIName)
	}
	// Everything Core defaults is known after apply.
	if state.Reference.ValueString() != "generated-1" || state.Author.ValueString() != "ci" || state.Transport.ValueString() != "HTTP" ||
		state.MaxRetryAttempts.ValueInt64() != 1 || state.ForwardAuthToken.IsNull() || state.ForwardAuthToken.ValueBool() {
		t.Errorf("Core defaults not read back: %+v", state.toolsetBody)
	}
	for name, set := range map[string]types.Set{"allowed_tools": state.AllowedTools, "dependencies": state.Dependencies, "interceptors": state.Interceptors, "description_keywords": state.DescriptionKeywords} {
		if set.IsNull() || set.IsUnknown() || len(set.Elements()) != 0 {
			t.Errorf("%s must be the empty set Core defaults to, got %v", name, set)
		}
	}
	// Collections Core leaves out stay null; the maps are empty, as on aidial_model.
	for name, set := range map[string]types.Set{"user_roles": state.UserRoles, "input_attachment_types": state.InputAttachmentTypes} {
		if !set.IsNull() {
			t.Errorf("%s must be null, got %v", name, set)
		}
	}
	if !state.Features.IsNull() || !state.Defaults.IsNull() || !state.CatalogProperties.IsNull() {
		t.Errorf("features, defaults and catalog_properties must be null")
	}
	if state.AuthSettings.IsNull() || state.AuthSettings.IsUnknown() || state.AuthSettings.Attributes()["authentication_type"] != types.StringValue("NONE") {
		t.Errorf("auth_settings must hold Core's default, got %v", state.AuthSettings)
	}
	if state.DefaultHeaders.IsNull() || len(state.DefaultHeaders.Elements()) != 0 || state.Interfaces.IsNull() || len(state.Interfaces.Elements()) != 0 {
		t.Errorf("default_headers and interfaces must be empty maps, got %v / %v", state.DefaultHeaders, state.Interfaces)
	}
}

func TestToolsetFullCreateRoundTrips(t *testing.T) {
	core := &fakeToolsetCore{}
	r := &toolsetResource{client: core.client(t)}

	plan := newToolsetPlan("platform", "t")
	plan.DisplayName, plan.Description = types.StringValue("Tool"), types.StringValue("d")
	plan.Intro, plan.DisplayVersion, plan.IconURL = types.StringValue("hi"), types.StringValue("1.0"), types.StringValue("http://i")
	plan.Reference, plan.ProviderName, plan.VendorWebsite = types.StringValue("http://r"), types.StringValue("Acme"), types.StringValue("http://v")
	plan.Endpoint, plan.ResponsesEndpoint, plan.BaseURL = types.StringValue("http://e"), types.StringValue("http://re"), types.StringValue("http://b")
	plan.OverrideName, plan.CatalogSchemaID, plan.Transport = types.StringValue("over"), types.StringValue("https://s/1"), types.StringValue("SSE")
	plan.ForwardAuthToken, plan.ForwardPerRequestKey = types.BoolValue(true), types.BoolValue(false)
	plan.MaxInputAttachments, plan.MaxRetryAttempts = types.Int64Value(3), types.Int64Value(2)
	plan.AllowedTools = stringSet(t, "b", "a")
	plan.UserRoles = stringSet(t, "admin")
	plan.AuthSettings = authObject(t, map[string]attr.Value{
		"authentication_type": types.StringValue("OAUTH"),
		"client_id":           types.StringValue("cid"),
		"client_secret":       types.StringValue("s3cr3t"),
		"scopes_supported":    stringSet(t, "read"),
	})
	plan.Defaults = types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"b": types.NumberType, "a": types.StringType},
		map[string]attr.Value{"b": types.NumberValue(bigInt(1)), "a": types.StringValue("x")},
	))

	resp, state := runToolsetCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	body := core.body(t)
	for key, want := range map[string]any{
		"display_name": "Tool", "description": "d", "intro": "hi", "display_version": "1.0",
		"icon_url": "http://i", "reference": "http://r", "provider": "Acme", "vendor_website": "http://v",
		"endpoint": "http://e", "responses_endpoint": "http://re", "base_url": "http://b", "override_name": "over",
		"catalog_schema_id": "https://s/1", "transport": "SSE", "forward_auth_token": true, "forward_per_request_key": false,
		"max_input_attachments": float64(3), "max_retry_attempts": float64(2),
	} {
		if body[key] != want {
			t.Errorf("body[%s] = %v, want %v", key, body[key], want)
		}
	}
	if tools, _ := body["allowed_tools"].([]any); len(tools) != 2 {
		t.Errorf("allowed_tools = %v", body["allowed_tools"])
	}
	auth, _ := body["auth_settings"].(map[string]any)
	if auth["authentication_type"] != "OAUTH" || auth["client_id"] != "cid" || auth["client_secret"] != "s3cr3t" {
		t.Errorf("auth_settings = %v", body["auth_settings"])
	}
	if defaults, _ := body["defaults"].(map[string]any); defaults["a"] != "x" || defaults["b"] != float64(1) {
		t.Errorf("defaults = %v", body["defaults"])
	}

	if state.ProviderName.ValueString() != "Acme" || state.Transport.ValueString() != "SSE" {
		t.Errorf("state = %v / %v", state.ProviderName, state.Transport)
	}
	if !state.AllowedTools.Equal(plan.AllowedTools) {
		t.Errorf("allowed_tools = %v", state.AllowedTools)
	}
	if !state.Defaults.Equal(plan.Defaults) {
		t.Errorf("free-form defaults must keep the configured value, got %v", state.Defaults)
	}
	// Core returned a hint, not the secret: state keeps the configured value.
	var a toolsetAuthSettingsModel
	if diags := state.AuthSettings.As(context.Background(), &a, basetypesObjectOptions); diags.HasError() {
		t.Fatalf("auth_settings: %v", diags)
	}
	if a.ClientSecret.ValueString() != "s3cr3t" || a.ClientID.ValueString() != "cid" {
		t.Errorf("auth_settings = %+v", a)
	}
}

func TestToolsetClientSecretKeptOnRefreshAndNullAfterImport(t *testing.T) {
	core := &fakeToolsetCore{}
	r := &toolsetResource{client: core.client(t)}

	plan := newToolsetPlan("platform", "t")
	plan.AuthSettings = authObject(t, map[string]attr.Value{
		"authentication_type": types.StringValue("OAUTH"),
		"client_secret":       types.StringValue("s3cr3t"),
	})
	if resp, _ := runToolsetCreate(t, r, plan); resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	_, created := runToolsetCreate(t, r, plan) // re-create to get a fully known model

	resp, refreshed := runToolsetRead(t, r, created)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	var a toolsetAuthSettingsModel
	refreshed.AuthSettings.As(context.Background(), &a, basetypesObjectOptions)
	if a.ClientSecret.ValueString() != "s3cr3t" {
		t.Fatalf("client_secret must survive refresh, got %v", a.ClientSecret)
	}

	// Import: identity only, no prior auth_settings.
	imported := newToolsetPlan("platform", "t")
	imported.ID, imported.Name = types.StringValue("platform/t"), types.StringNull()
	imported.DefaultHeaders = types.MapNull(types.StringType)
	imported.Interfaces = types.MapNull(types.ObjectType{AttrTypes: modelInterfaceAttrTypes})
	resp, afterImport := runToolsetRead(t, r, imported)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read after import: %v", resp.Diagnostics)
	}
	var b toolsetAuthSettingsModel
	afterImport.AuthSettings.As(context.Background(), &b, basetypesObjectOptions)
	if !b.ClientSecret.IsNull() || b.AuthenticationType.ValueString() != "OAUTH" {
		t.Fatalf("client_secret must be null after import, got %+v", b)
	}
}

func TestToolsetRuntimeAuthStatusIsIgnored(t *testing.T) {
	core := &fakeToolsetCore{extra: map[string]any{"auth_settings": map[string]any{
		"authentication_type":    "OAUTH",
		"user_level_auth_status": "SIGNED_OUT",
		"global_auth_status":     "SIGNED_IN",
		"code_verifier":          "v",
		"client_secret_hint":     "abc",
	}}}
	r := &toolsetResource{client: core.client(t)}

	plan := newToolsetPlan("platform", "t")
	plan.AuthSettings = authObject(t, map[string]attr.Value{"authentication_type": types.StringValue("OAUTH")})
	resp, state := runToolsetCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	if !state.AuthSettings.Equal(plan.AuthSettings) {
		t.Fatalf("runtime status fields must not change auth_settings:\n got %v\nwant %v", state.AuthSettings, plan.AuthSettings)
	}
}

func TestToolsetConfiguredEmptySetIsSentAndUnsetOptionalStaysNull(t *testing.T) {
	core := &fakeToolsetCore{}
	r := &toolsetResource{client: core.client(t)}

	plan := newToolsetPlan("platform", "t")
	plan.UserRoles = types.SetValueMust(types.StringType, []attr.Value{})
	plan.AllowedTools = stringSet(t, "a")
	resp, state := runToolsetCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	body := core.body(t)
	if roles, ok := body["user_roles"].([]any); !ok || len(roles) != 0 {
		t.Fatalf("an empty set must be sent as [], got %v", body["user_roles"])
	}
	if _, sent := body["dependencies"]; sent {
		t.Fatalf("an unknown (computed) set must not be sent, got %v", body["dependencies"])
	}
	if state.UserRoles.IsNull() || len(state.UserRoles.Elements()) != 0 {
		t.Fatalf("configured empty set must stay empty, got %v", state.UserRoles)
	}
	if !state.InputAttachmentTypes.IsNull() {
		t.Fatalf("unset optional set must stay null, got %v", state.InputAttachmentTypes)
	}
	if !state.AllowedTools.Equal(plan.AllowedTools) {
		t.Fatalf("configured allowed_tools = %v", state.AllowedTools)
	}
}

// Core regenerates `reference` on any PUT that does not carry it. Updates
// must therefore send the value kept in state, so it stays stable.
func TestToolsetUpdateKeepsCoreGeneratedValuesStable(t *testing.T) {
	core := &fakeToolsetCore{}
	r := &toolsetResource{client: core.client(t)}
	ctx := context.Background()

	resp, created := runToolsetCreate(t, r, newToolsetPlan("platform", "t"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	ref := created.Reference.ValueString()

	// UseStateForUnknown makes the plan carry the known state values.
	next := created
	next.Description = types.StringValue("changed")
	schema := toolsetResourceSchema(t).Schema
	req := resource.UpdateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	if diags := req.Plan.Set(ctx, &next); diags.HasError() {
		t.Fatal(diags)
	}
	upd := &resource.UpdateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.Update(ctx, req, upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("update: %v", upd.Diagnostics)
	}
	var after toolsetModel
	upd.State.Get(ctx, &after)
	if after.Reference.ValueString() != ref {
		t.Fatalf("reference changed across an update: %q -> %q", ref, after.Reference.ValueString())
	}
	if _, sent := core.body(t)["author"]; sent {
		t.Fatalf("author is server-set and must not be sent: %v", core.body(t))
	}
}

func TestToolsetFreeFormKeyOrderIsNoOp(t *testing.T) {
	core := &fakeToolsetCore{}
	r := &toolsetResource{client: core.client(t)}

	plan := newToolsetPlan("platform", "t")
	plan.CatalogProperties = types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"z": types.StringType, "a": types.StringType},
		map[string]attr.Value{"z": types.StringValue("1"), "a": types.StringValue("2")},
	))
	if resp, _ := runToolsetCreate(t, r, plan); resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	_, created := runToolsetCreate(t, r, plan)
	// Core returns keys in a different order than they were configured.
	core.mu.Lock()
	core.stored = []byte(`{"catalog_properties":{"a":"2","z":"1"}}`)
	core.mu.Unlock()
	resp, refreshed := runToolsetRead(t, r, created)
	if resp.Diagnostics.HasError() {
		t.Fatalf("read: %v", resp.Diagnostics)
	}
	if !refreshed.CatalogProperties.Equal(plan.CatalogProperties) {
		t.Fatalf("configured value must be kept for equal JSON, got %v", refreshed.CatalogProperties)
	}
}

func TestToolsetUpdateIsInPlaceAndOutOfBandDeletionRemovesState(t *testing.T) {
	core := &fakeToolsetCore{}
	r := &toolsetResource{client: core.client(t)}
	ctx := context.Background()

	plan := newToolsetPlan("platform", "t")
	plan.Description = types.StringValue("one")
	resp, created := runToolsetCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	schema := toolsetResourceSchema(t).Schema
	next := created
	next.Description = types.StringValue("two")
	req := resource.UpdateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	if diags := req.Plan.Set(ctx, &next); diags.HasError() {
		t.Fatal(diags)
	}
	upd := &resource.UpdateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	putsBefore := core.puts
	r.Update(ctx, req, upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("update: %v", upd.Diagnostics)
	}
	if core.puts != putsBefore+1 || core.body(t)["description"] != "two" {
		t.Fatalf("update must PUT the new body, puts=%d body=%v", core.puts, core.body(t))
	}

	core.mu.Lock()
	core.stored = nil
	core.mu.Unlock()
	readResp, _ := runToolsetRead(t, r, created)
	if readResp.Diagnostics.HasError() || !readResp.State.Raw.IsNull() {
		t.Fatalf("a missing toolset must be removed from state, diags=%v", readResp.Diagnostics)
	}
}

func TestToolsetDeleteIsIdempotent(t *testing.T) {
	core := &fakeToolsetCore{}
	r := &toolsetResource{client: core.client(t)}
	ctx := context.Background()
	schema := toolsetResourceSchema(t).Schema

	state := newToolsetPlan("platform", "t")
	state.ID, state.Name, state.APIName = types.StringValue("platform/t"), types.StringValue("t"), types.StringValue("t")
	state.DefaultHeaders = types.MapNull(types.StringType)
	state.Interfaces = types.MapNull(types.ObjectType{AttrTypes: modelInterfaceAttrTypes})
	req := resource.DeleteRequest{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	if diags := req.State.Set(ctx, &state); diags.HasError() {
		t.Fatal(diags)
	}
	resp := &resource.DeleteResponse{}
	r.Delete(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("delete of an absent toolset must succeed: %v", resp.Diagnostics)
	}
	if core.paths[len(core.paths)-1] != "DELETE /v1/toolsets/platform/t" {
		t.Fatalf("requests: %v", core.paths)
	}
}

func TestToolsetImportState(t *testing.T) {
	ctx := context.Background()
	schema := toolsetResourceSchema(t).Schema
	r := &toolsetResource{}

	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "public/team/tool"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import: %v", resp.Diagnostics)
	}
	var bucket, path, id types.String
	resp.State.GetAttribute(ctx, pathRoot("bucket"), &bucket)
	resp.State.GetAttribute(ctx, pathRoot("path"), &path)
	resp.State.GetAttribute(ctx, pathRoot("id"), &id)
	if bucket.ValueString() != "public" || path.ValueString() != "team/tool" || id.ValueString() != "public/team/tool" {
		t.Fatalf("bucket=%v path=%v id=%v", bucket, path, id)
	}

	for _, bad := range []string{"nope", "/path", "bucket/", ""} {
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
		r.ImportState(ctx, resource.ImportStateRequest{ID: bad}, resp)
		if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "<bucket>/<path>") {
			t.Errorf("import id %q must fail naming the expected format, got %v", bad, resp.Diagnostics)
		}
	}
}

// features and interfaces are shared with aidial_model: the same planned
// values must produce the same API JSON from both resources.
func TestToolsetFeaturesAndInterfacesMatchModel(t *testing.T) {
	ctx := context.Background()
	features := types.ObjectValueMust(modelFeaturesAttrTypes, func() map[string]attr.Value {
		values := map[string]attr.Value{}
		for name, typ := range modelFeaturesAttrTypes {
			switch typ {
			case types.BoolType:
				values[name] = types.BoolValue(true)
			case types.StringType:
				values[name] = types.StringValue("http://x/" + name)
			default:
				values[name] = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("low")})
			}
		}
		return values
	}())
	interfaces := types.MapValueMust(types.ObjectType{AttrTypes: modelInterfaceAttrTypes}, map[string]attr.Value{
		"openaiChatCompletions": types.ObjectValueMust(modelInterfaceAttrTypes, map[string]attr.Value{
			"base_url":        types.StringValue("http://i"),
			"mode":            types.StringValue("translator"),
			"override_paths":  types.MapValueMust(types.StringType, map[string]attr.Value{"/a": types.StringValue("/b")}),
			"default_headers": types.MapNull(types.StringType),
		}),
	})

	plan := newToolsetPlan("platform", "t")
	plan.Features, plan.Interfaces = features, interfaces
	ts, err := toolsetFromPlan(ctx, plan)
	if err != nil {
		t.Fatal(err)
	}

	model := modelResourceModel{Features: features, Interfaces: interfaces}
	m, err := modelFromPlan(ctx, model)
	if err != nil {
		t.Fatal(err)
	}

	for name, pair := range map[string][2]any{
		"features":   {ts.Features, m.Features},
		"interfaces": {ts.Interfaces, m.Interfaces},
	} {
		a, _ := json.Marshal(pair[0])
		b, _ := json.Marshal(pair[1])
		if string(a) != string(b) || string(a) == "null" {
			t.Errorf("%s differ between toolset and model:\n toolset: %s\n model:   %s", name, a, b)
		}
	}
}

func bigInt(n int64) *big.Float { return new(big.Float).SetInt64(n) }

var basetypesObjectOptions = basetypes.ObjectAsOptions{}

func pathRoot(name string) path.Path { return path.Root(name) }

// identityFromResourceSchema is resourceIdentity's per-attribute view for a
// schema already in hand.
func identityFromResourceSchema(t *testing.T, name string, s schema.Schema) identityAttr {
	t.Helper()
	a, ok := s.Attributes[name].(schema.StringAttribute)
	if !ok {
		t.Fatalf("%s is not a StringAttribute", name)
	}
	return identityAttr{a.Required, a.Optional, a.Computed, a.Validators}
}
