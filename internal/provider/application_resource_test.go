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

// fakeApplicationCore imitates the parts of DIAL Core 0.50.1 the application
// resource depends on, as observed against a real Core: snake_case bodies, and a
// GET that fills in forward_auth_token false, max_retry_attempts 1 and an empty
// defaults object for whatever was not sent.
type fakeApplicationCore struct {
	mu       sync.Mutex
	stored   []byte
	received []byte
	paths    []string
}

func (f *fakeApplicationCore) client(t *testing.T) *client.Client {
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
			for key, value := range map[string]any{"forward_auth_token": false, "max_retry_attempts": 1, "defaults": map[string]any{}, "status": "valid"} {
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

func (f *fakeApplicationCore) body(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(f.received, &doc); err != nil {
		t.Fatalf("received body is not JSON: %v", err)
	}
	return doc
}

func applicationResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	var resp resource.SchemaResponse
	(&applicationResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp
}

// newApplicationPlan is a plan with the identity set, computed attributes
// unknown and everything else null, as Terraform produces for a minimal
// configuration.
func newApplicationPlan(endpoint string) applicationModel {
	return applicationModel{
		ID:      types.StringUnknown(),
		Bucket:  types.StringValue("platform"),
		Path:    types.StringValue("f"),
		Name:    types.StringUnknown(),
		APIName: types.StringUnknown(),
		applicationBody: applicationBody{
			Endpoint: types.StringValue(endpoint), Description: types.StringNull(), DisplayName: types.StringNull(),
			DisplayVersion: types.StringNull(), IconURL: types.StringNull(), CatalogSchemaID: types.StringNull(),
			ForwardAuthToken: types.BoolUnknown(), MaxRetryAttempts: types.Int64Unknown(),
			UserRoles:    types.SetNull(types.StringType),
			Interceptors: types.SetNull(types.StringType),
			EditorURL:    types.StringNull(), ViewerURL: types.StringNull(), ApplicationTypeSchemaID: types.StringNull(),
			ApplicationProperties: types.DynamicNull(),
			DefaultHeaders:        types.MapUnknown(types.StringType),
			Features:              types.ObjectNull(modelFeaturesAttrTypes),
			Interfaces:            types.MapUnknown(types.ObjectType{AttrTypes: modelInterfaceAttrTypes}),
			Defaults:              types.DynamicNull(), CatalogProperties: types.DynamicNull(),
		},
	}
}

func newApplicationState(t *testing.T, plan applicationModel) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schema := applicationResourceSchema(t).Schema
	state := tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	return state
}

func runApplicationCreate(t *testing.T, r *applicationResource, plan applicationModel) (*resource.CreateResponse, applicationModel) {
	t.Helper()
	ctx := context.Background()
	schema := applicationResourceSchema(t).Schema
	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	if diags := req.Plan.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("plan: %v", diags)
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, req, resp)
	var state applicationModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("state: %v", diags)
		}
	}
	return resp, state
}

func runApplicationRead(t *testing.T, r *applicationResource, prior applicationModel) (*resource.ReadResponse, applicationModel) {
	t.Helper()
	ctx := context.Background()
	state := newApplicationState(t, prior)
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	var out applicationModel
	if !resp.Diagnostics.HasError() && !resp.State.Raw.IsNull() {
		if diags := resp.State.Get(ctx, &out); diags.HasError() {
			t.Fatalf("state: %v", diags)
		}
	}
	return resp, out
}

func TestApplicationSchemasAreValidAndMatchModels(t *testing.T) {
	ctx := context.Background()
	rs := applicationResourceSchema(t).Schema
	if diags := rs.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("resource schema: %v", diags)
	}
	// The model struct must be convertible to and from the schema.
	state := newApplicationState(t, newApplicationPlan("http://e"))
	if state.Raw.IsNull() {
		t.Fatal("state is null")
	}

	var api, file datasource.SchemaResponse
	(&applicationDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &api)
	(&fileApplicationDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &file)
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
			t.Errorf("aidial_application data source lacks attribute %q of the resource", name)
		}
	}
	for _, name := range []string{"author", "created_at", "updated_at"} {
		if _, ok := rs.Attributes[name]; ok {
			t.Errorf("%s must not be exposed", name)
		}
	}
}

func TestApplicationTypesRegistered(t *testing.T) {
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
	if !resources["aidial_application"] {
		t.Error("aidial_application resource is not registered")
	}
	for _, name := range []string{"aidial_application", "aidial_file_application"} {
		if !sources[name] {
			t.Errorf("%s data source is not registered", name)
		}
	}
}

func TestApplicationEndpointIsOptionalAndExclusiveWithSchemaID(t *testing.T) {
	rs := applicationResourceSchema(t).Schema
	endpoint := identityFromResourceSchema(t, "endpoint", rs)
	if endpoint.required || !endpoint.optional || endpoint.computed {
		t.Fatalf("endpoint must be optional only: %+v", endpoint)
	}
	if len(endpoint.validators) != 1 {
		t.Fatalf("endpoint needs the exactly-one-of validator with application_type_schema_id: %+v", endpoint)
	}
}

func TestApplicationMinimalCreateSendsOnlyConfiguredFields(t *testing.T) {
	core := &fakeApplicationCore{}
	r := &applicationResource{client: core.client(t)}

	resp, state := runApplicationCreate(t, r, newApplicationPlan("http://hook/chat"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	body := core.body(t)
	if len(body) != 1 || body["endpoint"] != "http://hook/chat" {
		t.Fatalf("expected only endpoint in the request body (computed attributes are not sent), got %v", body)
	}
	if core.paths[0] != "PUT /v1/applications/platform/f" || core.paths[1] != "GET /v1/applications/platform/f" {
		t.Fatalf("create must PUT and re-read with GET, requests: %v", core.paths)
	}
	if state.ID.ValueString() != "platform/f" || state.APIName.ValueString() != "f" || state.Name.ValueString() != "f" {
		t.Errorf("id=%v api_name=%v name=%v", state.ID, state.APIName, state.Name)
	}
	if state.ForwardAuthToken.IsNull() || state.ForwardAuthToken.ValueBool() || state.MaxRetryAttempts.ValueInt64() != 1 {
		t.Errorf("Core defaults not read back: %v / %v", state.ForwardAuthToken, state.MaxRetryAttempts)
	}
	if !state.UserRoles.IsNull() || !state.Features.IsNull() || !state.CatalogProperties.IsNull() || !state.Description.IsNull() {
		t.Errorf("attributes Core omits must be null: %+v", state.applicationBody)
	}
	if state.DefaultHeaders.IsNull() || len(state.DefaultHeaders.Elements()) != 0 || state.Interfaces.IsNull() || len(state.Interfaces.Elements()) != 0 {
		t.Errorf("default_headers and interfaces must be empty maps, got %v / %v", state.DefaultHeaders, state.Interfaces)
	}
	// Core returns defaults {} but the config left it unset: it stays null.
	if !state.Defaults.IsNull() {
		t.Errorf("an empty defaults object must read back as null when unconfigured, got %v", state.Defaults)
	}
}

func TestApplicationFullCreateRoundTrips(t *testing.T) {
	core := &fakeApplicationCore{}
	r := &applicationResource{client: core.client(t)}

	plan := newApplicationPlan("http://hook/chat")
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

	resp, state := runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	body := core.body(t)
	for key, want := range map[string]any{
		"endpoint": "http://hook/chat", "description": "d", "display_name": "Hook", "display_version": "1.0",
		"icon_url": "http://i", "catalog_schema_id": "https://s/1", "forward_auth_token": true, "max_retry_attempts": float64(3),
	} {
		if body[key] != want {
			t.Errorf("body[%s] = %v, want %v", key, body[key], want)
		}
	}
	if roles, _ := body["user_roles"].([]any); len(roles) != 2 {
		t.Errorf("user_roles = %v", body["user_roles"])
	}
	if h, _ := body["default_headers"].(map[string]any); h["X-A"] != "1" {
		t.Errorf("default_headers = %v", body["default_headers"])
	}
	if f, _ := body["features"].(map[string]any); f["tools_supported"] != true {
		t.Errorf("features = %v", body["features"])
	}
	if i, _ := body["interfaces"].(map[string]any); i["chat"] == nil {
		t.Errorf("interfaces = %v", body["interfaces"])
	}
	if c, _ := body["catalog_properties"].(map[string]any); c["k"] != "v" {
		t.Errorf("catalog_properties = %v", body["catalog_properties"])
	}

	if !state.UserRoles.Equal(plan.UserRoles) || !state.DefaultHeaders.Equal(plan.DefaultHeaders) || !state.Features.Equal(plan.Features) {
		t.Errorf("collections did not round-trip: %+v", state.applicationBody)
	}
	if !state.Defaults.Equal(plan.Defaults) || !state.CatalogProperties.Equal(plan.CatalogProperties) {
		t.Errorf("free-form values must keep the configured value, got %v / %v", state.Defaults, state.CatalogProperties)
	}
	if state.ForwardAuthToken.ValueBool() != true || state.MaxRetryAttempts.ValueInt64() != 3 {
		t.Errorf("configured scalars changed: %v / %v", state.ForwardAuthToken, state.MaxRetryAttempts)
	}
}

func TestApplicationConfiguredEmptySetIsSentAndStaysEmpty(t *testing.T) {
	core := &fakeApplicationCore{}
	r := &applicationResource{client: core.client(t)}

	plan := newApplicationPlan("http://e")
	plan.UserRoles = types.SetValueMust(types.StringType, []attr.Value{})
	resp, state := runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	if roles, ok := core.body(t)["user_roles"].([]any); !ok || len(roles) != 0 {
		t.Fatalf("an empty set must be sent as [], got %v", core.body(t)["user_roles"])
	}
	if state.UserRoles.IsNull() || len(state.UserRoles.Elements()) != 0 {
		t.Fatalf("configured empty set must stay empty, got %v", state.UserRoles)
	}
}

func TestApplicationFreeFormKeyOrderIsNoOp(t *testing.T) {
	core := &fakeApplicationCore{}
	r := &applicationResource{client: core.client(t)}

	plan := newApplicationPlan("http://e")
	plan.CatalogProperties = types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"z": types.StringType, "a": types.StringType},
		map[string]attr.Value{"z": types.StringValue("1"), "a": types.StringValue("2")},
	))
	resp, created := runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	_, refreshed := runApplicationRead(t, r, created)
	if !refreshed.CatalogProperties.Equal(plan.CatalogProperties) {
		t.Fatalf("refresh changed free-form value: %v", refreshed.CatalogProperties)
	}
}

func TestApplicationUpdateIsInPlaceAndOutOfBandDeletionRemovesState(t *testing.T) {
	core := &fakeApplicationCore{}
	r := &applicationResource{client: core.client(t)}
	ctx := context.Background()

	resp, created := runApplicationCreate(t, r, newApplicationPlan("http://e"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	next := created
	next.Description = types.StringValue("changed")
	schema := applicationResourceSchema(t).Schema
	req := resource.UpdateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	if diags := req.Plan.Set(ctx, &next); diags.HasError() {
		t.Fatal(diags)
	}
	upd := &resource.UpdateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.Update(ctx, req, upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("update: %v", upd.Diagnostics)
	}
	var after applicationModel
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
	rr, _ := runApplicationRead(t, r, after)
	if rr.Diagnostics.HasError() || !rr.State.Raw.IsNull() {
		t.Fatalf("a missing application must be removed from state, diags: %v", rr.Diagnostics)
	}
}

func TestApplicationDeleteIsIdempotent(t *testing.T) {
	core := &fakeApplicationCore{}
	r := &applicationResource{client: core.client(t)}
	ctx := context.Background()

	_, created := runApplicationCreate(t, r, newApplicationPlan("http://e"))
	for i := 0; i < 2; i++ { // the second delete gets a 404
		state := newApplicationState(t, created)
		resp := &resource.DeleteResponse{State: state}
		r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("delete #%d: %v", i+1, resp.Diagnostics)
		}
	}
}

func TestApplicationImportState(t *testing.T) {
	ctx := context.Background()
	schema := applicationResourceSchema(t).Schema
	r := &applicationResource{}

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

func TestApplicationSchemaBasedCreateSendsNoEndpoint(t *testing.T) {
	core := &fakeApplicationCore{}
	r := &applicationResource{client: core.client(t)}

	plan := newApplicationPlan("")
	plan.Endpoint = types.StringNull()
	plan.ApplicationTypeSchemaID = types.StringValue("https://dial.epam.com/application_type_schemas/x")
	plan.ApplicationProperties = types.DynamicValue(types.ObjectValueMust(
		map[string]attr.Type{"prompt": types.StringType}, map[string]attr.Value{"prompt": types.StringValue("hi")},
	))
	plan.Interceptors = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("i1")})
	plan.EditorURL = types.StringValue("http://editor")
	plan.ViewerURL = types.StringValue("http://viewer")

	resp, state := runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	body := core.body(t)
	if _, ok := body["endpoint"]; ok {
		t.Errorf("endpoint must not be sent for a schema-based application: %v", body)
	}
	for key, want := range map[string]any{
		"application_type_schema_id": "https://dial.epam.com/application_type_schemas/x",
		"editor_url":                 "http://editor", "viewer_url": "http://viewer",
	} {
		if body[key] != want {
			t.Errorf("body[%s] = %v, want %v", key, body[key], want)
		}
	}
	if p, _ := body["application_properties"].(map[string]any); p["prompt"] != "hi" {
		t.Errorf("application_properties = %v", body["application_properties"])
	}
	if i, _ := body["interceptors"].([]any); len(i) != 1 || i[0] != "i1" {
		t.Errorf("interceptors = %v", body["interceptors"])
	}
	if !state.Endpoint.IsNull() {
		t.Errorf("endpoint must read back as null, got %v", state.Endpoint)
	}
	if !state.ApplicationProperties.Equal(plan.ApplicationProperties) || !state.Interceptors.Equal(plan.Interceptors) {
		t.Errorf("application_properties/interceptors did not round-trip: %+v", state.applicationBody)
	}
}

func TestApplicationPublicBucketNestedPath(t *testing.T) {
	core := &fakeApplicationCore{}
	r := &applicationResource{client: core.client(t)}

	plan := newApplicationPlan("http://e")
	plan.Bucket, plan.Path = types.StringValue("public"), types.StringValue("team/app")
	resp, state := runApplicationCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	if core.paths[0] != "PUT /v1/applications/public/team/app" {
		t.Fatalf("requests: %v", core.paths)
	}
	if state.ID.ValueString() != "public/team/app" {
		t.Errorf("id = %v", state.ID)
	}
}

func TestApplicationIdentityFollowsToolsetRule(t *testing.T) {
	bucket, path, replaces := resourceIdentity(t, &applicationResource{})
	for name, a := range map[string]identityAttr{"bucket": bucket, "path": path} {
		if !a.required || a.optional || a.computed {
			t.Errorf("%s must be required only: %+v", name, a)
		}
	}
	if !replaces {
		t.Error("bucket and path must force replacement")
	}
	for _, ok := range []string{"platform", "public"} {
		if !validate(t, bucket.validators, ok) {
			t.Errorf("bucket %q must be accepted", ok)
		}
	}
	if validate(t, bucket.validators, "") || validate(t, path.validators, "") {
		t.Error("empty bucket and empty path must be rejected")
	}
	if !validate(t, path.validators, "team/app") {
		t.Error("nested path must be accepted")
	}

	dsBucket, dsPath := dataSourceIdentity(t, &applicationDataSource{})
	if !validate(t, dsBucket.validators, "public") || validate(t, dsBucket.validators, "") || validate(t, dsPath.validators, "") {
		t.Error("data source identity must follow the same rule")
	}
}

func TestApplicationImportKeepsNestedPath(t *testing.T) {
	ctx := context.Background()
	schema := applicationResourceSchema(t).Schema
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	(&applicationResource{}).ImportState(ctx, resource.ImportStateRequest{ID: "public/team/app"}, resp)
	var bucket, path types.String
	resp.State.GetAttribute(ctx, pathRoot("bucket"), &bucket)
	resp.State.GetAttribute(ctx, pathRoot("path"), &path)
	if bucket.ValueString() != "public" || path.ValueString() != "team/app" {
		t.Fatalf("bucket=%v path=%v", bucket, path)
	}
}
