package provider

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const translatorAPIBody = `{
	"name":"translators/platform/t",
	"baseUrl":"http://x:8000",
	"in":"openaiChatCompletions",
	"out":"anthropicMessages",
	"status":"valid"
}`

func TestTranslatorTypesRegistered(t *testing.T) {
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
	if !resources["aidial_translator"] {
		t.Error("aidial_translator resource is not registered")
	}
	for _, name := range []string{"aidial_translator", "aidial_file_translator"} {
		if !sources[name] {
			t.Errorf("%s data source is not registered", name)
		}
	}
}

func TestTranslatorDataSourcePopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(translatorAPIBody))
	})

	resp := readToolsetSource(t, &translatorDataSource{client: c}, map[string]string{"bucket": "platform", "path": "t"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/translators/platform/t" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	for name, want := range map[string]string{
		"id": "platform/t", "base_url": "http://x:8000", "in": "openaiChatCompletions", "out": "anthropicMessages",
	} {
		var got types.String
		resp.State.GetAttribute(context.Background(), pathRoot(name), &got)
		if got.ValueString() != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}
}

func TestTranslatorDataSourceMatchesResource(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(translatorAPIBody)) })

	resp := readToolsetSource(t, &translatorDataSource{client: c}, map[string]string{"bucket": "platform", "path": "t"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("data source: %v", resp.Diagnostics)
	}
	var fromSource translatorModel
	resp.State.Get(context.Background(), &fromSource)

	prior := newTranslatorPlan()
	prior.ID = types.StringValue("platform/t")
	_, fromResource := runTranslatorRead(t, &translatorResource{client: c}, prior)
	if fromSource != fromResource {
		t.Fatalf("data source and resource disagree:\n source   %+v\n resource %+v", fromSource, fromResource)
	}

	var res resource.SchemaResponse
	(&translatorResource{}).Schema(context.Background(), resource.SchemaRequest{}, &res)
	var api datasource.SchemaResponse
	(&translatorDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &api)
	for name := range res.Schema.Attributes {
		if _, ok := api.Schema.Attributes[name]; !ok {
			t.Errorf("data source lacks attribute %q of the resource", name)
		}
	}
}

func TestTranslatorDataSourceMissingAndAPIError(t *testing.T) {
	missing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	resp := readToolsetSource(t, &translatorDataSource{client: missing}, map[string]string{"bucket": "platform", "path": "nope"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"platform"`) || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"nope"`) {
		t.Fatalf("a missing translator must be an error naming bucket and path, got %v", resp.Diagnostics)
	}

	failing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})
	resp = readToolsetSource(t, &translatorDataSource{client: failing}, map[string]string{"bucket": "platform", "path": "x"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "500") || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "boom") {
		t.Fatalf("an API error must carry status and body, got %v", resp.Diagnostics)
	}
}

func TestFileTranslatorDataSourcePopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.EscapedPath()
		_, _ = w.Write([]byte(translatorAPIBody))
	})

	resp := readToolsetSource(t, &fileTranslatorDataSource{client: c}, map[string]string{"name": "a/b"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/admin/config/file/translators/a%2Fb" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	for name, want := range map[string]string{"name": "a/b", "base_url": "http://x:8000", "in": "openaiChatCompletions", "out": "anthropicMessages"} {
		var got types.String
		resp.State.GetAttribute(context.Background(), pathRoot(name), &got)
		if got.ValueString() != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}
}

func TestFileTranslatorDataSourceShape(t *testing.T) {
	var file, api datasource.SchemaResponse
	(&fileTranslatorDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &file)
	(&translatorDataSource{}).Schema(context.Background(), datasource.SchemaRequest{}, &api)
	if diags := file.Schema.ValidateImplementation(context.Background()); diags.HasError() {
		t.Fatalf("file data source schema: %v", diags)
	}
	for _, name := range []string{"id", "bucket", "path"} {
		if _, has := file.Schema.Attributes[name]; has {
			t.Errorf("aidial_file_translator must not have %q", name)
		}
	}
	for name := range file.Schema.Attributes {
		if _, ok := api.Schema.Attributes[name]; !ok && name != "name" {
			t.Errorf("aidial_translator lacks attribute %q present in aidial_file_translator", name)
		}
	}
	name := dataSourceIdentityFromSchema(t, "name", file)
	if !name.required || validate(t, name.validators, "") || !validate(t, name.validators, "a") {
		t.Error("name must be required, non-empty")
	}
}

func TestFileTranslatorDataSourceMissing(t *testing.T) {
	missing := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
	resp := readToolsetSource(t, &fileTranslatorDataSource{client: missing}, map[string]string{"name": "nope"})
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), `"nope"`) {
		t.Fatalf("a missing translator must be an error naming it, got %v", resp.Diagnostics)
	}
}
