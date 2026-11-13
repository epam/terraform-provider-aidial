package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

const toolsetAPIBody = `{
	"name":"toolsets/public/team/tool",
	"endpoint":"http://tool/mcp",
	"transport":"HTTP",
	"provider":"Acme",
	"allowed_tools":["a"],
	"default_headers":{"X-A":"1"},
	"catalog_properties":{"vendor":"Acme"},
	"features":{"tools_supported":true},
	"interfaces":{"openaiResponses":{"base_url":"http://i"}},
	"auth_settings":{"authentication_type":"OAUTH","client_id":"cid","client_secret":"leak","user_level_auth_status":"SIGNED_IN"}
}`

// readToolsetSource runs Read on a toolset data source for a config holding
// the given string arguments (all other attributes null).
func readToolsetSource(t *testing.T, ds datasource.DataSource, args map[string]string) *datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	values := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	for name, value := range args {
		values[name] = tftypes.NewValue(tftypes.String, value)
	}
	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, values)}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(objType, nil)}}
	ds.Read(ctx, req, resp)
	return resp
}

func TestToolsetDataSourcePopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(toolsetAPIBody))
	})

	resp := readToolsetSource(t, &toolsetDataSource{client: c}, map[string]string{"bucket": "public", "path": "team/tool"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/toolsets/public/team/tool" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	for name, want := range map[string]string{
		"id":            "public/team/tool",
		"bucket":        "public",
		"path":          "team/tool",
		"name":          "toolsets/public/team/tool",
		"api_name":      "toolsets/public/team/tool",
		"endpoint":      "http://tool/mcp",
		"transport":     "HTTP",
		"provider_name": "Acme",
	} {
		var got types.String
		resp.State.GetAttribute(context.Background(), pathRoot(name), &got)
		if got.ValueString() != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}

	// The auth secret Core echoed must not reach state.
	var auth types.Object
	resp.State.GetAttribute(context.Background(), pathRoot("auth_settings"), &auth)
	if _, leaked := auth.Attributes()["client_secret"]; leaked {
		t.Fatalf("auth_settings must not carry client_secret: %v", auth)
	}
	if auth.Attributes()["client_id"] != types.StringValue("cid") {
		t.Fatalf("auth_settings = %v", auth)
	}
}

// The data source must hold the same values the resource would for the same
// API response (apart from client_secret, which only the resource exposes).
func TestToolsetDataSourceMatchesResource(t *testing.T) {
	var ts client.ToolSet
	if err := json.Unmarshal([]byte(toolsetAPIBody), &ts); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	resourceState := newToolsetPlan("public", "team/tool")
	resourceState.Name = types.StringNull()
	if err := setToolsetState(ctx, &resourceState, &ts, true); err != nil {
		t.Fatal(err)
	}
	dataState := newToolsetPlan("public", "team/tool")
	dataState.Name = types.StringNull()
	if err := setToolsetState(ctx, &dataState, &ts, false); err != nil {
		t.Fatal(err)
	}

	resourceState.AuthSettings, dataState.AuthSettings = types.ObjectNull(nil), types.ObjectNull(nil)
	if !resourceState.Features.Equal(dataState.Features) ||
		!resourceState.Interfaces.Equal(dataState.Interfaces) ||
		!resourceState.AllowedTools.Equal(dataState.AllowedTools) ||
		!resourceState.DefaultHeaders.Equal(dataState.DefaultHeaders) ||
		!resourceState.CatalogProperties.Equal(dataState.CatalogProperties) ||
		!resourceState.Endpoint.Equal(dataState.Endpoint) {
		t.Fatalf("data source and resource disagree:\n resource: %+v\n data:     %+v", resourceState, dataState)
	}
}

func TestToolsetDataSourceOmittedCollectionsFollowResource(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"t"}`))
	})
	resp := readToolsetSource(t, &toolsetDataSource{client: c}, map[string]string{"bucket": "platform", "path": "t"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	var tools types.Set
	var features, auth types.Object
	var interfaces, headers types.Map
	resp.State.GetAttribute(context.Background(), pathRoot("allowed_tools"), &tools)
	resp.State.GetAttribute(context.Background(), pathRoot("features"), &features)
	resp.State.GetAttribute(context.Background(), pathRoot("auth_settings"), &auth)
	resp.State.GetAttribute(context.Background(), pathRoot("interfaces"), &interfaces)
	resp.State.GetAttribute(context.Background(), pathRoot("default_headers"), &headers)
	if !tools.IsNull() || !features.IsNull() || !auth.IsNull() {
		t.Errorf("allowed_tools, features and auth_settings must be null")
	}
	if interfaces.IsNull() || len(interfaces.Elements()) != 0 || headers.IsNull() || len(headers.Elements()) != 0 {
		t.Errorf("interfaces and default_headers must be empty maps, got %v / %v", interfaces, headers)
	}
}

func TestToolsetDataSourceMissingAndAPIError(t *testing.T) {
	missing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	resp := readToolsetSource(t, &toolsetDataSource{client: missing}, map[string]string{"bucket": "public", "path": "gone"})
	if !resp.Diagnostics.HasError() {
		t.Fatal("a missing toolset must be an error")
	}
	if d := resp.Diagnostics.Errors()[0]; !strings.Contains(d.Detail(), `"public"`) || !strings.Contains(d.Detail(), `"gone"`) {
		t.Fatalf("diagnostic must name bucket and path, got %q", d.Detail())
	}

	broken := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	resp = readToolsetSource(t, &toolsetDataSource{client: broken}, map[string]string{"bucket": "platform", "path": "t"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "500") || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "boom") {
		t.Fatalf("API error must carry status and body, got %v", resp.Diagnostics)
	}
}

func TestToolsetDataSourceIdentityValidation(t *testing.T) {
	var resp datasource.SchemaResponse
	(&toolsetDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	bucket, path := dataSourceIdentityFromSchema(t, "bucket", resp), dataSourceIdentityFromSchema(t, "path", resp)
	if !bucket.required || !path.required {
		t.Fatal("bucket and path must be required")
	}
	if !validate(t, bucket.validators, "public") || !validate(t, bucket.validators, "platform") {
		t.Error("platform and public buckets must be accepted")
	}
	if validate(t, bucket.validators, "") || validate(t, path.validators, "") {
		t.Error("empty bucket or path must be rejected")
	}
}

func TestFileToolsetDataSourcePopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(toolsetAPIBody))
	})

	resp := readToolsetSource(t, &fileToolsetDataSource{client: c}, map[string]string{"name": "a/b"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/admin/config/file/toolsets/a%2Fb" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	for name, want := range map[string]string{"name": "a/b", "endpoint": "http://tool/mcp", "transport": "HTTP", "provider_name": "Acme"} {
		var got types.String
		resp.State.GetAttribute(context.Background(), pathRoot(name), &got)
		if got.ValueString() != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}
}

func TestFileToolsetDataSourceShape(t *testing.T) {
	var resp datasource.SchemaResponse
	(&fileToolsetDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &resp)
	for _, name := range []string{"id", "bucket", "path", "api_name"} {
		if _, has := resp.Schema.Attributes[name]; has {
			t.Errorf("aidial_file_toolset must not have %q", name)
		}
	}
	var api datasource.SchemaResponse
	(&toolsetDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &api)
	for name := range resp.Schema.Attributes {
		if _, ok := api.Schema.Attributes[name]; !ok {
			t.Errorf("aidial_toolset lacks attribute %q present in aidial_file_toolset", name)
		}
	}

	name := dataSourceIdentityFromSchema(t, "name", resp)
	if !name.required {
		t.Error("name must be required")
	}
	if validate(t, name.validators, "") || !validate(t, name.validators, "a") {
		t.Error("an empty name must be rejected and a non-empty one accepted")
	}
}

func TestFileToolsetDataSourceMissingAndEmptyName(t *testing.T) {
	missing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	resp := readToolsetSource(t, &fileToolsetDataSource{client: missing}, map[string]string{"name": "nope"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"nope"`) {
		t.Fatalf("a missing toolset must be an error naming it, got %v", resp.Diagnostics)
	}
}

func dataSourceIdentityFromSchema(t *testing.T, name string, resp datasource.SchemaResponse) identityAttr {
	t.Helper()
	a, ok := resp.Schema.Attributes[name].(dsschema.StringAttribute)
	if !ok {
		t.Fatalf("%s is not a StringAttribute", name)
	}
	return identityAttr{a.Required, a.Optional, a.Computed, a.Validators}
}
