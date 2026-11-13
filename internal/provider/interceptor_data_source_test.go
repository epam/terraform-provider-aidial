package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const interceptorAPIBody = `{
	"name":"interceptors/platform/filter",
	"endpoint":"http://hook/chat",
	"displayName":"Hook",
	"forwardAuthToken":true,
	"maxRetryAttempts":2,
	"userRoles":["a"],
	"defaultHeaders":{"X-A":"1"},
	"catalogProperties":{"vendor":"Acme"},
	"features":{"tools_supported":true},
	"interfaces":{"chat":{"base_url":"http://i"}},
	"author":"someone","createdAt":1,"updatedAt":2,"status":"valid"
}`

func TestInterceptorDataSourcePopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(interceptorAPIBody))
	})

	resp := readToolsetSource(t, &interceptorDataSource{client: c}, map[string]string{"bucket": "platform", "path": "filter"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/interceptors/platform/filter" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	for name, want := range map[string]string{
		"id": "platform/filter", "api_name": "interceptors/platform/filter", "endpoint": "http://hook/chat", "display_name": "Hook",
	} {
		var got types.String
		resp.State.GetAttribute(context.Background(), pathRoot(name), &got)
		if got.ValueString() != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}
	var features types.Object
	resp.State.GetAttribute(context.Background(), pathRoot("features"), &features)
	if features.IsNull() {
		t.Error("features must be populated")
	}
}

func TestInterceptorDataSourceMatchesResource(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(interceptorAPIBody)) })

	resp := readToolsetSource(t, &interceptorDataSource{client: c}, map[string]string{"bucket": "platform", "path": "filter"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("data source: %v", resp.Diagnostics)
	}
	var fromSource interceptorModel
	resp.State.Get(context.Background(), &fromSource)

	r := &interceptorResource{client: c}
	_, fromResource := runInterceptorRead(t, r, interceptorModel{
		ID: types.StringValue("platform/filter"), Bucket: types.StringValue("platform"), Path: types.StringValue("filter"),
		Name: types.StringNull(), APIName: types.StringNull(), interceptorBody: newInterceptorPlan("").interceptorBody,
	})
	fromResource.Name = fromSource.Name
	if !fromSource.UserRoles.Equal(fromResource.UserRoles) || !fromSource.DefaultHeaders.Equal(fromResource.DefaultHeaders) ||
		!fromSource.Features.Equal(fromResource.Features) || !fromSource.Interfaces.Equal(fromResource.Interfaces) ||
		!fromSource.Defaults.Equal(fromResource.Defaults) || !fromSource.CatalogProperties.Equal(fromResource.CatalogProperties) ||
		fromSource.Endpoint != fromResource.Endpoint || fromSource.MaxRetryAttempts != fromResource.MaxRetryAttempts {
		t.Fatalf("data source and resource disagree:\n source   %+v\n resource %+v", fromSource.interceptorBody, fromResource.interceptorBody)
	}
}

func TestInterceptorDataSourceMissingAndAPIError(t *testing.T) {
	missing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	resp := readToolsetSource(t, &interceptorDataSource{client: missing}, map[string]string{"bucket": "platform", "path": "nope"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"platform"`) || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"nope"`) {
		t.Fatalf("a missing interceptor must be an error naming bucket and path, got %v", resp.Diagnostics)
	}

	failing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	resp = readToolsetSource(t, &interceptorDataSource{client: failing}, map[string]string{"bucket": "platform", "path": "x"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "500") || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "boom") {
		t.Fatalf("an API error must carry status and body, got %v", resp.Diagnostics)
	}
}

func TestFileInterceptorDataSourcePopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(interceptorAPIBody))
	})

	resp := readToolsetSource(t, &fileInterceptorDataSource{client: c}, map[string]string{"name": "a/b"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/admin/config/file/interceptors/a%2Fb" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	for name, want := range map[string]string{"name": "a/b", "endpoint": "http://hook/chat", "display_name": "Hook"} {
		var got types.String
		resp.State.GetAttribute(context.Background(), pathRoot(name), &got)
		if got.ValueString() != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}
}

func TestFileInterceptorDataSourceShape(t *testing.T) {
	var file, api datasource.SchemaResponse
	(&fileInterceptorDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &file)
	(&interceptorDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &api)
	for _, name := range []string{"id", "bucket", "path", "api_name"} {
		if _, has := file.Schema.Attributes[name]; has {
			t.Errorf("aidial_file_interceptor must not have %q", name)
		}
	}
	for name := range file.Schema.Attributes {
		if _, ok := api.Schema.Attributes[name]; !ok {
			t.Errorf("aidial_interceptor lacks attribute %q present in aidial_file_interceptor", name)
		}
	}
	name := dataSourceIdentityFromSchema(t, "name", file)
	if !name.required || validate(t, name.validators, "") || !validate(t, name.validators, "a") {
		t.Error("name must be required, non-empty")
	}
}

func TestFileInterceptorDataSourceMissing(t *testing.T) {
	missing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	resp := readToolsetSource(t, &fileInterceptorDataSource{client: missing}, map[string]string{"name": "nope"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"nope"`) {
		t.Fatalf("a missing interceptor must be an error naming it, got %v", resp.Diagnostics)
	}
}
