package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/epam/terraform-provider-aidial/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestModelDataSourceRegistered(t *testing.T) {
	var found bool
	for _, factory := range (&AidialProvider{}).DataSources(context.Background()) {
		ds := factory()
		resp := &datasource.MetadataResponse{}
		ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "aidial"}, resp)
		if resp.TypeName == "aidial_model" {
			found = true
		}
	}
	if !found {
		t.Fatal("aidial_model data source is not registered")
	}
}

// Every computed attribute of aidial_file_model must also exist in
// aidial_model, so the two model data sources cannot drift apart.
func TestModelDataSourceCoversFileModelAttributes(t *testing.T) {
	ctx := context.Background()
	fileResp := &datasource.SchemaResponse{}
	(&fileModelDataSource{}).Schema(ctx, datasource.SchemaRequest{}, fileResp)
	modelResp := &datasource.SchemaResponse{}
	(&modelDataSource{}).Schema(ctx, datasource.SchemaRequest{}, modelResp)

	for name := range fileResp.Schema.Attributes {
		if _, ok := modelResp.Schema.Attributes[name]; !ok {
			t.Errorf("aidial_model data source lacks attribute %q present in aidial_file_model", name)
		}
	}
	for _, name := range []string{"id", "bucket", "path", "name"} {
		if _, ok := modelResp.Schema.Attributes[name]; !ok {
			t.Errorf("aidial_model data source lacks attribute %q", name)
		}
	}
}

// readModelDataSource runs Read for a config holding the given string
// arguments (all other attributes null) and returns the response.
func readModelDataSource(t *testing.T, c *client.Client, args map[string]string) *datasource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	d := &modelDataSource{client: c}

	schemaResp := &datasource.SchemaResponse{}
	d.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
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
	d.Read(ctx, req, resp)
	return resp
}

func modelDataSourceServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *client.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(handler))
	t.Cleanup(srv.Close)
	c, err := client.NewClient(srv.URL, "k")
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func stateString(t *testing.T, resp *datasource.ReadResponse, name string) types.String {
	t.Helper()
	var value types.String
	resp.State.GetAttribute(context.Background(), path.Root(name), &value)
	return value
}

func TestModelDataSourceDefaultsAndPopulatesState(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":              "gpt-4o",
			"type":              "chat",
			"status":            "APPLIED",
			"catalogSchemaId":   "https://dial.epam.com/catalog-schemas/model",
			"catalogProperties": map[string]any{"vendor": "Vendor"},
		})
	})

	resp := readModelDataSource(t, c, map[string]string{"name": "gpt-4o"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/models/platform/gpt-4o" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	for name, want := range map[string]string{
		"id":                "platform/gpt-4o",
		"bucket":            "platform",
		"path":              "gpt-4o",
		"name":              "gpt-4o",
		"api_name":          "gpt-4o",
		"type":              "chat",
		"catalog_schema_id": "https://dial.epam.com/catalog-schemas/model",
	} {
		if got := stateString(t, resp, name); got.ValueString() != want {
			t.Errorf("%s = %v, want %q", name, got, want)
		}
	}
}

func TestModelDataSourceOmittedCollectionsFollowResource(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"name":"m","type":"chat","status":"APPLIED"}`))
	})

	resp := readModelDataSource(t, c, map[string]string{"name": "m"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	var state modelResourceModel
	if diags := resp.State.Get(context.Background(), &state); diags.HasError() {
		t.Fatalf("read state: %v", diags)
	}
	if !state.Features.IsNull() || !state.Limits.IsNull() || !state.Pricing.IsNull() ||
		!state.Upstreams.IsNull() || !state.UserRoles.IsNull() || !state.Dependencies.IsNull() ||
		!state.DescriptionKeywords.IsNull() || !state.InputAttachmentTypes.IsNull() ||
		!state.Interceptors.IsNull() || !state.FieldsHashingOrder.IsNull() ||
		!state.CatalogProperties.IsNull() {
		t.Fatalf("omitted objects, lists and sets must be null, got %+v", state)
	}
	// setModelState maps omitted maps to empty ones; the data source must
	// agree with the resource.
	if state.Interfaces.IsNull() || len(state.Interfaces.Elements()) != 0 ||
		state.DefaultHeaders.IsNull() || len(state.DefaultHeaders.Elements()) != 0 {
		t.Fatalf("omitted maps must be empty like the resource, got %v / %v", state.Interfaces, state.DefaultHeaders)
	}
}

func TestModelDataSourceExplicitBucketAndPath(t *testing.T) {
	var gotPath string
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`{"name":"m","type":"chat","status":"APPLIED"}`))
	})

	resp := readModelDataSource(t, c, map[string]string{"name": "m", "bucket": "team", "path": "folder/m"})
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics)
	}
	if gotPath != "/v1/models/team/folder/m" {
		t.Fatalf("unexpected request path %q", gotPath)
	}
	if got := stateString(t, resp, "id"); got.ValueString() != "team/folder/m" {
		t.Fatalf("unexpected id %v", got)
	}
}

func TestModelDataSourceNotFound(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	resp := readModelDataSource(t, c, map[string]string{"name": "missing"})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected a not-found diagnostic")
	}
	got := resp.Diagnostics[0]
	if got.Summary() != "Model not found" || !strings.Contains(got.Detail(), "platform/missing") {
		t.Fatalf("unexpected diagnostic %q: %q", got.Summary(), got.Detail())
	}
}

func TestModelDataSourceAPIError(t *testing.T) {
	c := modelDataSourceServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = io.WriteString(w, "boom")
	})

	resp := readModelDataSource(t, c, map[string]string{"name": "m"})
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected an API error diagnostic")
	}
	got := resp.Diagnostics[0]
	if got.Summary() != "Failed to read model" || !strings.Contains(got.Detail(), "500") {
		t.Fatalf("unexpected diagnostic %q: %q", got.Summary(), got.Detail())
	}
}
