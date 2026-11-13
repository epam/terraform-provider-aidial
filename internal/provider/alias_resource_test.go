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

// fakeAliasCore imitates the parts of DIAL Core 0.50.1 the alias resource
// depends on, as observed against a real Core: camelCase bodies, a GET that
// adds the name and status "valid", and omitted fields staying omitted.
type fakeAliasCore struct {
	mu       sync.Mutex
	stored   []byte
	received []byte
	paths    []string
}

func (f *fakeAliasCore) client(t *testing.T) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.paths = append(f.paths, r.Method+" "+r.URL.EscapedPath())
		switch r.Method {
		case http.MethodPut:
			f.stored, _ = io.ReadAll(r.Body)
			f.received = append([]byte(nil), f.stored...)
			_, _ = w.Write([]byte(`{"name":"a"}`))
		case http.MethodGet:
			if f.stored == nil {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var doc map[string]any
			_ = json.Unmarshal(f.stored, &doc)
			doc["name"] = "a"
			doc["status"] = "valid"
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

func (f *fakeAliasCore) body(t *testing.T) map[string]any {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(f.received, &doc); err != nil {
		t.Fatalf("received body is not JSON: %v", err)
	}
	return doc
}

func aliasResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	var resp resource.SchemaResponse
	(&aliasResource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	return resp
}

// newAliasPlan is a plan with the identity set, computed attributes unknown
// and everything else null, as Terraform produces for a minimal configuration.
func newAliasPlan(presentAs, targetPath string) aliasModel {
	return aliasModel{
		ID:      types.StringUnknown(),
		Bucket:  types.StringValue("public"),
		Path:    types.StringValue("a"),
		Name:    types.StringUnknown(),
		APIName: types.StringUnknown(),
		aliasBody: aliasBody{
			PresentAs: types.StringValue(presentAs), TargetPath: types.StringValue(targetPath),
			DisplayName: types.StringNull(), Description: types.StringNull(), DisplayVersion: types.StringNull(),
			IconURL: types.StringNull(), Intro: types.StringNull(),
			UserRoles: types.SetNull(types.StringType), DescriptionKeys: types.SetNull(types.StringType),
		},
	}
}

func newAliasState(t *testing.T, plan aliasModel) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schema := aliasResourceSchema(t).Schema
	state := tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}
	if diags := state.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("state: %v", diags)
	}
	return state
}

func runAliasCreate(t *testing.T, r *aliasResource, plan aliasModel) (*resource.CreateResponse, aliasModel) {
	t.Helper()
	ctx := context.Background()
	schema := aliasResourceSchema(t).Schema
	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	if diags := req.Plan.Set(ctx, &plan); diags.HasError() {
		t.Fatalf("plan: %v", diags)
	}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.Create(ctx, req, resp)
	var state aliasModel
	if !resp.Diagnostics.HasError() {
		if diags := resp.State.Get(ctx, &state); diags.HasError() {
			t.Fatalf("state: %v", diags)
		}
	}
	return resp, state
}

func runAliasRead(t *testing.T, r *aliasResource, prior aliasModel) (*resource.ReadResponse, aliasModel) {
	t.Helper()
	ctx := context.Background()
	state := newAliasState(t, prior)
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	var out aliasModel
	if !resp.Diagnostics.HasError() && !resp.State.Raw.IsNull() {
		if diags := resp.State.Get(ctx, &out); diags.HasError() {
			t.Fatalf("state: %v", diags)
		}
	}
	return resp, out
}

func TestAliasSchemasAreValidAndMatch(t *testing.T) {
	ctx := context.Background()
	rs := aliasResourceSchema(t).Schema
	if diags := rs.ValidateImplementation(ctx); diags.HasError() {
		t.Fatalf("resource schema: %v", diags)
	}
	if newAliasState(t, newAliasPlan("model", "applications/x")).Raw.IsNull() {
		t.Fatal("state is null")
	}

	var api, file datasource.SchemaResponse
	(&aliasDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &api)
	(&fileAliasDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &file)
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
			t.Errorf("aidial_alias data source lacks attribute %q of the resource", name)
		}
	}
	for _, name := range []string{"author", "created_at", "updated_at", "status", "target", "endpoint", "features"} {
		if _, ok := rs.Attributes[name]; ok {
			t.Errorf("%s must not be exposed", name)
		}
	}
}

func TestAliasTypesRegistered(t *testing.T) {
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
	if !resources["aidial_alias"] {
		t.Error("aidial_alias resource is not registered")
	}
	for _, name := range []string{"aidial_alias", "aidial_file_alias"} {
		if !sources[name] {
			t.Errorf("%s data source is not registered", name)
		}
	}
}

func TestAliasRequiredAndEnumAttributes(t *testing.T) {
	rs := aliasResourceSchema(t).Schema
	for _, name := range []string{"present_as", "target_path"} {
		a := identityFromResourceSchema(t, name, rs)
		if !a.required || a.optional || a.computed {
			t.Errorf("%s must be required only: %+v", name, a)
		}
	}

	presentAs := identityFromResourceSchema(t, "present_as", rs)
	for _, ok := range []string{"model", "application", "toolset"} {
		if !validate(t, presentAs.validators, ok) {
			t.Errorf("present_as %q must be accepted", ok)
		}
	}
	for _, bad := range []string{"", "bogus", "Model"} {
		if validate(t, presentAs.validators, bad) {
			t.Errorf("present_as %q must be rejected", bad)
		}
	}

	target := identityFromResourceSchema(t, "target_path", rs)
	for _, ok := range []string{"models/gpt", "applications/chat", "toolsets/tools"} {
		if !validate(t, target.validators, ok) {
			t.Errorf("target_path %q must be accepted", ok)
		}
	}
	for _, bad := range []string{"", "gpt", "models/", "models/platform/gpt", "http://core/v1/applications/platform/x", "prompts/x"} {
		if validate(t, target.validators, bad) {
			t.Errorf("target_path %q must be rejected", bad)
		}
	}
}

func TestAliasValidateConfigPresentAsAndTargetKind(t *testing.T) {
	ctx := context.Background()
	schema := aliasResourceSchema(t).Schema
	r := &aliasResource{}

	check := func(presentAs, targetPath types.String) bool {
		values := map[string]tftypes.Value{}
		objType := schema.Type().TerraformType(ctx).(tftypes.Object)
		for name, typ := range objType.AttributeTypes {
			values[name] = tftypes.NewValue(typ, nil)
		}
		toValue := func(s types.String) tftypes.Value {
			switch {
			case s.IsUnknown():
				return tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
			case s.IsNull():
				return tftypes.NewValue(tftypes.String, nil)
			}
			return tftypes.NewValue(tftypes.String, s.ValueString())
		}
		values["present_as"], values["target_path"] = toValue(presentAs), toValue(targetPath)
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schema, Raw: tftypes.NewValue(objType, values)}}, resp)
		return !resp.Diagnostics.HasError()
	}
	s := types.StringValue
	for _, tc := range []struct {
		presentAs, target string
		ok                bool
	}{
		{"model", "applications/x", true},
		{"model", "models/x", true},
		{"application", "applications/x", true},
		{"application", "models/x", true},
		{"toolset", "toolsets/x", true},
		{"model", "toolsets/x", false},
		{"application", "toolsets/x", false},
		{"toolset", "applications/x", false},
		{"toolset", "models/x", false},
	} {
		if got := check(s(tc.presentAs), s(tc.target)); got != tc.ok {
			t.Errorf("present_as=%s target_path=%s: valid=%v, want %v", tc.presentAs, tc.target, got, tc.ok)
		}
	}
	if !check(types.StringUnknown(), s("toolsets/x")) || !check(s("model"), types.StringUnknown()) || !check(types.StringNull(), s("x")) {
		t.Error("unknown or null values must not fail the cross-attribute check")
	}
}

func TestAliasMinimalCreateSendsOnlyConfiguredFields(t *testing.T) {
	core := &fakeAliasCore{}
	r := &aliasResource{client: core.client(t)}

	resp, state := runAliasCreate(t, r, newAliasPlan("model", "applications/chat"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	body := core.body(t)
	target, _ := body["target"].(map[string]any)
	if len(body) != 2 || body["presentAs"] != "model" || len(target) != 1 || target["path"] != "applications/chat" {
		t.Fatalf("expected only presentAs and target.path in the request body, got %v", body)
	}
	if core.paths[0] != "PUT /v1/aliases/public/a" || core.paths[1] != "GET /v1/aliases/public/a" {
		t.Fatalf("create must PUT and re-read with GET, requests: %v", core.paths)
	}
	if state.ID.ValueString() != "public/a" || state.APIName.ValueString() != "a" || state.Name.ValueString() != "a" {
		t.Errorf("id=%v api_name=%v name=%v", state.ID, state.APIName, state.Name)
	}
	if state.PresentAs.ValueString() != "model" || state.TargetPath.ValueString() != "applications/chat" {
		t.Errorf("present_as=%v target_path=%v", state.PresentAs, state.TargetPath)
	}
	if !state.UserRoles.IsNull() || !state.DescriptionKeys.IsNull() || !state.Description.IsNull() || !state.Intro.IsNull() ||
		!state.DisplayName.IsNull() || !state.DisplayVersion.IsNull() || !state.IconURL.IsNull() {
		t.Errorf("attributes Core omits must be null: %+v", state.aliasBody)
	}
}

func TestAliasFullCreateRoundTrips(t *testing.T) {
	core := &fakeAliasCore{}
	r := &aliasResource{client: core.client(t)}

	plan := newAliasPlan("toolset", "toolsets/tools")
	plan.Bucket = types.StringValue("platform")
	plan.DisplayName, plan.Description = types.StringValue("Fast"), types.StringValue("d")
	plan.DisplayVersion, plan.IconURL, plan.Intro = types.StringValue("1.0"), types.StringValue("http://i"), types.StringValue("hello")
	plan.UserRoles = stringSet(t, "admin", "dev")
	plan.DescriptionKeys = stringSet(t, "fast", "cheap")

	resp, state := runAliasCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	if core.paths[0] != "PUT /v1/aliases/platform/a" {
		t.Fatalf("unexpected request %v", core.paths)
	}
	body := core.body(t)
	for key, want := range map[string]any{
		"presentAs": "toolset", "displayName": "Fast", "description": "d", "displayVersion": "1.0",
		"iconUrl": "http://i", "intro": "hello",
	} {
		if body[key] != want {
			t.Errorf("body[%s] = %v, want %v", key, body[key], want)
		}
	}
	if roles, _ := body["userRoles"].([]any); len(roles) != 2 {
		t.Errorf("userRoles = %v", body["userRoles"])
	}
	if kw, _ := body["descriptionKeywords"].([]any); len(kw) != 2 {
		t.Errorf("descriptionKeywords = %v", body["descriptionKeywords"])
	}
	if !state.UserRoles.Equal(plan.UserRoles) || !state.DescriptionKeys.Equal(plan.DescriptionKeys) {
		t.Errorf("collections did not round-trip: %+v", state.aliasBody)
	}
	if state.DisplayName != plan.DisplayName || state.Intro != plan.Intro || state.IconURL != plan.IconURL {
		t.Errorf("scalars did not round-trip: %+v", state.aliasBody)
	}
}

func TestAliasConfiguredEmptySetsAreSentAndStayEmpty(t *testing.T) {
	core := &fakeAliasCore{}
	r := &aliasResource{client: core.client(t)}

	plan := newAliasPlan("model", "models/m")
	plan.UserRoles = types.SetValueMust(types.StringType, []attr.Value{})
	plan.DescriptionKeys = types.SetValueMust(types.StringType, []attr.Value{})
	resp, state := runAliasCreate(t, r, plan)
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}
	for _, key := range []string{"userRoles", "descriptionKeywords"} {
		if v, ok := core.body(t)[key].([]any); !ok || len(v) != 0 {
			t.Fatalf("an empty set must be sent as [] for %s, got %v", key, core.body(t)[key])
		}
	}
	if state.UserRoles.IsNull() || len(state.UserRoles.Elements()) != 0 || state.DescriptionKeys.IsNull() || len(state.DescriptionKeys.Elements()) != 0 {
		t.Fatalf("configured empty sets must stay empty, got %v / %v", state.UserRoles, state.DescriptionKeys)
	}
}

func TestAliasUpdateIsInPlaceAndOutOfBandDeletionRemovesState(t *testing.T) {
	core := &fakeAliasCore{}
	r := &aliasResource{client: core.client(t)}
	ctx := context.Background()

	resp, created := runAliasCreate(t, r, newAliasPlan("model", "applications/chat"))
	if resp.Diagnostics.HasError() {
		t.Fatalf("create: %v", resp.Diagnostics)
	}

	next := created
	next.TargetPath = types.StringValue("applications/other")
	next.DisplayName = types.StringValue("changed")
	schema := aliasResourceSchema(t).Schema
	req := resource.UpdateRequest{Plan: tfsdk.Plan{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	if diags := req.Plan.Set(ctx, &next); diags.HasError() {
		t.Fatal(diags)
	}
	upd := &resource.UpdateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.Update(ctx, req, upd)
	if upd.Diagnostics.HasError() {
		t.Fatalf("update: %v", upd.Diagnostics)
	}
	var after aliasModel
	upd.State.Get(ctx, &after)
	if after.TargetPath.ValueString() != "applications/other" || after.DisplayName.ValueString() != "changed" {
		t.Fatalf("not updated: %+v", after.aliasBody)
	}
	if after.ID.ValueString() != "public/a" {
		t.Fatalf("id changed: %v", after.ID)
	}
	if core.paths[len(core.paths)-2] != "PUT /v1/aliases/public/a" {
		t.Fatalf("update must PUT in place, requests: %v", core.paths)
	}

	core.mu.Lock()
	core.stored = nil
	core.mu.Unlock()
	rr, _ := runAliasRead(t, r, after)
	if rr.Diagnostics.HasError() || !rr.State.Raw.IsNull() {
		t.Fatalf("a missing alias must be removed from state, diags: %v", rr.Diagnostics)
	}
}

func TestAliasCoreRejectionSurfacesAtApply(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"validationWarnings":[{"field":"target.path","message":"target.path is required"}]}`))
	}))
	t.Cleanup(srv.Close)
	c, _ := client.NewClient(srv.URL, client.APIKeyCredentials("k"))

	resp, _ := runAliasCreate(t, &aliasResource{client: c}, newAliasPlan("model", "applications/x"))
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "400") ||
		!strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "validationWarnings") {
		t.Fatalf("Core's 400 and body must be shown, got %v", resp.Diagnostics)
	}
}

func TestAliasDeleteIsIdempotent(t *testing.T) {
	core := &fakeAliasCore{}
	r := &aliasResource{client: core.client(t)}
	ctx := context.Background()

	_, created := runAliasCreate(t, r, newAliasPlan("model", "applications/chat"))
	for i := 0; i < 2; i++ { // the second delete gets a 404
		state := newAliasState(t, created)
		resp := &resource.DeleteResponse{State: state}
		r.Delete(ctx, resource.DeleteRequest{State: state}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("delete #%d: %v", i+1, resp.Diagnostics)
		}
	}
}

func TestAliasImportState(t *testing.T) {
	ctx := context.Background()
	schema := aliasResourceSchema(t).Schema
	r := &aliasResource{}

	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schema, Raw: tftypes.NewValue(schema.Type().TerraformType(ctx), nil)}}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "public/fast"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("import: %v", resp.Diagnostics)
	}
	var bucket, path, id types.String
	resp.State.GetAttribute(ctx, pathRoot("bucket"), &bucket)
	resp.State.GetAttribute(ctx, pathRoot("path"), &path)
	resp.State.GetAttribute(ctx, pathRoot("id"), &id)
	if bucket.ValueString() != "public" || path.ValueString() != "fast" || id.ValueString() != "public/fast" {
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
