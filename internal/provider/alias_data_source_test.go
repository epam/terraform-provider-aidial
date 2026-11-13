package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const aliasAPIBody = `{
	"name":"fast",
	"presentAs":"model",
	"target":{"path":"applications/chat"},
	"userRoles":["a"],
	"displayName":"Fast",
	"description":"d",
	"intro":"hello",
	"descriptionKeywords":["k"],
	"status":"valid"
}`

func TestAliasDataSourcePopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(aliasAPIBody))
	})

	resp := readToolsetSource(t, &aliasDataSource{client: c}, map[string]string{"bucket": "public", "path": "fast"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/aliases/public/fast" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	for name, want := range map[string]string{
		"id": "public/fast", "api_name": "fast", "present_as": "model", "target_path": "applications/chat", "display_name": "Fast", "intro": "hello",
	} {
		var got types.String
		resp.State.GetAttribute(context.Background(), pathRoot(name), &got)
		if got.ValueString() != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}
	var keywords types.Set
	resp.State.GetAttribute(context.Background(), pathRoot("description_keywords"), &keywords)
	if keywords.IsNull() || len(keywords.Elements()) != 1 {
		t.Errorf("description_keywords must be populated, got %v", keywords)
	}
}

func TestAliasDataSourceMatchesResource(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(aliasAPIBody)) })

	resp := readToolsetSource(t, &aliasDataSource{client: c}, map[string]string{"bucket": "public", "path": "fast"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("data source: %v", resp.Diagnostics)
	}
	var fromSource aliasModel
	resp.State.Get(context.Background(), &fromSource)

	r := &aliasResource{client: c}
	_, fromResource := runAliasRead(t, r, aliasModel{
		ID: types.StringValue("public/fast"), Bucket: types.StringValue("public"), Path: types.StringValue("fast"),
		Name: types.StringNull(), APIName: types.StringNull(), aliasBody: newAliasPlan("model", "x").aliasBody,
	})
	fromResource.Name = fromSource.Name
	if fromSource.PresentAs != fromResource.PresentAs || fromSource.TargetPath != fromResource.TargetPath ||
		fromSource.DisplayName != fromResource.DisplayName || fromSource.Intro != fromResource.Intro ||
		!fromSource.UserRoles.Equal(fromResource.UserRoles) || !fromSource.DescriptionKeys.Equal(fromResource.DescriptionKeys) {
		t.Fatalf("data source and resource disagree:\n source   %+v\n resource %+v", fromSource.aliasBody, fromResource.aliasBody)
	}
}

func TestAliasDataSourceMissingAndAPIError(t *testing.T) {
	missing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	resp := readToolsetSource(t, &aliasDataSource{client: missing}, map[string]string{"bucket": "public", "path": "nope"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"public"`) || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"nope"`) {
		t.Fatalf("a missing alias must be an error naming bucket and path, got %v", resp.Diagnostics)
	}

	failing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	resp = readToolsetSource(t, &aliasDataSource{client: failing}, map[string]string{"bucket": "public", "path": "x"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "500") || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "boom") {
		t.Fatalf("an API error must carry status and body, got %v", resp.Diagnostics)
	}
}

func TestFileAliasDataSourcePopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(aliasAPIBody))
	})

	resp := readToolsetSource(t, &fileAliasDataSource{client: c}, map[string]string{"name": "a/b"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/admin/config/file/aliases/a%2Fb" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	for name, want := range map[string]string{"name": "a/b", "present_as": "model", "target_path": "applications/chat", "display_name": "Fast"} {
		var got types.String
		resp.State.GetAttribute(context.Background(), pathRoot(name), &got)
		if got.ValueString() != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}
}

func TestFileAliasDataSourceShape(t *testing.T) {
	var file, api datasource.SchemaResponse
	(&fileAliasDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &file)
	(&aliasDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &api)
	for _, name := range []string{"id", "bucket", "path", "api_name"} {
		if _, has := file.Schema.Attributes[name]; has {
			t.Errorf("aidial_file_alias must not have %q", name)
		}
	}
	for name := range file.Schema.Attributes {
		if _, ok := api.Schema.Attributes[name]; !ok {
			t.Errorf("aidial_alias lacks attribute %q present in aidial_file_alias", name)
		}
	}
	name := dataSourceIdentityFromSchema(t, "name", file)
	if !name.required || validate(t, name.validators, "") || !validate(t, name.validators, "a") {
		t.Error("name must be required, non-empty")
	}
}

func TestFileAliasDataSourceMissing(t *testing.T) {
	missing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	resp := readToolsetSource(t, &fileAliasDataSource{client: missing}, map[string]string{"name": "nope"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"nope"`) {
		t.Fatalf("a missing alias must be an error naming it, got %v", resp.Diagnostics)
	}
}
