package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// fakeInterceptorCore imitates the parts of DIAL Core 0.50.1 the interceptor
// resource depends on, as observed against a real Core: camelCase bodies, and a
// GET that fills in forwardAuthToken false, maxRetryAttempts 1 and an empty
// defaults object for whatever was not sent.
type fakeInterceptorCore struct {
	mu       sync.Mutex
	stored   []byte
	received []byte
	paths    []string
}

func (f *fakeInterceptorCore) client(t *testing.T) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.paths = append(f.paths, r.Method+" "+r.URL.EscapedPath())
		switch r.Method {
		case http.MethodPut:
			f.stored, _ = io.ReadAll(r.Body)
			f.received = append([]byte(nil), f.stored...)
			_, _ = w.Write([]byte(`{"name":"f"}`))
		case http.MethodGet:
			if f.stored == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var doc map[string]any
			_ = json.Unmarshal(f.stored, &doc)
			doc["name"] = "f"
			for key, value := range map[string]any{"forwardAuthToken": false, "maxRetryAttempts": 1, "defaults": map[string]any{}, "status": "valid"} {
				if _, set := doc[key]; !set {
					doc[key] = value
				}
			}
			_ = json.NewEncoder(w).Encode(doc)
		case http.MethodDelete:
			if f.stored == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			f.stored = nil
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(srv.URL, client.APIKeyCredentials("k"))
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func (f *fakeInterceptorCore) body(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(f.received, &doc); err != nil {
		t.Fatalf("received body is not JSON: %v", err)
	}
	return doc
}

func interceptorResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	var resp resource.SchemaResponse
	(&interceptorResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp
}

// newInterceptorPlan is a plan with the identity set, computed attributes
// unknown and everything else null, as Terraform produces for a minimal
// configuration.
func newInterceptorPlan(endpoint string) interceptorModel {
	return interceptorModel{
		ID:      types.StringUnknown(),
		Bucket:  types.StringValue("platform"),
		Path:    types.StringValue("f"),
		Name:    types.StringUnknown(),
		APIName: types.StringUnknown(),
		interceptorBody: interceptorBody{
			Endpoint: types.StringValue(endpoint), Description: types.StringNull(), DisplayName: types.StringNull(),
			DisplayVersion: types.StringNull(), IconURL: types.StringNull(), CatalogSchemaID: types.StringNull(),
			ForwardAuthToken: types.BoolUnknown(), MaxRetryAttempts: types.Int64Unknown(),
			UserRoles:      types.SetNull(types.StringType),
			DefaultHeaders: types.MapUnknown(types.StringType),
			Features:       types.ObjectNull(modelFeaturesAttrTypes),
			Interfaces:     types.MapUnknown(types.ObjectType{AttrTypes: modelInterfaceAttrTypes}),
			Defaults:       types.DynamicNull(), CatalogProperties: types.DynamicNull(),
		},
	}
}

func newInterceptorState(t *testing.T, plan interceptorModel) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schema := interceptorResourceSchema(t).Schema
	state := tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	return state
}

func runInterceptorCreate(t *testing.T, r *interceptorResource, plan interceptorModel) (*resource.CreateResponse, interceptorModel) {
	t.Helper()
	ctx := context.Background()
	schema := interceptorResourceSchema(t).Schema
	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	if diags := req.Plan.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("plan: %v", diags)
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, req, resp)
	var state interceptorModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("state: %v", diags)
		}
	}
	return resp, state
}

func runInterceptorRead(t *testing.T, r *interceptorResource, prior interceptorModel) (*resource.ReadResponse, interceptorModel) {
	t.Helper()
	ctx := context.Background()
	state := newInterceptorState(t, prior)
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	var out interceptorModel
	if !resp.Diagnostics.HasError() && !resp.State.Raw.IsNull() {
		if diags := resp.State.Get(ctx, &out); diags.HasError() {
			t.Fatalf("state: %v", diags)
		}
	}
	return resp, out
}

func TestInterceptorSchemasAreValidAndMatchModels(t *testing.T) {
	ctx := context.Background()
	rs := interceptorResourceSchema(t).Schema
	if diags := rs.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("resource schema: %v", diags)
	}
	// The model struct must be convertible to and from the schema.
	state := newInterceptorState(t, newInterceptorPlan("http://e"))
	if state.Raw.IsNull() {
		t.Fatal("state is null")
	}

	var api, file datasource.SchemaResponse
	(&interceptorDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &api)
	(&fileInterceptorDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &file)
	for name, s := range map[string]datasource.SchemaResponse{"api": api, "file": file} {
		if diags := s.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Fatalf("%s data source schema: %v", name, diags)
		}
	}
	for name := range rs.Attributes {
		if name == "bucket" || name == "path" || name == "id" || name == "api_name" || name == "name" {
			continue
		}
		if _, ok := api.Schema.Attributes[name]; !ok {
			t.Errorf("aidial_interceptor data source lacks attribute %q of the resource", name)
		}
	}
	for _, name := range []string{"author", "created_at", "updated_at"} {
		if _, ok := rs.Attributes[name]; ok {
			t.Errorf("%s must not be exposed", name)
		}
	}
}

func TestInterceptorTypesRegistered(t *testing.T) {
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
	if !resources["aidial_interceptor"] {
		t.Error("aidial_interceptor resource is not registered")
	}
	for _, name := range []string{"aidial_interceptor", "aidial_file_interceptor"} {
		if !sources[name] {
			t.Errorf("%s data source is not registered", name)
		}
	}
}

func TestInterceptorEndpointIsRequired(t *testing.T) {
	rs := interceptorResourceSchema(t).Schema
	endpoint := identityFromResourceSchema(t, "endpoint", rs)
	if !endpoint.required || endpoint.optional || endpoint.computed {
		t.Fatalf("endpoint must be required only: %+v", endpoint)
	}
}

func TestInterceptorMinimalCreateSendsOnlyConfiguredFields(t *testing.T) {
	core := &fakeInterceptorCore{}
	r := &interceptorResource{client: core.client(t)}

	resp, state := runInterceptorCreate(t, r, newInterceptorPlan("http://hook/chat"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	body := core.body(t)
	if len(body) != 1 || body["endpoint"] != "http://hook/chat" {
		t.Fatalf("expected only endpoint in the request body (computed attributes are not sent), got %v", body)
	}
	if core.paths[0] != "PUT /v1/interceptors/platform/f" || core.paths[1] != "GET /v1/interceptors/platform/f" {
		t.Fatalf("create must PUT and re-read with GET, requests: %v", core.paths)
	}
	if state.ID.ValueString() != "platform/f" || state.APIName.ValueString() != "f" || state.Name.ValueString() != "f" {
		t.Errorf("id=%v api_name=%v name=%v", state.ID, state.APIName, state.Name)
	}
	if state.ForwardAuthToken.IsNull() || state.ForwardAuthToken.ValueBool() || state.MaxRetryAttempts.ValueInt64() != 1 {
		t.Errorf("Core defaults not read back: %v / %v", state.ForwardAuthToken, state.MaxRetryAttempts)
	}
	if !state.UserRoles.IsNull() || !state.Features.IsNull() || !state.CatalogProperties.IsNull() || !state.Description.IsNull() {
		t.Errorf("attributes Core omits must be null: %+v", state.interceptorBody)
	}
	if state.DefaultHeaders.IsNull() || len(state.DefaultHeaders.Elements()) != 0 || state.Interfaces.IsNull() || len(state.Interfaces.Elements()) != 0 {
		t.Errorf("default_headers and interfaces must be empty maps, got %v / %v", state.DefaultHeaders, state.Interfaces)
	}
	// Core returns defaults {} but the config left it unset: it stays null.
	if !state.Defaults.IsNull() {
		t.Errorf("an empty defaults object must read back as null when unconfigured, got %v", state.Defaults)
	}
}

func TestInterceptorFullCreateRoundTrips(t *testing.T) {
	core := &fakeInterceptorCore{}
	r := &interceptorResource{client: core.client(t)}

	plan := newInterceptorPlan("http://hook/chat")
	plan.Description, plan.DisplayName = types.StringValue("d"), types.StringValue("Hook")
	plan.DisplayVersion, plan.IconURL = types.StringValue("1.0"), types.StringValue("http://i")
	plan.CatalogSchemaID = types.StringValue("https://s/1")
	plan.ForwardAuthToken, plan.MaxRetryAttempts = types.BoolValue(true), types.Int64Value(3)
	plan.UserRoles = stringSet(t, "admin", "dev")
	plan.DefaultHeaders = types.MapValueMust(types.StringType, map[string]attr.Value{"X-A": types.StringValue("1")})
	plan.Features = types.ObjectValueMust(modelFeaturesAttrTypes, func() map[string]attr.Value {
		values := map[string]attr.Value{}
		for name, typ := range modelFeaturesAttrTypes {
			switch typ {
			case types.BoolType:
				values[name] = types.BoolNull()
			case types.StringType:
				values[name] = types.StringNull()
			default:
				values[name] = types.SetNull(types.StringType)
			}
		}
		values["tools_supported"] = types.BoolValue(true)
		return values
	}())
	plan.Interfaces = types.MapValueMust(types.ObjectType{AttrTypes: modelInterfaceAttrTypes}, map[string]attr.Value{
		"chat": types.ObjectValueMust(modelInterfaceAttrTypes, map[string]attr.Value{
			"base_url": types.StringValue("http://i"), "mode": types.StringNull(),
			"override_paths": types.MapNull(types.StringType), "default_headers": types.MapNull(types.StringType),
		}),
	})
	plan.Defaults = types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"b": types.NumberType, "a": types.StringType},
		map[string]attr.Value{"b": types.NumberValue(bigInt(1)), "a": types.StringValue("x")},
	))
	plan.CatalogProperties = types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"k": types.StringType}, map[string]attr.Value{"k": types.StringValue("v")},
	))

	resp, state := runInterceptorCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	body := core.body(t)
	for key, want := range map[string]any{
		"endpoint": "http://hook/chat", "description": "d", "displayName": "Hook", "displayVersion": "1.0",
		"iconUrl": "http://i", "catalogSchemaId": "https://s/1", "forwardAuthToken": true, "maxRetryAttempts": float64(3),
	} {
		if body[key] != want {
			t.Errorf("body[%s] = %v, want %v", key, body[key], want)
		}
	}
	if roles, _ := body["userRoles"].([]any); len(roles) != 2 {
		t.Errorf("userRoles = %v", body["userRoles"])
	}
	if h, _ := body["defaultHeaders"].(map[string]any); h["X-A"] != "1" {
		t.Errorf("defaultHeaders = %v", body["defaultHeaders"])
	}
	if f, _ := body["features"].(map[string]any); f["tools_supported"] != true {
		t.Errorf("features = %v", body["features"])
	}
	if i, _ := body["interfaces"].(map[string]any); i["chat"] == nil {
		t.Errorf("interfaces = %v", body["interfaces"])
	}
	if c, _ := body["catalogProperties"].(map[string]any); c["k"] != "v" {
		t.Errorf("catalogProperties = %v", body["catalogProperties"])
	}

	if !state.UserRoles.Equal(plan.UserRoles) || !state.DefaultHeaders.Equal(plan.DefaultHeaders) || !state.Features.Equal(plan.Features) {
		t.Errorf("collections did not round-trip: %+v", state.interceptorBody)
	}
	if !state.Defaults.Equal(plan.Defaults) || !state.CatalogProperties.Equal(plan.CatalogProperties) {
		t.Errorf("free-form values must keep the configured value, got %v / %v", state.Defaults, state.CatalogProperties)
	}
	if state.ForwardAuthToken.ValueBool() != true || state.MaxRetryAttempts.ValueInt64() != 3 {
		t.Errorf("configured scalars changed: %v / %v", state.ForwardAuthToken, state.MaxRetryAttempts)
	}
}

func TestInterceptorConfiguredEmptySetIsSentAndStaysEmpty(t *testing.T) {
	core := &fakeInterceptorCore{}
	r := &interceptorResource{client: core.client(t)}

	plan := newInterceptorPlan("http://e")
	plan.UserRoles = types.SetValueMust(types.StringType, []attr.Value{})
	resp, state := runInterceptorCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	if roles, ok := core.body(t)["userRoles"].([]any); !ok || len(roles) != 0 {
		t.Fatalf("an empty set must be sent as [], got %v", core.body(t)["userRoles"])
	}
	if state.UserRoles.IsNull() || len(state.UserRoles.Elements()) != 0 {
		t.Fatalf("configured empty set must stay empty, got %v", state.UserRoles)
	}
}

func TestInterceptorFreeFormKeyOrderIsNoOp(t *testing.T) {
	core := &fakeInterceptorCore{}
	r := &interceptorResource{client: core.client(t)}

	plan := newInterceptorPlan("http://e")
	plan.CatalogProperties = types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"z": types.StringType, "a": types.StringType},
		map[string]attr.Value{"z": types.StringValue("1"), "a": types.StringValue("2")},
	))
	resp, created := runInterceptorCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	_, refreshed := runInterceptorRead(t, r, created)
	if !refreshed.CatalogProperties.Equal(plan.CatalogProperties) {
		t.Fatalf("refresh changed free-form value: %v", refreshed.CatalogProperties)
	}
}

func TestInterceptorUpdateIsInPlaceAndOutOfBandDeletionRemovesState(t *testing.T) {
	core := &fakeInterceptorCore{}
	r := &interceptorResource{client: core.client(t)}
	ctx := context.Background()

	resp, created := runInterceptorCreate(t, r, newInterceptorPlan("http://e"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	next := created
	next.Description = types.StringValue("changed")
	schema := interceptorResourceSchema(t).Schema
	req := resource.UpdateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	if diags := req.Plan.Set(ctx, &next); diags.HasError() {
		t.Fatal(diags)
	}
	upd := &resource.UpdateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.Update(ctx, req, upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("update: %v", upd.Diagnostics)
	}
	var after interceptorModel
	upd.State.Get(ctx, &after)
	if after.Description.ValueString() != "changed" || core.body(t)["description"] != "changed" {
		t.Fatalf("description not updated: state=%v body=%v", after.Description, core.body(t))
	}
	if after.ID.ValueString() != "platform/f" {
		t.Fatalf("id changed: %v", after.ID)
	}

	core.mu.Lock()
	core.stored = nil
	core.mu.Unlock()
	rr, _ := runInterceptorRead(t, r, after)
	if rr.Diagnostics.HasError() || !rr.State.Raw.IsNull() {
		t.Fatalf("a missing interceptor must be removed from state, diags: %v", rr.Diagnostics)
	}
}

func TestInterceptorDeleteIsIdempotent(t *testing.T) {
	core := &fakeInterceptorCore{}
	r := &interceptorResource{client: core.client(t)}
	ctx := context.Background()

	_, created := runInterceptorCreate(t, r, newInterceptorPlan("http://e"))
	for i := 0; i < 2; i++ { // the second delete gets a 404
		state := newInterceptorState(t, created)
		resp := &resource.DeleteResponse{State: state}
		r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("delete #%d: %v", i+1, resp.Diagnostics)
		}
	}
}

func TestInterceptorImportState(t *testing.T) {
	ctx := context.Background()
	schema := interceptorResourceSchema(t).Schema
	r := &interceptorResource{}

	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "platform/filter"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import: %v", resp.Diagnostics)
	}
	var bucket, path, id types.String
	resp.State.GetAttribute(ctx, pathRoot("bucket"), &bucket)
	resp.State.GetAttribute(ctx, pathRoot("path"), &path)
	resp.State.GetAttribute(ctx, pathRoot("id"), &id)
	if bucket.ValueString() != "platform" || path.ValueString() != "filter" || id.ValueString() != "platform/filter" {
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
